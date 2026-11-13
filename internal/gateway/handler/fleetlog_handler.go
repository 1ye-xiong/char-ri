package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	fleetlogv1 "github.com/BwCloudWeGo/bw-cli/api/gen/fleetlog/v1"
	"github.com/BwCloudWeGo/bw-cli/internal/gateway/request"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
	"github.com/BwCloudWeGo/bw-cli/pkg/httpx"
)

const (
	metadataKeyAuthorization = "authorization"
	metadataKeyVehicleToken  = "x-vehicle-token"
)

// FleetLogHandler 将 fleetlog HTTP 接口适配到 gRPC client。
type FleetLogHandler struct {
	client fleetlogv1.FleetLogServiceClient
	log    *zap.Logger
}

// NewFleetLogHandler 注入 fleetlog gRPC client。
func NewFleetLogHandler(client fleetlogv1.FleetLogServiceClient, log *zap.Logger) *FleetLogHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &FleetLogHandler{client: client, log: log}
}

// fleetLogContext 把会话与车辆凭证透传到 gRPC 元数据，身份由服务端校验。
func fleetLogContext(c *gin.Context) context.Context {
	ctx := outgoingContext(c)
	if authorization := c.GetHeader("Authorization"); authorization != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, metadataKeyAuthorization, authorization)
	}
	if vehicleToken := c.GetHeader("X-Vehicle-Token"); vehicleToken != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, metadataKeyVehicleToken, vehicleToken)
	}
	return ctx
}

// Login 处理 POST /api/v1/fleetlog/login。
func (h *FleetLogHandler) Login(c *gin.Context) {
	var req request.FleetLogLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.Login(c.Request.Context(), &fleetlogv1.LoginRequest{Account: req.Account, Password: req.Password})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// CreateApplication 创建申请草稿。
func (h *FleetLogHandler) CreateApplication(c *gin.Context) {
	var req request.CreateFleetLogApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.CreateApplication(fleetLogContext(c), &fleetlogv1.CreateApplicationRequest{
		Reason:          req.Reason,
		IntervalSeconds: req.IntervalSeconds,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.Created(c, resp)
}

// UpdateApplication 修改草稿或退回申请。
func (h *FleetLogHandler) UpdateApplication(c *gin.Context) {
	var req request.UpdateFleetLogApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.UpdateApplication(fleetLogContext(c), &fleetlogv1.UpdateApplicationRequest{
		Id:              c.Param("id"),
		Reason:          req.Reason,
		IntervalSeconds: req.IntervalSeconds,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// SubmitApplication 提交申请。
func (h *FleetLogHandler) SubmitApplication(c *gin.Context) {
	resp, err := h.client.SubmitApplication(fleetLogContext(c), &fleetlogv1.SubmitApplicationRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// ApproveApplication 批准申请。
func (h *FleetLogHandler) ApproveApplication(c *gin.Context) {
	resp, err := h.client.ApproveApplication(fleetLogContext(c), &fleetlogv1.ApproveApplicationRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// RejectApplication 退回申请。
func (h *FleetLogHandler) RejectApplication(c *gin.Context) {
	var req request.RejectFleetLogApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.RejectApplication(fleetLogContext(c), &fleetlogv1.RejectApplicationRequest{Id: c.Param("id"), Reason: req.Reason})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// ListApplications 返回申请列表。
func (h *FleetLogHandler) ListApplications(c *gin.Context) {
	resp, err := h.client.ListApplications(fleetLogContext(c), &fleetlogv1.ListApplicationsRequest{})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// CreateActivity 创建发布活动。
func (h *FleetLogHandler) CreateActivity(c *gin.Context) {
	var req request.CreateFleetLogActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.CreateActivity(fleetLogContext(c), &fleetlogv1.CreateActivityRequest{
		ApplicationId: req.ApplicationID,
		PilotVehicles: req.PilotVehicles,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.Created(c, resp)
}

// StartPilot 执行试点下发。
func (h *FleetLogHandler) StartPilot(c *gin.Context) {
	h.activityAction(c, func(ctx context.Context, id string) (*fleetlogv1.ActivityDetailResponse, error) {
		return h.client.StartPilot(ctx, &fleetlogv1.ActivityActionRequest{Id: id})
	})
}

// ConfirmPromotion 确认推广。
func (h *FleetLogHandler) ConfirmPromotion(c *gin.Context) {
	h.activityAction(c, func(ctx context.Context, id string) (*fleetlogv1.ActivityDetailResponse, error) {
		return h.client.ConfirmPromotion(ctx, &fleetlogv1.ActivityActionRequest{Id: id})
	})
}

// StopActivity 停止活动。
func (h *FleetLogHandler) StopActivity(c *gin.Context) {
	var req request.StopFleetLogActivityRequest
	_ = c.ShouldBindJSON(&req)
	resp, err := h.client.StopActivity(fleetLogContext(c), &fleetlogv1.StopActivityRequest{Id: c.Param("id"), Reason: req.Reason})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// ApproveRestore 批准恢复。
func (h *FleetLogHandler) ApproveRestore(c *gin.Context) {
	h.activityAction(c, func(ctx context.Context, id string) (*fleetlogv1.ActivityDetailResponse, error) {
		return h.client.ApproveRestore(ctx, &fleetlogv1.ActivityActionRequest{Id: id})
	})
}

// StartRestore 发起恢复下发。
func (h *FleetLogHandler) StartRestore(c *gin.Context) {
	h.activityAction(c, func(ctx context.Context, id string) (*fleetlogv1.ActivityDetailResponse, error) {
		return h.client.StartRestore(ctx, &fleetlogv1.ActivityActionRequest{Id: id})
	})
}

// MarkCompleted 标记活动完成。
func (h *FleetLogHandler) MarkCompleted(c *gin.Context) {
	h.activityAction(c, func(ctx context.Context, id string) (*fleetlogv1.ActivityDetailResponse, error) {
		return h.client.MarkCompleted(ctx, &fleetlogv1.ActivityActionRequest{Id: id})
	})
}

// MarkRestored 标记恢复完成。
func (h *FleetLogHandler) MarkRestored(c *gin.Context) {
	h.activityAction(c, func(ctx context.Context, id string) (*fleetlogv1.ActivityDetailResponse, error) {
		return h.client.MarkRestored(ctx, &fleetlogv1.ActivityActionRequest{Id: id})
	})
}

func (h *FleetLogHandler) activityAction(c *gin.Context, call func(ctx context.Context, id string) (*fleetlogv1.ActivityDetailResponse, error)) {
	resp, err := call(fleetLogContext(c), c.Param("id"))
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// GetActivity 返回活动详情。
func (h *FleetLogHandler) GetActivity(c *gin.Context) {
	resp, err := h.client.GetActivity(fleetLogContext(c), &fleetlogv1.GetActivityRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// GetCurrentActivity 返回当前未结束活动。
func (h *FleetLogHandler) GetCurrentActivity(c *gin.Context) {
	resp, err := h.client.GetCurrentActivity(fleetLogContext(c), &fleetlogv1.CurrentActivityRequest{})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// ListActivities 返回活动列表。
func (h *FleetLogHandler) ListActivities(c *gin.Context) {
	resp, err := h.client.ListActivities(fleetLogContext(c), &fleetlogv1.ListActivitiesRequest{})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// ListVehicles 返回车辆列表。
func (h *FleetLogHandler) ListVehicles(c *gin.Context) {
	resp, err := h.client.ListVehicles(fleetLogContext(c), &fleetlogv1.ListVehiclesRequest{})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// SetVehicleOnline 切换车辆在线状态。
func (h *FleetLogHandler) SetVehicleOnline(c *gin.Context) {
	var req request.SetFleetLogVehicleOnlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.SetVehicleOnline(fleetLogContext(c), &fleetlogv1.SetVehicleOnlineRequest{Id: c.Param("id"), Online: req.Online})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// PullConfig 处理车辆拉取。
func (h *FleetLogHandler) PullConfig(c *gin.Context) {
	var req request.FleetLogVehiclePullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.PullConfig(fleetLogContext(c), &fleetlogv1.PullConfigRequest{VehicleId: req.VehicleID})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// ReportReceipt 处理车辆回执。
func (h *FleetLogHandler) ReportReceipt(c *gin.Context) {
	var req request.FleetLogVehicleReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.ReportReceipt(fleetLogContext(c), &fleetlogv1.ReportReceiptRequest{
		VehicleId:     req.VehicleID,
		Version:       req.Version,
		Result:        req.Result,
		FailureReason: req.FailureReason,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// ListRejectedReceipts 返回被拒回执。
func (h *FleetLogHandler) ListRejectedReceipts(c *gin.Context) {
	resp, err := h.client.ListRejectedReceipts(fleetLogContext(c), &fleetlogv1.ListRejectedReceiptsRequest{
		ActivityId: c.Query("activity_id"),
		VehicleId:  c.Query("vehicle_id"),
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// ListAudits 返回审计记录。
func (h *FleetLogHandler) ListAudits(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	resp, err := h.client.ListAudits(fleetLogContext(c), &fleetlogv1.ListAuditsRequest{
		ActivityId: c.Query("activity_id"),
		Limit:      int32(limit),
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}
