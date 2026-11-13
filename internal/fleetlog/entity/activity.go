package entity

import (
	"time"

	"github.com/google/uuid"
)

// Activity 是一次分批发布活动的聚合根。
type Activity struct {
	ID                string
	ApplicationID     string
	Interval          int
	PilotVehicles     []string
	PromotionVehicles []string
	Status            string
	CreatedBy         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	StoppedBy         string
	StoppedAt         *time.Time
	StopScope         string
	RestoreApprovedBy string
	RestoreApprovedAt *time.Time
	RestoreStartedBy  string
	RestoreStartedAt  *time.Time
	CompletedBy       string
	CompletedAt       *time.Time
	RestoredBy        string
	RestoredAt        *time.Time
}

// NewActivity 由已批准申请创建待试点活动。
func NewActivity(application *Application, pilots []string, promotion []string, actor Actor, now time.Time) *Activity {
	return &Activity{
		ID:                uuid.NewString(),
		ApplicationID:     application.ID,
		Interval:          application.Interval,
		PilotVehicles:     pilots,
		PromotionVehicles: promotion,
		Status:            ActivityPilotPending,
		CreatedBy:         actor.ID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// IsOpen 判断活动是否仍未结束（completed/restored 为结束）。
func (a *Activity) IsOpen() bool {
	return a.Status != ActivityCompleted && a.Status != ActivityRestored
}

// CanStop 判断当前状态是否允许停止。
func (a *Activity) CanStop() bool {
	switch a.Status {
	case ActivityPilotPending, ActivityPiloting, ActivityPilotPassed, ActivityPromoting:
		return true
	default:
		return false
	}
}

// MarkPiloting 将活动推进到试点中。
func (a *Activity) MarkPiloting(now time.Time) error {
	if a.Status != ActivityPilotPending {
		return ErrConflict
	}
	a.Status = ActivityPiloting
	a.UpdatedAt = now
	return nil
}

// MarkPilotPassed 在 2 台试点车当前版本均成功时推进到试点通过。
func (a *Activity) MarkPilotPassed(now time.Time) error {
	if a.Status != ActivityPiloting {
		return ErrConflict
	}
	a.Status = ActivityPilotPassed
	a.UpdatedAt = now
	return nil
}

// MarkPromoting 推进到推广中。
func (a *Activity) MarkPromoting(now time.Time) error {
	if a.Status != ActivityPilotPassed {
		return ErrConflict
	}
	a.Status = ActivityPromoting
	a.UpdatedAt = now
	return nil
}

// Stop 停止活动的未发布部分，重复停止幂等。
func (a *Activity) Stop(actor Actor, scope string, now time.Time) error {
	if a.Status == ActivityStopped {
		return nil
	}
	if !a.CanStop() {
		return ErrConflict
	}
	stopped := now
	a.Status = ActivityStopped
	a.StoppedBy = actor.ID
	a.StoppedAt = &stopped
	a.StopScope = scope
	a.UpdatedAt = now
	return nil
}

// ApproveRestore 由负责人批准恢复，重复批准幂等。
func (a *Activity) ApproveRestore(actor Actor, now time.Time) error {
	if a.Status != ActivityStopped {
		return ErrConflict
	}
	if a.RestoreApprovedAt != nil {
		return nil
	}
	approved := now
	a.RestoreApprovedBy = actor.ID
	a.RestoreApprovedAt = &approved
	a.UpdatedAt = now
	return nil
}

// RestoreApproved 判断是否已批准恢复。
func (a *Activity) RestoreApproved() bool {
	return a.RestoreApprovedAt != nil
}

// MarkRestoring 由运营发起恢复下发。
func (a *Activity) MarkRestoring(actor Actor, now time.Time) error {
	if a.Status != ActivityStopped {
		return ErrConflict
	}
	if !a.RestoreApproved() {
		return ErrForbidden
	}
	started := now
	a.Status = ActivityRestoring
	a.RestoreStartedBy = actor.ID
	a.RestoreStartedAt = &started
	a.UpdatedAt = now
	return nil
}

// MarkCompleted 由负责人标记活动完成。
func (a *Activity) MarkCompleted(actor Actor, now time.Time) error {
	if a.Status != ActivityPromoting {
		return ErrConflict
	}
	completed := now
	a.Status = ActivityCompleted
	a.CompletedBy = actor.ID
	a.CompletedAt = &completed
	a.UpdatedAt = now
	return nil
}

// MarkRestored 由负责人标记恢复完成。
func (a *Activity) MarkRestored(actor Actor, now time.Time) error {
	if a.Status != ActivityRestoring {
		return ErrConflict
	}
	restored := now
	a.Status = ActivityRestored
	a.RestoredBy = actor.ID
	a.RestoredAt = &restored
	a.UpdatedAt = now
	return nil
}
