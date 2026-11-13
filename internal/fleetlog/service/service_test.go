package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

var (
	testRequester = entity.Actor{ID: "rd_wang", Name: "研发-王工", Role: entity.RoleRequester}
	testApprover  = entity.Actor{ID: "lead_li", Name: "负责人-李工", Role: entity.RoleApprover}
	testOperator  = entity.Actor{ID: "ops_zhao", Name: "运营-赵工", Role: entity.RoleOperator}
)

func newTestEnv(t *testing.T) (*Service, *fakeRepo, *fakeClock) {
	t.Helper()
	repo := newFakeRepo()
	clock := &fakeClock{now: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	svc, err := NewService(repo, clock, testServiceConfig(), nil)
	require.NoError(t, err)
	return svc, repo, clock
}

func testServiceConfig() Config {
	cfg := DefaultConfig()
	cfg.AuthSecret = "test-secret"
	cfg.TimeoutSeconds = 60
	cfg.VehicleTokens = map[string]string{
		"v1-token": "V1", "v2-token": "V2", "v3-token": "V3", "v4-token": "V4", "v5-token": "V5",
	}
	return cfg
}

func mustApprovedApplication(t *testing.T, svc *Service, ctx context.Context) *dto.ApplicationDTO {
	t.Helper()
	app, err := svc.CreateApplication(ctx, testRequester, dto.CreateApplicationCommand{Reason: "排查电机控制器丢帧", Interval: 10})
	require.NoError(t, err)
	app, err = svc.SubmitApplication(ctx, testRequester, app.ID)
	require.NoError(t, err)
	app, err = svc.ApproveApplication(ctx, testApprover, app.ID)
	require.NoError(t, err)
	require.Equal(t, entity.AppStatusApproved, app.Status)
	return app
}

func mustActivity(t *testing.T, svc *Service, ctx context.Context, pilots []string) *dto.ActivityDetailDTO {
	t.Helper()
	app := mustApprovedApplication(t, svc, ctx)
	detail, err := svc.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: app.ID, PilotVehicles: pilots})
	require.NoError(t, err)
	return detail
}

func rowByID(detail *dto.ActivityDetailDTO, vehicleID string) *dto.VehicleRowDTO {
	for _, row := range detail.Rows {
		if row.VehicleID == vehicleID {
			return row
		}
	}
	return nil
}

func TestApplicationWorkflow(t *testing.T) {
	svc, repo, clock := newTestEnv(t)
	ctx := context.Background()

	app, err := svc.CreateApplication(ctx, testRequester, dto.CreateApplicationCommand{})
	require.NoError(t, err)
	require.Equal(t, entity.AppStatusDraft, app.Status)

	_, err = svc.SubmitApplication(ctx, testRequester, app.ID)
	require.Error(t, err)

	_, err = svc.UpdateApplication(ctx, testRequester, dto.UpdateApplicationCommand{ID: app.ID, Reason: "原因", Interval: 5})
	require.Error(t, err)
	_, err = svc.UpdateApplication(ctx, testRequester, dto.UpdateApplicationCommand{ID: app.ID, Reason: "原因", Interval: 301})
	require.Error(t, err)

	app, err = svc.UpdateApplication(ctx, testRequester, dto.UpdateApplicationCommand{ID: app.ID, Reason: "排查丢帧", Interval: 10})
	require.NoError(t, err)
	app, err = svc.SubmitApplication(ctx, testRequester, app.ID)
	require.NoError(t, err)
	require.Equal(t, entity.AppStatusPending, app.Status)
	require.Equal(t, 1, app.VersionNo)

	_, err = svc.ApproveApplication(ctx, testRequester, app.ID)
	require.Error(t, err)
	app, err = svc.ApproveApplication(ctx, testApprover, app.ID)
	require.NoError(t, err)
	require.Equal(t, entity.AppStatusApproved, app.Status)
	_, err = svc.ApproveApplication(ctx, testApprover, app.ID)
	require.Error(t, err)
	_, err = svc.UpdateApplication(ctx, testRequester, dto.UpdateApplicationCommand{ID: app.ID, Reason: "改参数", Interval: 20})
	require.Error(t, err)

	second, err := svc.CreateApplication(ctx, testRequester, dto.CreateApplicationCommand{Reason: "二次排查", Interval: 20})
	require.NoError(t, err)
	second, err = svc.SubmitApplication(ctx, testRequester, second.ID)
	require.NoError(t, err)
	_, err = svc.RejectApplication(ctx, testApprover, second.ID, "")
	require.Error(t, err)
	second, err = svc.RejectApplication(ctx, testApprover, second.ID, "间隔过小，影响日志带宽")
	require.NoError(t, err)
	require.Equal(t, entity.AppStatusRejected, second.Status)
	require.Equal(t, "间隔过小，影响日志带宽", second.RejectReason)
	require.Len(t, second.History, 0)

	second, err = svc.SubmitApplication(ctx, testRequester, second.ID)
	require.NoError(t, err)
	require.Equal(t, entity.AppStatusPending, second.Status)
	require.Equal(t, 2, second.VersionNo)
	require.Len(t, second.History, 1)
	require.Equal(t, entity.AppStatusRejected, second.History[0].Status)

	self := entity.NewApplication(entity.Actor{ID: "lead_li", Name: "负责人-李工", Role: entity.RoleRequester}, "自审申请", 30, clock.Now())
	self.Status = entity.AppStatusPending
	require.NoError(t, repo.SaveApplication(ctx, self))
	_, err = svc.ApproveApplication(ctx, testApprover, self.ID)
	require.Error(t, err)

	_, err = svc.CreateApplication(ctx, testOperator, dto.CreateApplicationCommand{Reason: "越权", Interval: 10})
	require.Error(t, err)

	requesterList, err := svc.ListApplications(ctx, testRequester)
	require.NoError(t, err)
	require.Len(t, requesterList, 2)
	approverList, err := svc.ListApplications(ctx, testApprover)
	require.NoError(t, err)
	require.Len(t, approverList, 3)
}

func TestLoginAndVehicleToken(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	ctx := context.Background()

	session, err := svc.Login(ctx, dto.LoginCommand{Account: "rd_wang", Password: "rd123456"})
	require.NoError(t, err)
	require.NotEmpty(t, session.Token)
	require.Equal(t, "R", session.Role)
	require.Equal(t, 60, session.TimeoutSeconds)

	_, err = svc.Login(ctx, dto.LoginCommand{Account: "rd_wang", Password: "wrong"})
	require.Error(t, err)
	_, err = svc.Login(ctx, dto.LoginCommand{Account: "nobody", Password: "x"})
	require.Error(t, err)

	vehicleID, ok := svc.VehicleByToken("v1-token")
	require.True(t, ok)
	require.Equal(t, "V1", vehicleID)
	_, ok = svc.VehicleByToken("bad-token")
	require.False(t, ok)

	actor := svc.Actor("rd_wang", "R")
	require.Equal(t, "研发-王工", actor.Name)
	require.True(t, actor.Is(entity.RoleRequester))
}

func TestPullConfig(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	ctx := context.Background()

	pull, err := svc.PullConfig(ctx, "V1", dto.PullConfigCommand{VehicleID: "V1"})
	require.NoError(t, err)
	require.Equal(t, 1, pull.Version)
	require.Equal(t, 60, pull.Interval)
	require.False(t, pull.HasRelease)

	_, err = svc.PullConfig(ctx, "V2", dto.PullConfigCommand{VehicleID: "V2"})
	require.Error(t, err)

	_, err = svc.PullConfig(ctx, "V1", dto.PullConfigCommand{VehicleID: "V2"})
	require.Error(t, err)

	_, err = svc.PullConfig(ctx, "", dto.PullConfigCommand{})
	require.Error(t, err)

	detail := mustActivity(t, svc, ctx, []string{"V1", "V3"})
	_, err = svc.StartPilot(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	pull, err = svc.PullConfig(ctx, "V1", dto.PullConfigCommand{})
	require.NoError(t, err)
	require.Equal(t, 2, pull.Version)
	require.Equal(t, 10, pull.Interval)
	require.True(t, pull.HasRelease)
}

func TestPilotPromotionAndCompletion(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	ctx := context.Background()

	app := mustApprovedApplication(t, svc, ctx)
	_, err := svc.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: app.ID, PilotVehicles: []string{"V1"}})
	require.Error(t, err)
	_, err = svc.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: app.ID, PilotVehicles: []string{"V1", "V1"}})
	require.Error(t, err)
	_, err = svc.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: app.ID, PilotVehicles: []string{"V1", "V9"}})
	require.Error(t, err)

	detail, err := svc.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: app.ID, PilotVehicles: []string{"V1", "V3"}})
	require.NoError(t, err)
	require.Equal(t, entity.ActivityPilotPending, detail.Activity.Status)
	require.Equal(t, []string{"V2", "V4", "V5"}, detail.Activity.PromotionVehicles)
	require.True(t, detail.Actions.CanStartPilot)
	require.True(t, detail.Actions.CanStop)
	require.False(t, detail.Actions.CanPromote)

	otherApp := mustApprovedApplication(t, svc, ctx)
	_, err = svc.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: otherApp.ID, PilotVehicles: []string{"V2", "V4"}})
	require.Error(t, err)

	detail, err = svc.StartPilot(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ActivityPiloting, detail.Activity.Status)
	require.True(t, rowByID(detail, "V1").Released)
	require.True(t, rowByID(detail, "V3").Released)
	require.False(t, rowByID(detail, "V2").Released)
	require.Equal(t, 2, rowByID(detail, "V1").Version)

	detail, err = svc.StartPilot(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, 2, rowByID(detail, "V1").Version)

	_, err = svc.ConfirmPromotion(ctx, testOperator, detail.Activity.ID)
	require.Error(t, err)

	result, err := svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	require.True(t, result.Accepted)
	require.Equal(t, 10, result.EffectiveInterval)
	detail, err = svc.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ActivityPiloting, detail.Activity.Status)
	require.False(t, detail.Actions.CanPromote)

	_, err = svc.ReportReceipt(ctx, "V3", dto.ReportReceiptCommand{VehicleID: "V3", Version: 2, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	detail, err = svc.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ActivityPilotPassed, detail.Activity.Status)
	require.True(t, detail.Actions.CanPromote)

	detail, err = svc.ConfirmPromotion(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ActivityPromoting, detail.Activity.Status)
	for _, vehicleID := range []string{"V2", "V4", "V5"} {
		require.True(t, rowByID(detail, vehicleID).Released)
		require.Equal(t, 2, rowByID(detail, vehicleID).Version)
	}
	require.Equal(t, 2, rowByID(detail, "V1").Version)

	_, err = svc.ConfirmPromotion(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)

	_, err = svc.MarkCompleted(ctx, testOperator, detail.Activity.ID)
	require.Error(t, err)
	_, err = svc.MarkCompleted(ctx, testApprover, detail.Activity.ID)
	require.Error(t, err)

	for _, vehicleID := range []string{"V2", "V4", "V5"} {
		_, err = svc.ReportReceipt(ctx, vehicleID, dto.ReportReceiptCommand{VehicleID: vehicleID, Version: 2, Result: entity.ReceiptSuccess})
		require.NoError(t, err)
	}
	detail, err = svc.GetActivityDetail(ctx, testApprover, detail.Activity.ID)
	require.NoError(t, err)
	require.True(t, detail.Actions.CanMarkCompleted)
	require.Equal(t, 5, detail.Summary.Success)
	require.Equal(t, 5, detail.Summary.Effective)

	detail, err = svc.MarkCompleted(ctx, testApprover, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ActivityCompleted, detail.Activity.Status)
	current, err := svc.GetCurrentActivity(ctx, testApprover)
	require.NoError(t, err)
	require.Nil(t, current)

	audits, err := svc.ListAudits(ctx, detail.Activity.ID, 100)
	require.NoError(t, err)
	require.NotEmpty(t, audits)
	found := false
	for _, audit := range audits {
		if audit.Action == "activity_completed" {
			found = true
		}
	}
	require.True(t, found)
}

func TestStopAndRestore(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	ctx := context.Background()

	detail := mustActivity(t, svc, ctx, []string{"V1", "V2"})
	detail, err := svc.StartPilot(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	_, err = svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
	require.NoError(t, err)

	detail, err = svc.StopActivity(ctx, testOperator, dto.StopActivityCommand{ID: detail.Activity.ID, Reason: "试点一台失败，先停止"})
	require.NoError(t, err)
	require.Equal(t, entity.ActivityStopped, detail.Activity.Status)
	require.True(t, strings.Contains(detail.Activity.StopScope, "V3"))
	require.True(t, strings.Contains(detail.Activity.StopScope, "V4"))
	require.True(t, strings.Contains(detail.Activity.StopScope, "V5"))
	require.Equal(t, testOperator.ID, detail.Activity.StoppedBy)
	require.False(t, rowByID(detail, "V3").Released)

	detail, err = svc.StopActivity(ctx, testApprover, dto.StopActivityCommand{ID: detail.Activity.ID})
	require.NoError(t, err)
	require.Equal(t, entity.ActivityStopped, detail.Activity.Status)

	_, err = svc.MarkCompleted(ctx, testApprover, detail.Activity.ID)
	require.Error(t, err)

	_, err = svc.ApproveRestore(ctx, testOperator, detail.Activity.ID)
	require.Error(t, err)
	_, err = svc.StartRestore(ctx, testOperator, detail.Activity.ID)
	require.Error(t, err)

	detail, err = svc.ApproveRestore(ctx, testApprover, detail.Activity.ID)
	require.NoError(t, err)
	require.NotEmpty(t, detail.Activity.RestoreApprovedAt)

	detail, err = svc.StartRestore(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ActivityRestoring, detail.Activity.Status)
	require.Equal(t, 3, rowByID(detail, "V1").Version)
	require.Equal(t, 60, rowByID(detail, "V1").ExpectedInterval)
	require.Equal(t, 3, rowByID(detail, "V2").Version)
	require.False(t, rowByID(detail, "V3").Released)

	detail, err = svc.StartRestore(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)

	_, err = svc.MarkRestored(ctx, testApprover, detail.Activity.ID)
	require.Error(t, err)
	_, err = svc.MarkRestored(ctx, testOperator, detail.Activity.ID)
	require.Error(t, err)

	for _, vehicleID := range []string{"V1", "V2"} {
		_, err = svc.ReportReceipt(ctx, vehicleID, dto.ReportReceiptCommand{VehicleID: vehicleID, Version: 3, Result: entity.ReceiptSuccess})
		require.NoError(t, err)
	}
	detail, err = svc.GetActivityDetail(ctx, testApprover, detail.Activity.ID)
	require.NoError(t, err)
	require.True(t, detail.Actions.CanMarkRestored)

	detail, err = svc.MarkRestored(ctx, testApprover, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ActivityRestored, detail.Activity.Status)
	require.Equal(t, 60, rowByID(detail, "V1").EffectiveInterval)
	require.Equal(t, 60, rowByID(detail, "V2").EffectiveInterval)
	require.Equal(t, entity.StatusNotReleased, rowByID(detail, "V3").Status)

	current, err := svc.GetCurrentActivity(ctx, testApprover)
	require.NoError(t, err)
	require.Nil(t, current)
}

func TestReceiptJudgement(t *testing.T) {
	svc, repo, clock := newTestEnv(t)
	ctx := context.Background()

	detail := mustActivity(t, svc, ctx, []string{"V1", "V3"})
	detail, err := svc.StartPilot(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	activityID := detail.Activity.ID

	result, err := svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V2", Version: 2, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	require.False(t, result.Accepted)
	require.Equal(t, entity.RejectCrossVehicle, result.Reason)

	result, err = svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 9, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	require.Equal(t, entity.RejectUnknownVersion, result.Reason)

	result, err = svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptFailure})
	require.NoError(t, err)
	require.Equal(t, entity.RejectInvalidPayload, result.Reason)

	result, err = svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 0, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	require.Equal(t, entity.RejectInvalidPayload, result.Reason)

	result, err = svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	require.True(t, result.Accepted)
	require.Equal(t, 10, result.EffectiveInterval)

	result, err = svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	require.False(t, result.Accepted)
	require.Equal(t, entity.RejectDuplicate, result.Reason)
	detail, err = svc.GetActivityDetail(ctx, testOperator, activityID)
	require.NoError(t, err)
	require.Equal(t, 1, rowByID(detail, "V1").DuplicateReceipts)

	result, err = svc.ReportReceipt(ctx, "V3", dto.ReportReceiptCommand{VehicleID: "V3", Version: 2, Result: entity.ReceiptFailure, FailureReason: "应用失败"})
	require.NoError(t, err)
	require.True(t, result.Accepted)
	require.Equal(t, entity.ReleaseFailed, result.ReleaseStatus)
	require.Equal(t, 60, result.EffectiveInterval)

	second := entity.NewRelease(activityID, "V1", 3, entity.BatchRestore, 60, 10, clock.Now())
	require.NoError(t, repo.SaveRelease(ctx, second))
	vehicle := repo.data.vehicles["V1"]
	vehicle.CurrentVersion = 3

	result, err = svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	require.Equal(t, entity.RejectStaleVersion, result.Reason)

	rejected, err := svc.ListRejectedReceipts(ctx, "", "")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rejected), 5)
	filtered, err := svc.ListRejectedReceipts(ctx, "", "V2")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
}

func TestTimeoutAndLateReceipt(t *testing.T) {
	svc, _, clock := newTestEnv(t)
	ctx := context.Background()

	detail := mustActivity(t, svc, ctx, []string{"V1", "V3"})
	detail, err := svc.StartPilot(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	activityID := detail.Activity.ID

	clock.Advance(61 * time.Second)
	count, err := svc.SweepTimeouts(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	detail, err = svc.GetActivityDetail(ctx, testOperator, activityID)
	require.NoError(t, err)
	require.Equal(t, entity.ReleaseTimeout, rowByID(detail, "V1").Status)
	require.Equal(t, entity.ReleaseTimeout, rowByID(detail, "V3").Status)
	require.Len(t, detail.Timeouts, 2)
	require.Equal(t, 2, detail.Summary.Timeout)
	require.Equal(t, entity.ActivityPiloting, detail.Activity.Status)

	count, err = svc.SweepTimeouts(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	result, err := svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	require.True(t, result.Accepted)

	detail, err = svc.GetActivityDetail(ctx, testOperator, activityID)
	require.NoError(t, err)
	row := rowByID(detail, "V1")
	require.Equal(t, entity.ReleaseSuccess, row.Status)
	require.True(t, row.Late)
	require.NotEmpty(t, row.TimeoutAt)
	require.Equal(t, 10, row.EffectiveInterval)
	detail, err = svc.GetActivityDetail(ctx, testOperator, activityID)
	require.NoError(t, err)
	resolved := false
	for _, event := range detail.Timeouts {
		if event.VehicleID == "V1" && event.ResolvedAt != "" {
			resolved = true
		}
	}
	require.True(t, resolved)

	_, err = svc.ReportReceipt(ctx, "V3", dto.ReportReceiptCommand{VehicleID: "V3", Version: 2, Result: entity.ReceiptFailure, FailureReason: "版本写入失败"})
	require.NoError(t, err)
	detail, err = svc.GetActivityDetail(ctx, testOperator, activityID)
	require.NoError(t, err)
	require.Equal(t, entity.ReleaseFailed, rowByID(detail, "V3").Status)
	require.Equal(t, entity.ActivityPiloting, detail.Activity.Status)
	require.Len(t, detail.Timeouts, 2)
}

func TestConcurrentStartPilotIsIdempotent(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	ctx := context.Background()

	detail := mustActivity(t, svc, ctx, []string{"V1", "V3"})
	var wait sync.WaitGroup
	for i := 0; i < 4; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _ = svc.StartPilot(ctx, testOperator, detail.Activity.ID)
		}()
	}
	wait.Wait()

	detail, err := svc.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	released := 0
	for _, row := range detail.Rows {
		if row.Released {
			released++
			require.Equal(t, 2, row.Version)
		}
	}
	require.Equal(t, 2, released)
	require.Equal(t, entity.ActivityPiloting, detail.Activity.Status)
}

func TestVehicleOnlineAndQueries(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	ctx := context.Background()

	_, err := svc.SetVehicleOnline(ctx, testRequester, "V2", true)
	require.Error(t, err)
	vehicle, err := svc.SetVehicleOnline(ctx, testOperator, "V2", true)
	require.NoError(t, err)
	require.True(t, vehicle.Online)

	vehicles, err := svc.ListVehicles(ctx)
	require.NoError(t, err)
	require.Len(t, vehicles, 5)

	detail := mustActivity(t, svc, ctx, []string{"V1", "V3"})
	_, err = svc.StartPilot(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	_, err = svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
	require.NoError(t, err)
	_, err = svc.ReportReceipt(ctx, "V3", dto.ReportReceiptCommand{VehicleID: "V3", Version: 2, Result: entity.ReceiptFailure, FailureReason: "配置解析失败"})
	require.NoError(t, err)

	detail, err = svc.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, 2, detail.Summary.Released)
	require.Equal(t, 1, detail.Summary.Success)
	require.Equal(t, 1, detail.Summary.Failed)
	require.Equal(t, 3, detail.Summary.NotReleased)
	require.Contains(t, detail.Actions.PromoteBlockedReason, "V3=failed")
	require.False(t, detail.Actions.CanPromote)

	activities, err := svc.ListActivities(ctx)
	require.NoError(t, err)
	require.Len(t, activities, 1)

	_, err = svc.GetActivityDetail(ctx, testOperator, "missing")
	require.Error(t, err)
}
