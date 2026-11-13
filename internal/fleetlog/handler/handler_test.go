package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	fleetlogv1 "github.com/BwCloudWeGo/bw-cli/api/gen/fleetlog/v1"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/service"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
	"github.com/BwCloudWeGo/bw-cli/pkg/middleware"
)

const (
	testSecret = "unit-test-secret"
	testIssuer = "fleetlog-unit-test"
)

type stubRepo struct {
	entity.Repository
	applications []*entity.Application
	activity     *entity.Activity
	vehicle      *entity.Vehicle
	release      *entity.Release
	rejected     *entity.RejectedReceipt
	audit        *entity.Audit
}

func (s *stubRepo) ListApplications(context.Context) ([]*entity.Application, error) {
	return s.applications, nil
}

func (s *stubRepo) ListApplicationSnapshots(context.Context, string) ([]*entity.ApplicationSnapshot, error) {
	return nil, nil
}

func (s *stubRepo) ListActivities(context.Context) ([]*entity.Activity, error) {
	if s.activity == nil {
		return nil, nil
	}
	return []*entity.Activity{s.activity}, nil
}

func (s *stubRepo) ListVehicles(context.Context) ([]*entity.Vehicle, error) {
	if s.vehicle == nil {
		return nil, nil
	}
	return []*entity.Vehicle{s.vehicle}, nil
}

func (s *stubRepo) ListLatestReleasesByVehicle(context.Context) (map[string]*entity.Release, error) {
	output := map[string]*entity.Release{}
	if s.release != nil {
		output[s.release.VehicleID] = s.release
	}
	return output, nil
}

func (s *stubRepo) ListRejectedReceipts(context.Context, string, string) ([]*entity.RejectedReceipt, error) {
	if s.rejected == nil {
		return nil, nil
	}
	return []*entity.RejectedReceipt{s.rejected}, nil
}

func (s *stubRepo) ListAudits(context.Context, string, int) ([]*entity.Audit, error) {
	if s.audit == nil {
		return nil, nil
	}
	return []*entity.Audit{s.audit}, nil
}

func newTestServer(t *testing.T, repo *stubRepo) *Server {
	t.Helper()
	cfg := service.DefaultConfig()
	cfg.AuthSecret = testSecret
	cfg.Issuer = testIssuer
	cfg.Accounts = []service.Account{{Account: "ops_zhao", Password: "ops123456", Role: "O", DisplayName: "运维-赵工"}}
	cfg.VehicleTokens = map[string]string{"v1-token": "V1"}
	svc, err := service.NewService(repo, nil, cfg, zap.NewNop())
	require.NoError(t, err)
	return NewServer(svc, testSecret, testIssuer, zap.NewNop())
}

func actorContext(t *testing.T, userID string, role string) context.Context {
	t.Helper()
	token, err := middleware.GenerateToken(middleware.JWTConfig{
		Secret: testSecret, Issuer: testIssuer, ExpireSeconds: 60,
	}, middleware.JWTClaims{UserID: userID, Role: role})
	require.NoError(t, err)
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
}

func vehicleContext(token string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-vehicle-token", token))
}

func requireAppCode(t *testing.T, err error, code string) {
	t.Helper()
	appErr, ok := apperrors.As(err)
	require.True(t, ok, "expected AppError, got %v", err)
	require.Equal(t, code, appErr.Code)
}

func TestAuthActorBranches(t *testing.T) {
	server := newTestServer(t, &stubRepo{})

	_, err := server.auth.actor(context.Background())
	requireAppCode(t, err, "fleetlog_missing_token")

	_, err = server.auth.actor(metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Token abc")))
	requireAppCode(t, err, "fleetlog_missing_token")

	_, err = server.auth.actor(metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer broken")))
	requireAppCode(t, err, "fleetlog_invalid_token")

	actor, err := server.auth.actor(actorContext(t, "ops_zhao", "O"))
	require.NoError(t, err)
	require.Equal(t, "ops_zhao", actor.ID)
	require.Equal(t, entity.RoleOperator, actor.Role)
	require.Equal(t, "运维-赵工", actor.Name)

	noSecret := NewServer(mustService(t), "", testIssuer, zap.NewNop())
	_, err = noSecret.auth.actor(context.Background())
	requireAppCode(t, err, "fleetlog_auth_misconfigured")
}

func mustService(t *testing.T) *service.Service {
	t.Helper()
	cfg := service.DefaultConfig()
	cfg.AuthSecret = testSecret
	cfg.Issuer = testIssuer
	svc, err := service.NewService(&stubRepo{}, nil, cfg, zap.NewNop())
	require.NoError(t, err)
	return svc
}

func TestAuthVehicleBranches(t *testing.T) {
	server := newTestServer(t, &stubRepo{})

	_, err := server.auth.vehicle(context.Background())
	requireAppCode(t, err, "fleetlog_missing_vehicle_token")

	_, err = server.auth.vehicle(vehicleContext("bogus"))
	requireAppCode(t, err, "fleetlog_invalid_vehicle_token")

	vehicleID, err := server.auth.vehicle(vehicleContext("v1-token"))
	require.NoError(t, err)
	require.Equal(t, "V1", vehicleID)
}

func TestLoginRPC(t *testing.T) {
	server := newTestServer(t, &stubRepo{})

	response, err := server.Login(context.Background(), &fleetlogv1.LoginRequest{Account: "ops_zhao", Password: "ops123456"})
	require.NoError(t, err)
	require.NotEmpty(t, response.GetToken())
	require.Equal(t, "O", response.GetRole())
	require.Equal(t, "运维-赵工", response.GetDisplayName())
	require.Equal(t, int32(60), response.GetTimeoutSeconds())

	_, err = server.Login(context.Background(), &fleetlogv1.LoginRequest{Account: "ops_zhao", Password: "wrong"})
	requireAppCode(t, err, "fleetlog_login_failed")
}

func TestUpdateApplicationWithoutToken(t *testing.T) {
	server := newTestServer(t, &stubRepo{})
	_, err := server.UpdateApplication(context.Background(), &fleetlogv1.UpdateApplicationRequest{Id: "app-1"})
	requireAppCode(t, err, "fleetlog_missing_token")
}

func TestListRPCs(t *testing.T) {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	repo := &stubRepo{
		applications: []*entity.Application{{
			ID: "app-1", VersionNo: 1, RequesterID: "rd_wang", RequesterName: "研发-王工",
			Reason: "排查丢帧", Interval: 20, Status: entity.AppStatusPending, CreatedAt: now, UpdatedAt: now,
		}},
		activity: &entity.Activity{
			ID: "activity-1", ApplicationID: "app-1", Interval: 20,
			PilotVehicles: []string{"V1", "V3"}, PromotionVehicles: []string{"V2", "V4", "V5"},
			Status: entity.ActivityPiloting, CreatedBy: "ops_zhao", CreatedAt: now, UpdatedAt: now,
		},
		vehicle: &entity.Vehicle{ID: "V1", Online: true, EffectiveInterval: 10, EffectiveVersion: 2, CurrentVersion: 3},
		release: &entity.Release{
			ID: "release-1", ActivityID: "activity-1", VehicleID: "V1", Version: 3, Batch: entity.BatchPilot,
			ExpectedInterval: 20, BaselineInterval: 10, Status: entity.ReleaseWaiting,
			DispatchedAt: now, CreatedAt: now, UpdatedAt: now,
		},
		rejected: &entity.RejectedReceipt{
			ID: "rejected-1", ActivityID: "activity-1", VehicleID: "V1", Version: 3,
			Reason: entity.RejectCrossVehicle, Detail: "凭证车辆 V4 上报了 V1 的回执", ReportedAt: now,
		},
		audit: &entity.Audit{
			ActorID: "ops_zhao", ActorRole: "O", Action: "pilot_started", TargetType: "activity",
			TargetID: "activity-1", Detail: "试点下发", CreatedAt: now,
		},
	}
	server := newTestServer(t, repo)
	ctx := actorContext(t, "ops_zhao", "O")

	applications, err := server.ListApplications(ctx, &fleetlogv1.ListApplicationsRequest{})
	require.NoError(t, err)
	require.Len(t, applications.GetItems(), 1)
	require.Equal(t, int32(20), applications.GetItems()[0].GetIntervalSeconds())

	activities, err := server.ListActivities(ctx, &fleetlogv1.ListActivitiesRequest{})
	require.NoError(t, err)
	require.Len(t, activities.GetItems(), 1)
	require.Equal(t, []string{"V2", "V4", "V5"}, activities.GetItems()[0].GetPromotionVehicles())

	vehicles, err := server.ListVehicles(ctx, &fleetlogv1.ListVehiclesRequest{})
	require.NoError(t, err)
	require.Len(t, vehicles.GetItems(), 1)
	require.Equal(t, int32(3), vehicles.GetItems()[0].GetCurrentVersion())

	rejected, err := server.ListRejectedReceipts(ctx, &fleetlogv1.ListRejectedReceiptsRequest{})
	require.NoError(t, err)
	require.Len(t, rejected.GetItems(), 1)
	require.Equal(t, entity.RejectCrossVehicle, rejected.GetItems()[0].GetReason())

	audits, err := server.ListAudits(ctx, &fleetlogv1.ListAuditsRequest{})
	require.NoError(t, err)
	require.Len(t, audits.GetItems(), 1)
	require.Equal(t, "pilot_started", audits.GetItems()[0].GetAction())
}

func TestMappers(t *testing.T) {
	require.Nil(t, toApplicationItem(nil))
	require.Nil(t, toActivityItem(nil))
	require.Nil(t, toSummary(nil))
	require.Nil(t, toActions(nil))
	require.Equal(t, &fleetlogv1.ActivityDetailResponse{}, toActivityDetail(nil))

	detail := &dto.ActivityDetailDTO{
		Activity: &dto.ActivityDTO{
			ID: "activity-1", ApplicationID: "app-1", Interval: 20,
			PilotVehicles: []string{"V1", "V3"}, PromotionVehicles: []string{"V2", "V4", "V5"},
			Status: "piloting", CreatedBy: "ops_zhao", CreatedAt: "2026-11-13 10:00:00",
		},
		Application: &dto.ApplicationDTO{
			ID: "app-1", VersionNo: 1, Reason: "排查丢帧", Interval: 20, Status: "approved",
			History: []*dto.ApplicationVersionDTO{{
				VersionNo: 1, Reason: "旧版本", Interval: 10, Status: "rejected", RejectReason: "间隔过小",
			}},
		},
		Rows: []*dto.VehicleRowDTO{{
			VehicleID: "V1", Released: true, Batch: "pilot", Version: 3, BaselineInterval: 10,
			ExpectedInterval: 20, EffectiveInterval: 20, EffectiveVersion: 3, Status: "success",
			DispatchedAt: "2026-11-13 10:00:00", ReceiptAt: "2026-11-13 10:00:01", Late: true, DuplicateReceipts: 2,
		}},
		Summary:  &dto.SummaryDTO{Total: 5, Released: 2, NotReleased: 3, Success: 1, Failed: 0, Timeout: 0, Waiting: 1, RejectedReceipt: 2, Effective: 1},
		Rejected: []*dto.RejectedReceiptDTO{{ID: "rejected-1", VehicleID: "V1", Version: 3, Reason: "cross_vehicle", Detail: "凭证不符", ReportedAt: "2026-11-13 10:00:02"}},
		Timeouts: []*dto.TimeoutEventDTO{{VehicleID: "V4", Version: 3, TimedOutAt: "2026-11-13 10:01:00", ResolvedAt: "2026-11-13 10:01:30"}},
		Audits:   []*dto.AuditDTO{{ActorID: "ops_zhao", ActorRole: "O", Action: "pilot_started", TargetType: "activity", TargetID: "activity-1", Detail: "试点下发", CreatedAt: "2026-11-13 10:00:00"}},
		Actions: &dto.ActionsDTO{
			CanStartPilot: true, CanStop: true, PromoteBlockedReason: "试点未全部成功",
		},
		TimeoutSecs: 60, ServerTime: "2026-11-13 10:02:00",
	}
	response := toActivityDetail(detail)
	require.Equal(t, "activity-1", response.GetActivity().GetId())
	require.Equal(t, "app-1", response.GetApplication().GetId())
	require.Len(t, response.GetApplication().GetHistory(), 1)
	require.Len(t, response.GetRows(), 1)
	require.True(t, response.GetRows()[0].GetLate())
	require.Equal(t, int32(2), response.GetSummary().GetRejectedReceipts())
	require.Len(t, response.GetRejectedReceipts(), 1)
	require.Len(t, response.GetTimeoutEvents(), 1)
	require.Len(t, response.GetAudits(), 1)
	require.True(t, response.GetActions().GetCanStop())
	require.Equal(t, int32(60), response.GetTimeoutSeconds())

	vehicle := toVehicleItem(&dto.VehicleDTO{
		ID: "V1", Online: true, EffectiveInterval: 20, EffectiveVersion: 3, CurrentVersion: 4,
		ExpectedInterval: 20, LatestStatus: "success",
	})
	require.Equal(t, "V1", vehicle.GetId())
	require.Equal(t, int32(4), vehicle.GetCurrentVersion())
}
