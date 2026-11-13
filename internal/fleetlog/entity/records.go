package entity

import (
	"time"

	"github.com/google/uuid"
)

// Receipt 是被服务端接受的回执。
type Receipt struct {
	ID            string
	ReleaseID     string
	ActivityID    string
	VehicleID     string
	Version       int
	Result        string
	FailureReason string
	ReportedAt    time.Time
	AcceptedAt    time.Time
}

// NewReceipt 构造被接受的回执记录。
func NewReceipt(release *Release, result string, failureReason string, now time.Time) *Receipt {
	return &Receipt{
		ID:            uuid.NewString(),
		ReleaseID:     release.ID,
		ActivityID:    release.ActivityID,
		VehicleID:     release.VehicleID,
		Version:       release.Version,
		Result:        result,
		FailureReason: failureReason,
		ReportedAt:    now,
		AcceptedAt:    now,
	}
}

// RejectedReceipt 是未通过判定的回执，保留用于核查"为什么没算"。
type RejectedReceipt struct {
	ID         string
	ActivityID string
	VehicleID  string
	Version    int
	Reason     string
	Detail     string
	ReportedAt time.Time
}

// NewRejectedReceipt 构造一条被拒回执记录。
func NewRejectedReceipt(vehicleID string, version int, reason string, detail string, now time.Time) *RejectedReceipt {
	return &RejectedReceipt{
		ID:         uuid.NewString(),
		VehicleID:  vehicleID,
		Version:    version,
		Reason:     reason,
		Detail:     detail,
		ReportedAt: now,
	}
}

// TimeoutEvent 是超时事件历史，迟到成功不清除该事件。
type TimeoutEvent struct {
	ID         string
	ReleaseID  string
	ActivityID string
	VehicleID  string
	Version    int
	TimedOutAt time.Time
	ResolvedAt *time.Time
}

// NewTimeoutEvent 构造超时事件。
func NewTimeoutEvent(release *Release, now time.Time) *TimeoutEvent {
	return &TimeoutEvent{
		ID:         uuid.NewString(),
		ReleaseID:  release.ID,
		ActivityID: release.ActivityID,
		VehicleID:  release.VehicleID,
		Version:    release.Version,
		TimedOutAt: now,
	}
}

// PullRecord 是车辆拉取配置的记录，不改变生效值。
type PullRecord struct {
	ID        string
	VehicleID string
	Version   int
	Interval  int
	ReleaseID string
	PulledAt  time.Time
}

// NewPullRecord 构造拉取记录。
func NewPullRecord(vehicleID string, version int, interval int, releaseID string, now time.Time) *PullRecord {
	return &PullRecord{
		ID:        uuid.NewString(),
		VehicleID: vehicleID,
		Version:   version,
		Interval:  interval,
		ReleaseID: releaseID,
		PulledAt:  now,
	}
}

// Audit 是操作审计记录。
type Audit struct {
	ID         string
	ActorID    string
	ActorRole  string
	Action     string
	TargetType string
	TargetID   string
	Detail     string
	CreatedAt  time.Time
}

// NewAudit 构造审计记录。
func NewAudit(actor Actor, action string, targetType string, targetID string, detail string, now time.Time) *Audit {
	return &Audit{
		ID:         uuid.NewString(),
		ActorID:    actor.ID,
		ActorRole:  string(actor.Role),
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		CreatedAt:  now,
	}
}

// Baseline 是活动创建时快照的活动前生效值，作为恢复目标。
type Baseline struct {
	ActivityID string
	VehicleID  string
	Interval   int
	Version    int
	CreatedAt  time.Time
}

// FleetState 是车队级单行状态，用于数据库层保证同时只有一个未结束活动。
type FleetState struct {
	ID             string
	OpenActivityID string
	UpdatedAt      time.Time
}

// FleetStateID 是单行车队状态的固定主键。
const FleetStateID = "fleet-default"
