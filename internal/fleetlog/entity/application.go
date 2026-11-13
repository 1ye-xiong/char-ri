package entity

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Application 是日志间隔调整申请聚合。
type Application struct {
	ID            string
	VersionNo     int
	RequesterID   string
	RequesterName string
	Reason        string
	Interval      int
	Status        string
	RejectReason  string
	DecidedBy     string
	SubmittedAt   *time.Time
	DecidedAt     *time.Time
	ActivityID    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ApplicationSnapshot 保存申请的历史版本，退回重提不覆盖旧记录。
type ApplicationSnapshot struct {
	ID            string
	ApplicationID string
	VersionNo     int
	Reason        string
	Interval      int
	Status        string
	RejectReason  string
	DecidedBy     string
	CreatedAt     time.Time
}

// NewApplication 创建草稿申请，草稿允许原因/间隔暂未填写完整。
func NewApplication(actor Actor, reason string, interval int, now time.Time) *Application {
	return &Application{
		ID:            uuid.NewString(),
		VersionNo:     1,
		RequesterID:   actor.ID,
		RequesterName: actor.Name,
		Reason:        strings.TrimSpace(reason),
		Interval:      interval,
		Status:        AppStatusDraft,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// ValidateReason 校验申请原因非空白且不超过 200 字符。
func ValidateReason(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrReasonRequired
	}
	if utf8.RuneCountInString(reason) > ReasonMaxLen {
		return ErrReasonTooLong
	}
	return nil
}

// ValidateInterval 校验日志间隔落在 10～300 秒。
func ValidateInterval(interval int) error {
	if interval < IntervalMin || interval > IntervalMax {
		return ErrIntervalOutOfRange
	}
	return nil
}

// ValidateFields 校验提交所需的完整字段。
func (a *Application) ValidateFields() error {
	if err := ValidateReason(a.Reason); err != nil {
		return err
	}
	return ValidateInterval(a.Interval)
}

// Edit 修改草稿或已退回申请；已批准申请参数锁定。
func (a *Application) Edit(actor Actor, reason string, interval int, now time.Time) error {
	if a.RequesterID != actor.ID {
		return ErrNotRequester
	}
	if a.Status != AppStatusDraft && a.Status != AppStatusRejected {
		return ErrApplicationNotEditable
	}
	if interval != 0 {
		if err := ValidateInterval(interval); err != nil {
			return err
		}
	}
	if interval < 0 || interval > IntervalMax {
		return ErrIntervalOutOfRange
	}
	a.Reason = strings.TrimSpace(reason)
	a.Interval = interval
	a.UpdatedAt = now
	return nil
}

// Snapshot 生成当前字段的历史快照。
func (a *Application) Snapshot(now time.Time) *ApplicationSnapshot {
	return &ApplicationSnapshot{
		ID:            uuid.NewString(),
		ApplicationID: a.ID,
		VersionNo:     a.VersionNo,
		Reason:        a.Reason,
		Interval:      a.Interval,
		Status:        a.Status,
		RejectReason:  a.RejectReason,
		DecidedBy:     a.DecidedBy,
		CreatedAt:     now,
	}
}

// Submit 将草稿或已退回申请提交为待审批；退回重提生成新的版本号。
func (a *Application) Submit(now time.Time) error {
	if a.Status != AppStatusDraft && a.Status != AppStatusRejected {
		return ErrApplicationNotSubmittable
	}
	if err := a.ValidateFields(); err != nil {
		return err
	}
	if a.Status == AppStatusRejected {
		a.VersionNo++
		a.RejectReason = ""
		a.DecidedBy = ""
		a.DecidedAt = nil
	}
	submitted := now
	a.Status = AppStatusPending
	a.SubmittedAt = &submitted
	a.UpdatedAt = now
	return nil
}

// Approve 批准待审批申请，禁止申请人自审。
func (a *Application) Approve(approver Actor, now time.Time) error {
	if a.Status != AppStatusPending {
		return ErrApplicationNotPending
	}
	if approver.ID == a.RequesterID {
		return ErrSelfApproval
	}
	decided := now
	a.Status = AppStatusApproved
	a.DecidedBy = approver.ID
	a.DecidedAt = &decided
	a.UpdatedAt = now
	return nil
}

// Reject 退回待审批申请，必须给出退回原因。
func (a *Application) Reject(approver Actor, reason string, now time.Time) error {
	if a.Status != AppStatusPending {
		return ErrApplicationNotPending
	}
	if approver.ID == a.RequesterID {
		return ErrSelfApproval
	}
	if err := ValidateReason(reason); err != nil {
		return err
	}
	decided := now
	a.Status = AppStatusRejected
	a.RejectReason = strings.TrimSpace(reason)
	a.DecidedBy = approver.ID
	a.DecidedAt = &decided
	a.UpdatedAt = now
	return nil
}

// Void 将已批准申请标记为已使用。
func (a *Application) Void(activityID string, now time.Time) error {
	if a.Status != AppStatusApproved {
		return ErrApplicationNotApproved
	}
	a.Status = AppStatusVoided
	a.ActivityID = activityID
	a.UpdatedAt = now
	return nil
}
