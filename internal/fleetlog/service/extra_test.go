package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
)

func TestToAppErrorMappings(t *testing.T) {
	cases := []struct {
		err  error
		code string
	}{
		{entity.ErrNotFound, "fleetlog_not_found"},
		{entity.ErrForbidden, "fleetlog_forbidden"},
		{entity.ErrSelfApproval, "fleetlog_self_approval"},
		{entity.ErrReasonRequired, "fleetlog_reason_required"},
		{entity.ErrReasonTooLong, "fleetlog_reason_too_long"},
		{entity.ErrIntervalOutOfRange, "fleetlog_interval_out_of_range"},
		{entity.ErrNotRequester, "fleetlog_not_requester"},
		{entity.ErrApplicationNotEditable, "fleetlog_application_not_editable"},
		{entity.ErrApplicationNotSubmittable, "fleetlog_application_not_submittable"},
		{entity.ErrApplicationNotPending, "fleetlog_application_not_pending"},
		{entity.ErrApplicationNotApproved, "fleetlog_application_not_approved"},
		{entity.ErrOpenActivity, "fleetlog_open_activity"},
		{entity.ErrInvalidPilotSelection, "fleetlog_invalid_pilot"},
		{entity.ErrPilotNotPassed, "fleetlog_pilot_not_passed"},
		{entity.ErrUnconfirmedVehicles, "fleetlog_unconfirmed_vehicles"},
		{entity.ErrVehicleOffline, "fleetlog_vehicle_offline"},
		{entity.ErrInvalidInput, "fleetlog_invalid_input"},
		{entity.ErrConflict, "fleetlog_conflict"},
		{errors.New("boom"), "fleetlog_internal"},
	}
	for _, item := range cases {
		mapped := toAppError(item.err)
		appErr, ok := apperrors.As(mapped)
		require.True(t, ok)
		require.Equal(t, item.code, appErr.Code, "mapping %v", item.err)
	}
	require.Nil(t, toAppError(nil))
	existing := apperrors.Conflict("custom", "自定义")
	require.Same(t, existing, toAppError(existing))
}

func TestServiceDefaults(t *testing.T) {
	ctx := context.Background()
	_, err := NewService(nil, nil, Config{}, nil)
	require.Error(t, err)

	repo := newFakeRepo()
	svc, err := NewService(repo, nil, Config{}, nil)
	require.NoError(t, err)
	require.Equal(t, 60, svc.TimeoutSeconds())
	require.False(t, svc.now().IsZero())

	_, err = svc.Login(ctx, dto.LoginCommand{Account: "rd_wang", Password: "rd123456"})
	require.Error(t, err)
	_, err = svc.Login(ctx, dto.LoginCommand{Account: "nobody", Password: "x"})
	require.Error(t, err)
}

func TestAdditionalTransitionsAndErrors(t *testing.T) {
	svc, repo, _ := newTestEnv(t)
	ctx := context.Background()

	pendingApp, err := svc.CreateApplication(ctx, testRequester, dto.CreateApplicationCommand{Reason: "待审批申请", Interval: 10})
	require.NoError(t, err)
	pendingApp, err = svc.SubmitApplication(ctx, testRequester, pendingApp.ID)
	require.NoError(t, err)
	_, err = svc.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: pendingApp.ID, PilotVehicles: []string{"V1", "V2"}})
	require.Error(t, err)

	detail := mustActivity(t, svc, ctx, []string{"V1", "V3"})
	activityID := detail.Activity.ID

	_, err = svc.ConfirmPromotion(ctx, testOperator, activityID)
	require.Error(t, err)

	detail, err = svc.StartPilot(ctx, testOperator, activityID)
	require.NoError(t, err)
	_, err = svc.ConfirmPromotion(ctx, testOperator, activityID)
	require.Error(t, err)

	_, err = svc.UpdateApplication(ctx, testApprover, dto.UpdateApplicationCommand{ID: pendingApp.ID, Reason: "x", Interval: 10})
	require.Error(t, err)
	_, err = svc.SubmitApplication(ctx, testApprover, pendingApp.ID)
	require.Error(t, err)

	_, err = svc.SetVehicleOnline(ctx, testOperator, "V9", true)
	require.Error(t, err)
	_, err = svc.PullConfig(ctx, "V9", dto.PullConfigCommand{})
	require.Error(t, err)

	delete(repo.data.vehicles, "V1")
	_, err = svc.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
	require.Error(t, err)

	repo.data.vehicles["V1"] = &entity.Vehicle{
		ID: "V1", Online: true, EffectiveInterval: 60, EffectiveVersion: 1, CurrentVersion: 2,
	}

	audits, err := svc.ListAudits(ctx, "", 0)
	require.NoError(t, err)
	require.NotEmpty(t, audits)
	audits, err = svc.ListAudits(ctx, activityID, 10)
	require.NoError(t, err)
	require.NotEmpty(t, audits)

	current, err := svc.GetCurrentActivity(ctx, testOperator)
	require.NoError(t, err)
	require.NotNil(t, current)

	stopped, err := svc.StopActivity(ctx, testOperator, dto.StopActivityCommand{ID: activityID, Reason: ""})
	require.NoError(t, err)
	require.Equal(t, entity.ActivityStopped, stopped.Activity.Status)
	doubled, err := svc.ApproveRestore(ctx, testApprover, activityID)
	require.NoError(t, err)
	require.NotEmpty(t, doubled.Activity.RestoreApprovedAt)
	doubled, err = svc.ApproveRestore(ctx, testApprover, activityID)
	require.NoError(t, err)
	require.NotEmpty(t, doubled.Activity.RestoreApprovedAt)

	restoring, err := svc.StartRestore(ctx, testOperator, activityID)
	require.NoError(t, err)
	require.Equal(t, entity.ActivityRestoring, restoring.Activity.Status)
	require.Equal(t, 3, rowByID(restoring, "V1").Version)

	injected := entity.NewRelease(activityID, "V1", 4, entity.BatchPromotion, 10, 60, svc.now())
	require.NoError(t, repo.SaveRelease(ctx, injected))
	vehicle := repo.data.vehicles["V1"]
	vehicle.CurrentVersion = 4
	_, err = svc.MarkRestored(ctx, testApprover, activityID)
	require.Error(t, err)

	completed := repo.data.activities[activityID]
	completed.Status = entity.ActivityCompleted
	repo.data.activities[activityID] = completed
	_, err = svc.StopActivity(ctx, testOperator, dto.StopActivityCommand{ID: activityID})
	require.Error(t, err)
	_, err = svc.StartPilot(ctx, testOperator, activityID)
	require.Error(t, err)
}

func TestReadTimeTimeoutComputation(t *testing.T) {
	svc, _, clock := newTestEnv(t)
	ctx := context.Background()

	detail := mustActivity(t, svc, ctx, []string{"V1", "V3"})
	detail, err := svc.StartPilot(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)

	clock.Advance(61 * time.Second)
	detail, err = svc.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ReleaseTimeout, rowByID(detail, "V1").Status)
	require.Equal(t, entity.ReleaseTimeout, rowByID(detail, "V3").Status)
	require.Len(t, detail.Timeouts, 2)

	count, err := svc.SweepTimeouts(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
