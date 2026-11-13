package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testNow() time.Time {
	return time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
}

func requesterActor() Actor {
	return Actor{ID: "rd_wang", Name: "研发-王工", Role: RoleRequester}
}

func approverActor() Actor {
	return Actor{ID: "lead_li", Name: "负责人-李工", Role: RoleApprover}
}

func TestValidateReasonAndInterval(t *testing.T) {
	require.ErrorIs(t, ValidateReason("   "), ErrReasonRequired)
	require.ErrorIs(t, ValidateReason(string(make([]rune, ReasonMaxLen+1))), ErrReasonTooLong)
	require.NoError(t, ValidateReason(string(make([]rune, ReasonMaxLen))))
	require.ErrorIs(t, ValidateInterval(IntervalMin-1), ErrIntervalOutOfRange)
	require.ErrorIs(t, ValidateInterval(IntervalMax+1), ErrIntervalOutOfRange)
	require.NoError(t, ValidateInterval(IntervalMin))
	require.NoError(t, ValidateInterval(IntervalMax))
}

func TestApplicationStateMachine(t *testing.T) {
	now := testNow()
	actor := requesterActor()

	app := NewApplication(actor, "  排查丢帧  ", 10, now)
	require.Equal(t, AppStatusDraft, app.Status)
	require.Equal(t, "排查丢帧", app.Reason)
	require.Equal(t, 1, app.VersionNo)
	require.NoError(t, app.ValidateFields())

	require.ErrorIs(t, app.Edit(approverActor(), "x", 10, now), ErrNotRequester)
	require.ErrorIs(t, app.Edit(actor, "x", 5, now), ErrIntervalOutOfRange)
	require.ErrorIs(t, app.Edit(actor, "x", 301, now), ErrIntervalOutOfRange)
	require.NoError(t, app.Edit(actor, "", 0, now))
	require.Equal(t, 0, app.Interval)

	require.ErrorIs(t, app.Submit(now), ErrReasonRequired)
	require.NoError(t, app.Edit(actor, "排查丢帧", 0, now))
	require.ErrorIs(t, app.Submit(now), ErrIntervalOutOfRange)
	require.NoError(t, app.Edit(actor, "排查丢帧", 10, now))
	require.NoError(t, app.Submit(now))
	require.Equal(t, AppStatusPending, app.Status)
	require.NotNil(t, app.SubmittedAt)
	require.ErrorIs(t, app.Submit(now), ErrApplicationNotSubmittable)
	require.ErrorIs(t, app.Edit(actor, "改", 10, now), ErrApplicationNotEditable)

	require.ErrorIs(t, app.Approve(actor, now), ErrSelfApproval)
	require.NoError(t, app.Approve(approverActor(), now))
	require.Equal(t, AppStatusApproved, app.Status)
	require.NotNil(t, app.DecidedAt)
	require.ErrorIs(t, app.Approve(approverActor(), now), ErrApplicationNotPending)
	require.ErrorIs(t, app.Reject(approverActor(), "原因", now), ErrApplicationNotPending)
	require.ErrorIs(t, app.Submit(now), ErrApplicationNotSubmittable)

	require.NoError(t, app.Void("activity-1", now))
	require.Equal(t, AppStatusVoided, app.Status)
	require.Equal(t, "activity-1", app.ActivityID)
	require.ErrorIs(t, app.Void("activity-2", now), ErrApplicationNotApproved)

	second := NewApplication(actor, "二次排查", 20, now)
	require.NoError(t, second.Submit(now))
	require.ErrorIs(t, second.Reject(approverActor(), "", now), ErrReasonRequired)
	require.NoError(t, second.Reject(approverActor(), "间隔过小", now))
	require.Equal(t, AppStatusRejected, second.Status)
	require.Equal(t, "间隔过小", second.RejectReason)

	snapshot := second.Snapshot(now)
	require.Equal(t, 1, snapshot.VersionNo)
	require.Equal(t, AppStatusRejected, snapshot.Status)
	require.Equal(t, "间隔过小", snapshot.RejectReason)

	require.NoError(t, second.Submit(now))
	require.Equal(t, 2, second.VersionNo)
	require.Empty(t, second.RejectReason)
	require.Nil(t, second.DecidedAt)
}

func TestActivityStateMachine(t *testing.T) {
	now := testNow()
	app := &Application{ID: "app-1", Interval: 10}
	activity := NewActivity(app, []string{"V1", "V3"}, []string{"V2", "V4", "V5"}, requesterActor(), now)
	require.Equal(t, ActivityPilotPending, activity.Status)
	require.True(t, activity.IsOpen())
	require.True(t, activity.CanStop())

	require.ErrorIs(t, activity.MarkPilotPassed(now), ErrConflict)
	require.ErrorIs(t, activity.MarkPromoting(now), ErrConflict)
	require.ErrorIs(t, activity.MarkCompleted(approverActor(), now), ErrConflict)
	require.ErrorIs(t, activity.MarkRestored(approverActor(), now), ErrConflict)

	require.NoError(t, activity.MarkPiloting(now))
	require.Equal(t, ActivityPiloting, activity.Status)
	require.ErrorIs(t, activity.MarkPiloting(now), ErrConflict)
	require.NoError(t, activity.MarkPilotPassed(now))
	require.Equal(t, ActivityPilotPassed, activity.Status)
	require.ErrorIs(t, activity.MarkPilotPassed(now), ErrConflict)
	require.NoError(t, activity.MarkPromoting(now))
	require.ErrorIs(t, activity.MarkPromoting(now), ErrConflict)
	require.ErrorIs(t, activity.ApproveRestore(approverActor(), now), ErrConflict)

	stopped := *activity
	require.NoError(t, stopped.Stop(requesterActor(), `{"released":[]}`, now))
	require.Equal(t, ActivityStopped, stopped.Status)
	require.NotNil(t, stopped.StoppedAt)
	require.NoError(t, stopped.Stop(requesterActor(), "", now))
	require.ErrorIs(t, stopped.MarkRestoring(requesterActor(), now), ErrForbidden)
	require.NoError(t, stopped.ApproveRestore(approverActor(), now))
	require.True(t, stopped.RestoreApproved())
	require.NoError(t, stopped.ApproveRestore(approverActor(), now))
	require.NoError(t, stopped.MarkRestoring(requesterActor(), now))
	require.Equal(t, ActivityRestoring, stopped.Status)
	require.ErrorIs(t, stopped.MarkRestoring(requesterActor(), now), ErrConflict)
	require.ErrorIs(t, stopped.MarkCompleted(approverActor(), now), ErrConflict)
	require.NoError(t, stopped.MarkRestored(approverActor(), now))
	require.Equal(t, ActivityRestored, stopped.Status)
	require.False(t, stopped.IsOpen())
	require.ErrorIs(t, stopped.Stop(requesterActor(), "", now), ErrConflict)

	completed := Activity{Status: ActivityPromoting}
	require.NoError(t, completed.MarkCompleted(approverActor(), now))
	require.False(t, completed.IsOpen())
	require.False(t, completed.CanStop())

	pending := Activity{Status: ActivityPilotPending}
	require.True(t, pending.CanStop())
	piloting := Activity{Status: ActivityPiloting}
	require.True(t, piloting.CanStop())
	pilotPassed := Activity{Status: ActivityPilotPassed}
	require.True(t, pilotPassed.CanStop())
	stoppedActivity := Activity{Status: ActivityStopped}
	require.False(t, stoppedActivity.CanStop())
	restoringActivity := Activity{Status: ActivityRestoring}
	require.False(t, restoringActivity.CanStop())
}

func TestReleaseLifecycle(t *testing.T) {
	now := testNow()
	release := NewRelease("activity-1", "V1", 2, BatchPilot, 10, 60, now)
	require.Equal(t, ReleaseWaiting, release.Status)
	require.False(t, release.TimedOut(now.Add(59*time.Second), time.Minute))
	require.True(t, release.TimedOut(now.Add(60*time.Second), time.Minute))

	release.MarkSuccess(now.Add(time.Second))
	require.Equal(t, ReleaseSuccess, release.Status)
	require.NotNil(t, release.ReceiptAt)
	require.False(t, release.Late)
	require.False(t, release.TimedOut(now.Add(time.Hour), time.Minute))
	require.False(t, release.MarkTimeout(now.Add(time.Hour)))

	late := NewRelease("activity-1", "V1", 3, BatchPilot, 10, 60, now)
	require.True(t, late.MarkTimeout(now.Add(time.Minute)))
	require.Equal(t, ReleaseTimeout, late.Status)
	require.NotNil(t, late.TimeoutAt)
	late.MarkSuccess(now.Add(2 * time.Minute))
	require.Equal(t, ReleaseSuccess, late.Status)
	require.True(t, late.Late)

	failed := NewRelease("activity-1", "V1", 4, BatchPromotion, 10, 60, now)
	failed.MarkFailure("应用失败", now)
	require.Equal(t, ReleaseFailed, failed.Status)
	require.Equal(t, "应用失败", failed.FailureReason)

	failed.Duplicate()
	failed.Duplicate()
	require.Equal(t, 2, failed.DuplicateReceipts)
}

func TestRecordConstructors(t *testing.T) {
	now := testNow()
	release := NewRelease("activity-1", "V1", 2, BatchPilot, 10, 60, now)

	receipt := NewReceipt(release, ReceiptSuccess, "", now)
	require.Equal(t, release.ID, receipt.ReleaseID)
	require.Equal(t, "V1", receipt.VehicleID)

	rejected := NewRejectedReceipt("V2", 2, RejectCrossVehicle, "跨车", now)
	require.Equal(t, RejectCrossVehicle, rejected.Reason)
	require.Equal(t, 2, rejected.Version)

	event := NewTimeoutEvent(release, now)
	require.Equal(t, release.ID, event.ReleaseID)
	require.Nil(t, event.ResolvedAt)

	pull := NewPullRecord("V1", 2, 10, release.ID, now)
	require.Equal(t, 10, pull.Interval)

	audit := NewAudit(requesterActor(), "application_created", "application", "app-1", "创建", now)
	require.Equal(t, "R", audit.ActorRole)
	require.Equal(t, "application_created", audit.Action)

	require.True(t, requesterActor().Is(RoleRequester))
	require.False(t, requesterActor().Is(RoleApprover))
	require.False(t, SystemClock{}.Now().IsZero())
}
