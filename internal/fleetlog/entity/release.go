package entity

import (
	"time"

	"github.com/google/uuid"
)

// Vehicle 是车队中的一台测试车辆，分别保存期望版本与生效值。
type Vehicle struct {
	ID                string
	Online            bool
	EffectiveInterval int
	EffectiveVersion  int
	CurrentVersion    int
	UpdatedAt         time.Time
}

// ExpectedInterval 返回该车当前最新已发布版本的间隔，未发布过则为初始值。
func (v *Vehicle) ExpectedInterval(latest *Release) int {
	if latest == nil {
		return DefaultInterval
	}
	return latest.ExpectedInterval
}

// Release 是一条下发记录，只证明"已发出"，不代表车端已生效。
type Release struct {
	ID                string
	ActivityID        string
	VehicleID         string
	Version           int
	Batch             string
	ExpectedInterval  int
	BaselineInterval  int
	Status            string
	FailureReason     string
	DispatchedAt      time.Time
	ReceiptAt         *time.Time
	TimeoutAt         *time.Time
	Late              bool
	DuplicateReceipts int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewRelease 创建一条等待回执的下发记录。
func NewRelease(activityID string, vehicleID string, version int, batch string, expected int, baseline int, now time.Time) *Release {
	return &Release{
		ID:               uuid.NewString(),
		ActivityID:       activityID,
		VehicleID:        vehicleID,
		Version:          version,
		Batch:            batch,
		ExpectedInterval: expected,
		BaselineInterval: baseline,
		Status:           ReleaseWaiting,
		DispatchedAt:     now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// TimedOut 判断当前版本是否已超过超时阈值且仍无有效回执。
func (r *Release) TimedOut(now time.Time, timeout time.Duration) bool {
	if r.Status != ReleaseWaiting {
		return false
	}
	return !now.Before(r.DispatchedAt.Add(timeout))
}

// MarkTimeout 把等待中的发布记录标记为超时展示态。
func (r *Release) MarkTimeout(now time.Time) bool {
	if r.Status != ReleaseWaiting {
		return false
	}
	timedOut := now
	r.Status = ReleaseTimeout
	r.TimeoutAt = &timedOut
	r.UpdatedAt = now
	return true
}

// MarkSuccess 接受当前版本成功回执；此前超时的记录保留超时历史并标记迟到。
func (r *Release) MarkSuccess(now time.Time) {
	receipt := now
	r.Status = ReleaseSuccess
	r.ReceiptAt = &receipt
	r.FailureReason = ""
	if r.TimeoutAt != nil {
		r.Late = true
	}
	r.UpdatedAt = now
}

// MarkFailure 接受当前版本失败回执，不改变既有生效值。
func (r *Release) MarkFailure(reason string, now time.Time) {
	receipt := now
	r.Status = ReleaseFailed
	r.FailureReason = reason
	r.ReceiptAt = &receipt
	r.UpdatedAt = now
}

// Duplicate 记录一次重复成功回执，不重复记账。
func (r *Release) Duplicate() {
	r.DuplicateReceipts++
}
