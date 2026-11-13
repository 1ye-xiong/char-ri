package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	fleetlogv1 "github.com/BwCloudWeGo/bw-cli/api/gen/fleetlog/v1"
	fleetentity "github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
	fleethandler "github.com/BwCloudWeGo/bw-cli/internal/fleetlog/handler"
	fleetrepo "github.com/BwCloudWeGo/bw-cli/internal/fleetlog/repo"
	fleetservice "github.com/BwCloudWeGo/bw-cli/internal/fleetlog/service"
	"github.com/BwCloudWeGo/bw-cli/pkg/config"
	"github.com/BwCloudWeGo/bw-cli/pkg/database"
	"github.com/BwCloudWeGo/bw-cli/pkg/grpcx"
	"github.com/BwCloudWeGo/bw-cli/pkg/logger"
)

const serviceName = "fleetlog-service"
const defaultGRPCPort = 9300
const configPath = "configs/config.yaml"

type fleetlogConfigFile struct {
	TimeoutSeconds int                     `mapstructure:"timeout_seconds"`
	AuthSecret     string                  `mapstructure:"auth_secret"`
	Issuer         string                  `mapstructure:"issuer"`
	Accounts       []fleetlogAccountFile   `mapstructure:"accounts"`
	Vehicles       []fleetlogVehicleConfig `mapstructure:"vehicles"`
}

type fleetlogAccountFile struct {
	Account     string `mapstructure:"account"`
	Password    string `mapstructure:"password"`
	Role        string `mapstructure:"role"`
	DisplayName string `mapstructure:"display_name"`
}

type fleetlogVehicleConfig struct {
	ID    string `mapstructure:"id"`
	Token string `mapstructure:"token"`
}

func main() {
	if err := config.InitGlobal(configPath); err != nil {
		panic(err)
	}
	cfg := config.MustGlobal()
	cfg.Log.Service = cfg.ServiceName("fleetlog")
	cfg.Log = logger.WithDailyFileName(cfg.Log, time.Now())

	log, err := logger.New(cfg.Log)
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	config.PrintSourceNotice(cfg, os.Stdout)

	fleetCfg, err := loadFleetlogConfig(configPath, log)
	if err != nil {
		log.Fatal("load fleetlog config failed", zap.Error(err))
	}

	db, err := database.Open(cfg.Database, cfg.MySQL, cfg.PostgreSQL, log)
	if err != nil {
		log.Fatal("open database failed", zap.Error(err))
	}
	if err := fleetrepo.AutoMigrate(db); err != nil {
		log.Fatal("migrate fleetlog database failed", zap.Error(err))
	}
	if err := fleetrepo.Seed(context.Background(), db); err != nil {
		log.Fatal("seed fleetlog database failed", zap.Error(err))
	}

	repo := fleetrepo.NewGormRepository(db, log)
	svc, err := fleetservice.NewService(repo, fleetentity.SystemClock{}, fleetCfg, log)
	if err != nil {
		log.Fatal("create fleetlog service failed", zap.Error(err))
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(grpcx.UnaryServerInterceptor(log)))
	fleetlogv1.RegisterFleetLogServiceServer(server, fleethandler.NewServer(svc, fleetCfg.AuthSecret, fleetCfg.Issuer, log))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sweepTimeouts(ctx, svc, log)

	port := cfg.ServicePort("fleetlog", defaultGRPCPort)
	addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[Service Start Failed]\n  service: %s\n  listen: %s\n  error: %v\n\n", serviceName, addr, err)
		log.Fatal("listen failed", zap.String("addr", addr), zap.Error(err))
	}

	printStartupSummary(cfg.App.Env, addr, port, fleetCfg.TimeoutSeconds)
	go shutdownOnSignal(server, cancel, log)
	if err := server.Serve(listener); err != nil {
		log.Fatal("service stopped unexpectedly", zap.Error(err))
	}
}

func loadFleetlogConfig(path string, log *zap.Logger) (fleetservice.Config, error) {
	defaults := fleetservice.DefaultConfig()
	file := fleetlogConfigFile{}
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fleetservice.Config{}, err
	}
	if err := v.UnmarshalKey("fleetlog", &file); err != nil {
		return fleetservice.Config{}, err
	}
	output := defaults
	if file.TimeoutSeconds > 0 {
		output.TimeoutSeconds = file.TimeoutSeconds
	}
	if strings.TrimSpace(file.Issuer) != "" {
		output.Issuer = file.Issuer
	}
	output.AuthSecret = strings.TrimSpace(file.AuthSecret)
	if len(file.Accounts) > 0 {
		output.Accounts = make([]fleetservice.Account, 0, len(file.Accounts))
		for _, item := range file.Accounts {
			output.Accounts = append(output.Accounts, fleetservice.Account{
				Account:     strings.TrimSpace(item.Account),
				Password:    item.Password,
				Role:        strings.TrimSpace(item.Role),
				DisplayName: item.DisplayName,
			})
		}
	}
	output.VehicleTokens = make(map[string]string, len(file.Vehicles))
	for _, item := range file.Vehicles {
		vehicleID := strings.TrimSpace(item.ID)
		token := strings.TrimSpace(item.Token)
		if vehicleID == "" || token == "" {
			continue
		}
		output.VehicleTokens[token] = vehicleID
	}
	if output.AuthSecret == "" {
		return fleetservice.Config{}, fmt.Errorf("fleetlog.auth_secret 不能为空")
	}
	if len(output.VehicleTokens) == 0 {
		log.Warn("fleetlog vehicles config is empty, vehicle APIs will reject all tokens")
	}
	return output, nil
}

func sweepTimeouts(ctx context.Context, svc *fleetservice.Service, log *zap.Logger) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := svc.SweepTimeouts(ctx)
			if err != nil {
				log.Warn("fleetlog timeout sweep failed", zap.Error(err))
				continue
			}
			if count > 0 {
				log.Info("fleetlog timeout marked", zap.Int("count", count))
			}
		}
	}
}

func printStartupSummary(env string, addr string, port int, timeoutSeconds int) {
	host := strings.Split(addr, ":")[0]
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	fmt.Fprintf(os.Stdout, "\n[Service Started]\n")
	fmt.Fprintf(os.Stdout, "  service: %s\n", serviceName)
	fmt.Fprintf(os.Stdout, "  env: %s\n", env)
	fmt.Fprintf(os.Stdout, "  listen: %s\n", addr)
	fmt.Fprintf(os.Stdout, "  grpc: %s:%d\n", host, port)
	fmt.Fprintf(os.Stdout, "  timeout: %d 秒\n", timeoutSeconds)
	fmt.Fprintf(os.Stdout, "  config: services.fleetlog.port / fleetlog.timeout_seconds\n\n")
}

func shutdownOnSignal(server *grpc.Server, cancel context.CancelFunc, log *zap.Logger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	cancel()
	log.Info("service shutting down", zap.String("service", serviceName))
	server.GracefulStop()
}
