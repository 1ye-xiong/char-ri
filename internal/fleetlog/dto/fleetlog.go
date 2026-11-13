package dto

import (
	"time"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

const timeLayout = "2006-01-02 15:04:05"

// FormatTime 以本地时区格式化时间。
func FormatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Local().Format(timeLayout)
}

// FormatTimePtr 格式化可空时间。
func FormatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return FormatTime(*value)
}

// SessionDTO 是登录成功后的会话信息。
type SessionDTO struct {
	Token          string
	UserID         string
	Role           string
	DisplayName    string
	TimeoutSeconds int
}

// ApplicationVersionDTO 是申请历史版本。
type ApplicationVersionDTO struct {
	VersionNo    int
	Reason       string
	Interval     int
	Status       string
	RejectReason string
	DecidedBy    string
	CreatedAt    string
}

// ApplicationDTO 是申请视图。
type ApplicationDTO struct {
	ID            string
	VersionNo     int
	RequesterID   string
	RequesterName string
	Reason        string
	Interval      int
	Status        string
	RejectReason  string
	DecidedBy     string
	SubmittedAt   string
	DecidedAt     string
	CreatedAt     string
	ActivityID    string
	CanEdit       bool
	CanSubmit     bool
	CanDecide     bool
	History       []*ApplicationVersionDTO
}

// ActivityDTO 是活动视图。
type ActivityDTO struct {
	ID                string
	ApplicationID     string
	Interval          int
	PilotVehicles     []string
	PromotionVehicles []string
	Status            string
	CreatedBy         string
	CreatedAt         string
	StoppedBy         string
	StoppedAt         string
	StopScope         string
	RestoreApprovedBy string
	RestoreApprovedAt string
	RestoreStartedBy  string
	RestoreStartedAt  string
	CompletedBy       string
	CompletedAt       string
	RestoredBy        string
	RestoredAt        string
}

// VehicleRowDTO 是活动内逐车核查行。
type VehicleRowDTO struct {
	VehicleID         string
	Released          bool
	Batch             string
	Version           int
	BaselineInterval  int
	ExpectedInterval  int
	EffectiveInterval int
	EffectiveVersion  int
	Status            string
	FailureReason     string
	DispatchedAt      string
	ReceiptAt         string
	TimeoutAt         string
	Late              bool
	DuplicateReceipts int
}

// SummaryDTO 是活动汇总，必须与逐车明细可互相校验。
type SummaryDTO struct {
	Total           int
	Released        int
	NotReleased     int
	Success         int
	Failed          int
	Timeout         int
	Waiting         int
	RejectedReceipt int
	Effective       int
}

// ActionsDTO 是服务端按角色与状态推导的可用动作。
type ActionsDTO struct {
	CanStartPilot        bool
	CanPromote           bool
	CanStop              bool
	CanApproveRestore    bool
	CanStartRestore      bool
	CanMarkCompleted     bool
	CanMarkRestored      bool
	PromoteBlockedReason string
}

// RejectedReceiptDTO 是被拒回执视图。
type RejectedReceiptDTO struct {
	ID         string
	VehicleID  string
	Version    int
	Reason     string
	Detail     string
	ReportedAt string
}

// TimeoutEventDTO 是超时事件视图。
type TimeoutEventDTO struct {
	VehicleID  string
	Version    int
	TimedOutAt string
	ResolvedAt string
}

// AuditDTO 是审计视图。
type AuditDTO struct {
	ActorID    string
	ActorRole  string
	Action     string
	TargetType string
	TargetID   string
	Detail     string
	CreatedAt  string
}

// ActivityDetailDTO 是活动看板与核查页的完整数据。
type ActivityDetailDTO struct {
	Activity    *ActivityDTO
	Application *ApplicationDTO
	Rows        []*VehicleRowDTO
	Summary     *SummaryDTO
	Rejected    []*RejectedReceiptDTO
	Timeouts    []*TimeoutEventDTO
	Audits      []*AuditDTO
	Actions     *ActionsDTO
	TimeoutSecs int
	ServerTime  string
}

// VehicleDTO 是车辆视图。
type VehicleDTO struct {
	ID                string
	Online            bool
	EffectiveInterval int
	EffectiveVersion  int
	CurrentVersion    int
	ExpectedInterval  int
	LatestStatus      string
}

// PullResultDTO 是车辆拉取结果。
type PullResultDTO struct {
	VehicleID  string
	Version    int
	Interval   int
	Online     bool
	PulledAt   string
	Status     string
	HasRelease bool
}

// ReceiptResultDTO 是回执判定结果。
type ReceiptResultDTO struct {
	Accepted          bool
	Reason            string
	Message           string
	VehicleID         string
	Version           int
	ReleaseStatus     string
	ActivityStatus    string
	EffectiveInterval int
	EffectiveVersion  int
}

// NewApplicationDTO 转换申请视图。
func NewApplicationDTO(app *entity.Application, history []*entity.ApplicationSnapshot, canEdit bool, canSubmit bool, canDecide bool) *ApplicationDTO {
	output := &ApplicationDTO{
		ID:            app.ID,
		VersionNo:     app.VersionNo,
		RequesterID:   app.RequesterID,
		RequesterName: app.RequesterName,
		Reason:        app.Reason,
		Interval:      app.Interval,
		Status:        app.Status,
		RejectReason:  app.RejectReason,
		DecidedBy:     app.DecidedBy,
		SubmittedAt:   FormatTimePtr(app.SubmittedAt),
		DecidedAt:     FormatTimePtr(app.DecidedAt),
		CreatedAt:     FormatTime(app.CreatedAt),
		ActivityID:    app.ActivityID,
		CanEdit:       canEdit,
		CanSubmit:     canSubmit,
		CanDecide:     canDecide,
		History:       make([]*ApplicationVersionDTO, 0, len(history)),
	}
	for _, item := range history {
		output.History = append(output.History, &ApplicationVersionDTO{
			VersionNo:    item.VersionNo,
			Reason:       item.Reason,
			Interval:     item.Interval,
			Status:       item.Status,
			RejectReason: item.RejectReason,
			DecidedBy:    item.DecidedBy,
			CreatedAt:    FormatTime(item.CreatedAt),
		})
	}
	return output
}

// NewActivityDTO 转换活动视图。
func NewActivityDTO(activity *entity.Activity) *ActivityDTO {
	return &ActivityDTO{
		ID:                activity.ID,
		ApplicationID:     activity.ApplicationID,
		Interval:          activity.Interval,
		PilotVehicles:     append([]string{}, activity.PilotVehicles...),
		PromotionVehicles: append([]string{}, activity.PromotionVehicles...),
		Status:            activity.Status,
		CreatedBy:         activity.CreatedBy,
		CreatedAt:         FormatTime(activity.CreatedAt),
		StoppedBy:         activity.StoppedBy,
		StoppedAt:         FormatTimePtr(activity.StoppedAt),
		StopScope:         activity.StopScope,
		RestoreApprovedBy: activity.RestoreApprovedBy,
		RestoreApprovedAt: FormatTimePtr(activity.RestoreApprovedAt),
		RestoreStartedBy:  activity.RestoreStartedBy,
		RestoreStartedAt:  FormatTimePtr(activity.RestoreStartedAt),
		CompletedBy:       activity.CompletedBy,
		CompletedAt:       FormatTimePtr(activity.CompletedAt),
		RestoredBy:        activity.RestoredBy,
		RestoredAt:        FormatTimePtr(activity.RestoredAt),
	}
}

// NewVehicleDTO 转换车辆视图。
func NewVehicleDTO(vehicle *entity.Vehicle, latest *entity.Release) *VehicleDTO {
	output := &VehicleDTO{
		ID:                vehicle.ID,
		Online:            vehicle.Online,
		EffectiveInterval: vehicle.EffectiveInterval,
		EffectiveVersion:  vehicle.EffectiveVersion,
		CurrentVersion:    vehicle.CurrentVersion,
		ExpectedInterval:  vehicle.ExpectedInterval(latest),
	}
	if latest != nil {
		output.LatestStatus = latest.Status
	}
	return output
}

// NewRejectedReceiptDTO 转换被拒回执视图。
func NewRejectedReceiptDTO(item *entity.RejectedReceipt) *RejectedReceiptDTO {
	return &RejectedReceiptDTO{
		ID:         item.ID,
		VehicleID:  item.VehicleID,
		Version:    item.Version,
		Reason:     item.Reason,
		Detail:     item.Detail,
		ReportedAt: FormatTime(item.ReportedAt),
	}
}

// NewTimeoutEventDTO 转换超时事件视图。
func NewTimeoutEventDTO(item *entity.TimeoutEvent) *TimeoutEventDTO {
	return &TimeoutEventDTO{
		VehicleID:  item.VehicleID,
		Version:    item.Version,
		TimedOutAt: FormatTime(item.TimedOutAt),
		ResolvedAt: FormatTimePtr(item.ResolvedAt),
	}
}

// NewAuditDTO 转换审计视图。
func NewAuditDTO(item *entity.Audit) *AuditDTO {
	return &AuditDTO{
		ActorID:    item.ActorID,
		ActorRole:  item.ActorRole,
		Action:     item.Action,
		TargetType: item.TargetType,
		TargetID:   item.TargetID,
		Detail:     item.Detail,
		CreatedAt:  FormatTime(item.CreatedAt),
	}
}
