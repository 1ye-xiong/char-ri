package service

import (
	"errors"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
)

// toAppError 把领域哨兵错误转换为带稳定 code 的应用错误。
func toAppError(err error) error {
	if err == nil {
		return nil
	}
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return err
	}
	switch {
	case errors.Is(err, entity.ErrNotFound):
		return apperrors.NotFound("fleetlog_not_found", "记录不存在")
	case errors.Is(err, entity.ErrForbidden):
		return apperrors.Unauthorized("fleetlog_forbidden", "当前身份无权执行该操作")
	case errors.Is(err, entity.ErrSelfApproval):
		return apperrors.Conflict("fleetlog_self_approval", "审批负责人不得审批自己提交的申请")
	case errors.Is(err, entity.ErrReasonRequired):
		return apperrors.InvalidArgument("fleetlog_reason_required", "申请原因不能为空")
	case errors.Is(err, entity.ErrReasonTooLong):
		return apperrors.InvalidArgument("fleetlog_reason_too_long", "申请原因不能超过 200 个字符")
	case errors.Is(err, entity.ErrIntervalOutOfRange):
		return apperrors.InvalidArgument("fleetlog_interval_out_of_range", "日志间隔必须为 10～300 的整数秒")
	case errors.Is(err, entity.ErrNotRequester):
		return apperrors.Conflict("fleetlog_not_requester", "只能修改本人提交的申请")
	case errors.Is(err, entity.ErrApplicationNotEditable):
		return apperrors.Conflict("fleetlog_application_not_editable", "该申请当前状态不可修改")
	case errors.Is(err, entity.ErrApplicationNotSubmittable):
		return apperrors.Conflict("fleetlog_application_not_submittable", "该申请当前状态不可提交")
	case errors.Is(err, entity.ErrApplicationNotPending):
		return apperrors.Conflict("fleetlog_application_not_pending", "该申请已被处理")
	case errors.Is(err, entity.ErrApplicationNotApproved):
		return apperrors.Conflict("fleetlog_application_not_approved", "申请未批准，不能创建发布活动")
	case errors.Is(err, entity.ErrOpenActivity):
		return apperrors.Conflict("fleetlog_open_activity", "该车队已存在未结束的发布活动")
	case errors.Is(err, entity.ErrInvalidPilotSelection):
		return apperrors.InvalidArgument("fleetlog_invalid_pilot", "试点车必须为 2 台不同车辆")
	case errors.Is(err, entity.ErrPilotNotPassed):
		return apperrors.Conflict("fleetlog_pilot_not_passed", "试点未全部成功，不能推广")
	case errors.Is(err, entity.ErrUnconfirmedVehicles):
		return apperrors.Conflict("fleetlog_unconfirmed_vehicles", "仍存在未确认车辆，不能标记完成")
	case errors.Is(err, entity.ErrVehicleOffline):
		return apperrors.Conflict("fleetlog_vehicle_offline", "车辆离线，无法拉取配置")
	case errors.Is(err, entity.ErrInvalidInput):
		return apperrors.InvalidArgument("fleetlog_invalid_input", "入参不合法")
	case errors.Is(err, entity.ErrConflict):
		return apperrors.Conflict("fleetlog_conflict", "当前状态不允许该操作")
	default:
		return apperrors.Wrap(apperrors.KindInternal, "fleetlog_internal", "fleetlog 服务内部错误", err)
	}
}
