package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
	"github.com/BwCloudWeGo/bw-cli/pkg/middleware"
)

// Account 是预置的演示账号（角色在服务端判定）。
type Account struct {
	Account     string
	Password    string
	Role        string
	DisplayName string
}

// Config 是 fleetlog 专属配置，与全局配置解耦，避免修改 pkg/config 公共结构。
type Config struct {
	TimeoutSeconds int
	AuthSecret     string
	Issuer         string
	Accounts       []Account
	VehicleTokens  map[string]string
}

// DefaultConfig 返回教学默认配置：超时 60 秒，签名 issuer 固定。
func DefaultConfig() Config {
	return Config{
		TimeoutSeconds: 60,
		Issuer:         "fleetlog",
		Accounts: []Account{
			{Account: "rd_wang", Password: "rd123456", Role: string(entity.RoleRequester), DisplayName: "研发-王工"},
			{Account: "lead_li", Password: "lead123456", Role: string(entity.RoleApprover), DisplayName: "负责人-李工"},
			{Account: "ops_zhao", Password: "ops123456", Role: string(entity.RoleOperator), DisplayName: "运营-赵工"},
		},
	}
}

func (c Config) normalized() Config {
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 60
	}
	if strings.TrimSpace(c.Issuer) == "" {
		c.Issuer = "fleetlog"
	}
	if c.Accounts == nil {
		c.Accounts = DefaultConfig().Accounts
	}
	if c.VehicleTokens == nil {
		c.VehicleTokens = map[string]string{}
	}
	return c
}

// Service 编排 fleetlog 用例，不感知 gorm、gin 与 gRPC。
type Service struct {
	repo     entity.Repository
	clock    entity.Clock
	log      *zap.Logger
	cfg      Config
	timeout  time.Duration
	accounts map[string]Account
}

const sessionTTLSeconds int64 = 7200

// NewService 创建 fleetlog 用例服务。
func NewService(repo entity.Repository, clock entity.Clock, cfg Config, log *zap.Logger) (*Service, error) {
	if repo == nil {
		return nil, errors.New("fleetlog: repository is required")
	}
	if clock == nil {
		clock = entity.SystemClock{}
	}
	if log == nil {
		log = zap.NewNop()
	}
	cfg = cfg.normalized()
	accounts := make(map[string]Account, len(cfg.Accounts))
	for _, item := range cfg.Accounts {
		accounts[item.Account] = item
	}
	return &Service{
		repo:     repo,
		clock:    clock,
		log:      log,
		cfg:      cfg,
		timeout:  time.Duration(cfg.TimeoutSeconds) * time.Second,
		accounts: accounts,
	}, nil
}

// TimeoutSeconds 返回当前教学时钟的超时阈值（秒）。
func (s *Service) TimeoutSeconds() int {
	return s.cfg.TimeoutSeconds
}

// Actor 依据签名令牌中的用户与角色还原操作人，名称取服务端预置信息。
func (s *Service) Actor(userID string, role string) entity.Actor {
	actor := entity.Actor{ID: userID, Role: entity.Role(role), Name: userID}
	if account, ok := s.accounts[userID]; ok {
		actor.Name = account.DisplayName
		if actor.Role == "" {
			actor.Role = entity.Role(account.Role)
		}
	}
	return actor
}

// VehicleByToken 将车辆令牌映射为车辆编号，未命中返回 false。
func (s *Service) VehicleByToken(token string) (string, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	vehicleID, ok := s.cfg.VehicleTokens[token]
	return vehicleID, ok
}

// Login 校验预置账号并签发会话令牌。
func (s *Service) Login(ctx context.Context, cmd dto.LoginCommand) (*dto.SessionDTO, error) {
	account, ok := s.accounts[strings.TrimSpace(cmd.Account)]
	if !ok || subtle.ConstantTimeCompare([]byte(account.Password), []byte(cmd.Password)) != 1 {
		return nil, apperrors.Unauthorized("fleetlog_login_failed", "账号或密码不正确")
	}
	if strings.TrimSpace(s.cfg.AuthSecret) == "" {
		return nil, apperrors.Internal("fleetlog_auth_misconfigured", "未配置 fleetlog 会话签名密钥")
	}
	token, err := middleware.GenerateToken(middleware.JWTConfig{
		Secret:        s.cfg.AuthSecret,
		Issuer:        s.cfg.Issuer,
		ExpireSeconds: sessionTTLSeconds,
	}, middleware.JWTClaims{UserID: account.Account, Role: account.Role})
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "fleetlog_login_failed", "签发会话令牌失败", err)
	}
	return &dto.SessionDTO{
		Token:          token,
		UserID:         account.Account,
		Role:           account.Role,
		DisplayName:    account.DisplayName,
		TimeoutSeconds: s.cfg.TimeoutSeconds,
	}, nil
}

func requireRole(actor entity.Actor, role entity.Role) error {
	if !actor.Is(role) {
		return entity.ErrForbidden
	}
	return nil
}

func requireAnyRole(actor entity.Actor, roles ...entity.Role) error {
	for _, role := range roles {
		if actor.Is(role) {
			return nil
		}
	}
	return entity.ErrForbidden
}

func (s *Service) audit(ctx context.Context, repo entity.Repository, actor entity.Actor, action string, targetType string, targetID string, detail string) error {
	return repo.SaveAudit(ctx, entity.NewAudit(actor, action, targetType, targetID, detail, s.clock.Now()))
}

func (s *Service) now() time.Time {
	return s.clock.Now()
}

func latestByVehicle(releases []*entity.Release) map[string]*entity.Release {
	output := make(map[string]*entity.Release, len(releases))
	for _, release := range releases {
		current, ok := output[release.VehicleID]
		if !ok || release.Version > current.Version {
			output[release.VehicleID] = release
		}
	}
	return output
}

func baselineByVehicle(baselines []*entity.Baseline) map[string]*entity.Baseline {
	output := make(map[string]*entity.Baseline, len(baselines))
	for _, baseline := range baselines {
		output[baseline.VehicleID] = baseline
	}
	return output
}

func releaseStatus(release *entity.Release) string {
	if release == nil {
		return entity.StatusNotReleased
	}
	return release.Status
}
