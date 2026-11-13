package dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

func TestFormatTime(t *testing.T) {
	require.Equal(t, "", FormatTime(time.Time{}))
	require.Equal(t, "", FormatTimePtr(nil))

	value := time.Date(2026, 11, 13, 9, 8, 7, 0, time.UTC)
	require.NotEmpty(t, FormatTime(value))
	require.Equal(t, FormatTime(value), FormatTimePtr(&value))
}

func TestConverters(t *testing.T) {
	now := time.Date(2026, 11, 13, 9, 8, 7, 0, time.UTC)
	submitted := now.Add(time.Minute)
	decided := now.Add(2 * time.Minute)

	application := &entity.Application{
		ID: "app-1", VersionNo: 2, RequesterID: "rd_wang", RequesterName: "研发-王工",
		Reason: "排查丢帧", Interval: 10, Status: entity.AppStatusRejected, RejectReason: "间隔过小",
		DecidedBy: "lead_li", SubmittedAt: &submitted, DecidedAt: &decided, CreatedAt: now,
		ActivityID: "activity-1",
	}
	history := []*entity.ApplicationSnapshot{{
		VersionNo: 1, Reason: "旧原因", Interval: 20, Status: entity.AppStatusRejected,
		RejectReason: "旧退回", DecidedBy: "lead_li", CreatedAt: now,
	}}
	appView := NewApplicationDTO(application, history, true, true, false)
	require.Equal(t, "app-1", appView.ID)
	require.Equal(t, 2, appView.VersionNo)
	require.Len(t, appView.History, 1)
	require.Equal(t, 1, appView.History[0].VersionNo)
	require.True(t, appView.CanEdit)
	require.False(t, appView.CanDecide)

	activity := &entity.Activity{
		ID: "activity-1", ApplicationID: "app-1", Interval: 10,
		PilotVehicles: []string{"V1", "V3"}, PromotionVehicles: []string{"V2", "V4", "V5"},
		Status: entity.ActivityPiloting, CreatedBy: "ops_zhao", CreatedAt: now,
	}
	activityView := NewActivityDTO(activity)
	require.Equal(t, []string{"V1", "V3"}, activityView.PilotVehicles)
	activityView.PilotVehicles[0] = "V9"
	require.Equal(t, "V1", activity.PilotVehicles[0])

	vehicle := &entity.Vehicle{ID: "V1", Online: true, EffectiveInterval: 60, EffectiveVersion: 1, CurrentVersion: 2}
	release := &entity.Release{Version: 2, ExpectedInterval: 10, Status: entity.ReleaseWaiting}
	vehicleView := NewVehicleDTO(vehicle, release)
	require.Equal(t, 10, vehicleView.ExpectedInterval)
	require.Equal(t, entity.ReleaseWaiting, vehicleView.LatestStatus)
	require.Equal(t, 60, NewVehicleDTO(vehicle, nil).ExpectedInterval)
	require.Equal(t, "", NewVehicleDTO(vehicle, nil).LatestStatus)

	rejected := NewRejectedReceiptDTO(&entity.RejectedReceipt{
		ID: "r-1", VehicleID: "V2", Version: 2, Reason: entity.RejectCrossVehicle, Detail: "跨车", ReportedAt: now,
	})
	require.Equal(t, entity.RejectCrossVehicle, rejected.Reason)

	event := NewTimeoutEventDTO(&entity.TimeoutEvent{VehicleID: "V1", Version: 2, TimedOutAt: now})
	require.Equal(t, "", event.ResolvedAt)

	audit := NewAuditDTO(&entity.Audit{ActorID: "ops_zhao", ActorRole: "O", Action: "pilot_started", CreatedAt: now})
	require.Equal(t, "pilot_started", audit.Action)
}
