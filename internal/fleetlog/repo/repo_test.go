package repo

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

func TestModelRoundTrips(t *testing.T) {
	now := time.Date(2026, 11, 13, 9, 8, 7, 0, time.UTC)

	vehicle := &entity.Vehicle{ID: "V1", Online: true, EffectiveInterval: 60, EffectiveVersion: 1, CurrentVersion: 2}
	require.Equal(t, vehicle, toVehicle(fromVehicle(vehicle)))

	submitted := now.Add(time.Minute)
	application := &entity.Application{
		ID: "app-1", VersionNo: 2, RequesterID: "rd_wang", RequesterName: "研发-王工",
		Reason: "排查丢帧", Interval: 10, Status: entity.AppStatusPending, RejectReason: "旧原因",
		DecidedBy: "lead_li", SubmittedAt: &submitted, CreatedAt: now, UpdatedAt: now,
	}
	require.Equal(t, application, toApplication(fromApplication(application)))

	activity := &entity.Activity{
		ID: "activity-1", ApplicationID: "app-1", Interval: 10,
		PilotVehicles: []string{"V1", "V3"}, PromotionVehicles: []string{"V2", "V4", "V5"},
		Status: entity.ActivityPiloting, CreatedBy: "ops_zhao", CreatedAt: now, UpdatedAt: now,
		StopScope: `{"released":[]}`,
	}
	require.Equal(t, activity, toActivity(fromActivity(activity)))

	release := &entity.Release{
		ID: "release-1", ActivityID: "activity-1", VehicleID: "V1", Version: 2, Batch: entity.BatchPilot,
		ExpectedInterval: 10, BaselineInterval: 60, Status: entity.ReleaseWaiting,
		DispatchedAt: now, CreatedAt: now, UpdatedAt: now, DuplicateReceipts: 1,
	}
	require.Equal(t, release, toRelease(fromRelease(release)))
	require.Len(t, toReleases([]releaseModel{*fromRelease(release)}), 1)
}

func TestListEncoding(t *testing.T) {
	require.Equal(t, "[]", encodeList(nil))
	require.Equal(t, []string{}, decodeList(""))
	require.Equal(t, []string{}, decodeList("not-json"))
	require.Equal(t, []string{"V1", "V3"}, decodeList(encodeList([]string{"V1", "V3"})))
}

func TestDuplicateDetection(t *testing.T) {
	require.False(t, isDuplicateKey(nil))
	require.True(t, isDuplicateKey(errors.New("Error 1062: Duplicate entry 'x'")))
	require.True(t, isDuplicateKey(gorm.ErrDuplicatedKey))
	require.False(t, isDuplicateKey(errors.New("other")))
}
