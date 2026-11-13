// Package entity 定义测试车队日志策略闭环的聚合、状态机与仓储契约。
package entity

import (
	"errors"
	"time"
)

// Role 是服务端判定的会话角色，不来自请求体。
type Role string

const (
	// RoleRequester 研发申请人。
	RoleRequester Role = "R"
	// RoleApprover 审批负责人。
	RoleApprover Role = "A"
	// RoleOperator 发布运营。
	RoleOperator Role = "O"
	// RoleVehicle 模拟车辆（仅车辆令牌场景使用）。
	RoleVehicle Role = "V"
)

// Actor 是经过签名令牌校验后的操作人身份。
type Actor struct {
	ID   string
	Name string
	Role Role
}

// Is 判断操作人是否具备指定角色。
func (a Actor) Is(role Role) bool {
	return a.Role == role
}

// Clock 是可注入时钟，业务代码不得直接调用 time.Now，以保证超时判定可测试。
type Clock interface {
	Now() time.Time
}

// SystemClock 使用系统时间（UTC）的实现。
type SystemClock struct{}

// Now 返回当前 UTC 时间。
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

// 申请状态。
const (
	AppStatusDraft    = "draft"
	AppStatusPending  = "pending"
	AppStatusRejected = "rejected"
	AppStatusApproved = "approved"
	AppStatusVoided   = "voided"
)

// 发布活动状态。
const (
	ActivityPilotPending = "pilot_pending"
	ActivityPiloting     = "piloting"
	ActivityPilotPassed  = "pilot_passed"
	ActivityPromoting    = "promoting"
	ActivityCompleted    = "completed"
	ActivityStopped      = "stopped"
	ActivityRestoring    = "restoring"
	ActivityRestored     = "restored"
)

// 发布批次。
const (
	BatchPilot     = "pilot"
	BatchPromotion = "promotion"
	BatchRestore   = "restore"
)

// 单车发布状态。
const (
	ReleaseWaiting = "waiting"
	ReleaseSuccess = "success"
	ReleaseFailed  = "failed"
	ReleaseTimeout = "timeout"
)

// 逐车展示状态中包含"未发布"。
const StatusNotReleased = "not_released"

// 回执结果。
const (
	ReceiptSuccess = "success"
	ReceiptFailure = "failure"
)

// 被拒回执原因。
const (
	RejectCrossVehicle   = "cross_vehicle"
	RejectUnknownVersion = "unknown_version"
	RejectStaleVersion   = "stale_version"
	RejectDuplicate      = "duplicate_receipt"
	RejectInvalidPayload = "invalid_payload"
)

// 业务校验边界。
const (
	// ReasonMaxLen 申请原因最大字符数。
	ReasonMaxLen = 200
	// IntervalMin 日志间隔下限（秒）。
	IntervalMin = 10
	// IntervalMax 日志间隔上限（秒）。
	IntervalMax = 300
	// DefaultInterval 初始日志间隔（秒）。
	DefaultInterval = 60
	// DefaultVehicleVersion 活动前的初始版本。
	DefaultVehicleVersion = 1
	// PilotVehicleCount 试点车辆数量。
	PilotVehicleCount = 2
	// FleetVehicleCount 车队规模。
	FleetVehicleCount = 5
)

var (
	// ErrNotFound 表示资源不存在。
	ErrNotFound = errors.New("fleetlog: not found")
	// ErrForbidden 表示当前身份不允许该操作。
	ErrForbidden = errors.New("fleetlog: forbidden")
	// ErrConflict 表示与当前状态冲突。
	ErrConflict = errors.New("fleetlog: conflict")
	// ErrInvalidInput 表示入参不合法。
	ErrInvalidInput = errors.New("fleetlog: invalid input")
	// ErrOpenActivity 表示车队已存在未结束活动。
	ErrOpenActivity = errors.New("fleetlog: open activity exists")
	// ErrSelfApproval 表示申请人与审批人相同。
	ErrSelfApproval = errors.New("fleetlog: approver must differ from requester")
	// ErrReasonRequired 表示原因缺失。
	ErrReasonRequired = errors.New("fleetlog: reason required")
	// ErrReasonTooLong 表示原因超长。
	ErrReasonTooLong = errors.New("fleetlog: reason too long")
	// ErrIntervalOutOfRange 表示间隔越界。
	ErrIntervalOutOfRange = errors.New("fleetlog: interval out of range")
	// ErrNotRequester 表示操作人不是申请本人。
	ErrNotRequester = errors.New("fleetlog: not the requester")
	// ErrApplicationNotEditable 表示申请不可编辑。
	ErrApplicationNotEditable = errors.New("fleetlog: application not editable")
	// ErrApplicationNotSubmittable 表示申请不可提交。
	ErrApplicationNotSubmittable = errors.New("fleetlog: application not submittable")
	// ErrApplicationNotPending 表示申请不在待审批状态。
	ErrApplicationNotPending = errors.New("fleetlog: application not pending")
	// ErrApplicationNotApproved 表示申请未批准。
	ErrApplicationNotApproved = errors.New("fleetlog: application not approved")
	// ErrInvalidPilotSelection 表示试点车辆不合法。
	ErrInvalidPilotSelection = errors.New("fleetlog: invalid pilot selection")
	// ErrPilotNotPassed 表示试点未通过。
	ErrPilotNotPassed = errors.New("fleetlog: pilot not passed")
	// ErrUnconfirmedVehicles 表示仍有未确认车辆。
	ErrUnconfirmedVehicles = errors.New("fleetlog: unconfirmed vehicles")
	// ErrVehicleOffline 表示车辆离线。
	ErrVehicleOffline = errors.New("fleetlog: vehicle offline")
	// ErrTimeoutSecondsNotConfigured 表示超时配置缺失。
	ErrTimeoutSecondsNotConfigured = errors.New("fleetlog: timeout not configured")
)
