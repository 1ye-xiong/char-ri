package handler

import (
	"context"

	"go.uber.org/zap"

	fleetlogv1 "github.com/BwCloudWeGo/bw-cli/api/gen/fleetlog/v1"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/service"
)

// Server 将 fleetlog gRPC 请求适配到 service 用例。
type Server struct {
	fleetlogv1.UnimplementedFleetLogServiceServer
	svc  *service.Service
	auth *authHelper
	log  *zap.Logger
}

// NewServer 创建 fleetlog gRPC 服务端适配器。
func NewServer(svc *service.Service, secret string, issuer string, log *zap.Logger) *Server {
	if log == nil {
		log = zap.NewNop()
	}
	return &Server{svc: svc, auth: newAuthHelper(svc, secret, issuer), log: log}
}

// Login 处理账号登录。
func (s *Server) Login(ctx context.Context, req *fleetlogv1.LoginRequest) (*fleetlogv1.LoginResponse, error) {
	session, err := s.svc.Login(ctx, dto.LoginCommand{Account: req.GetAccount(), Password: req.GetPassword()})
	if err != nil {
		return nil, err
	}
	return &fleetlogv1.LoginResponse{
		Token:          session.Token,
		UserId:         session.UserID,
		Role:           session.Role,
		DisplayName:    session.DisplayName,
		TimeoutSeconds: int32(session.TimeoutSeconds),
	}, nil
}

// CreateApplication 创建申请草稿。
func (s *Server) CreateApplication(ctx context.Context, req *fleetlogv1.CreateApplicationRequest) (*fleetlogv1.ApplicationItem, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.svc.CreateApplication(ctx, actor, dto.CreateApplicationCommand{Reason: req.GetReason(), Interval: int(req.GetIntervalSeconds())})
	if err != nil {
		return nil, err
	}
	return toApplicationItem(item), nil
}

// UpdateApplication 修改草稿或退回申请。
func (s *Server) UpdateApplication(ctx context.Context, req *fleetlogv1.UpdateApplicationRequest) (*fleetlogv1.ApplicationItem, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.svc.UpdateApplication(ctx, actor, dto.UpdateApplicationCommand{
		ID:       req.GetId(),
		Reason:   req.GetReason(),
		Interval: int(req.GetIntervalSeconds()),
	})
	if err != nil {
		return nil, err
	}
	return toApplicationItem(item), nil
}

// SubmitApplication 提交申请。
func (s *Server) SubmitApplication(ctx context.Context, req *fleetlogv1.SubmitApplicationRequest) (*fleetlogv1.ApplicationItem, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.svc.SubmitApplication(ctx, actor, req.GetId())
	if err != nil {
		return nil, err
	}
	return toApplicationItem(item), nil
}

// ApproveApplication 批准申请。
func (s *Server) ApproveApplication(ctx context.Context, req *fleetlogv1.ApproveApplicationRequest) (*fleetlogv1.ApplicationItem, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.svc.ApproveApplication(ctx, actor, req.GetId())
	if err != nil {
		return nil, err
	}
	return toApplicationItem(item), nil
}

// RejectApplication 退回申请。
func (s *Server) RejectApplication(ctx context.Context, req *fleetlogv1.RejectApplicationRequest) (*fleetlogv1.ApplicationItem, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.svc.RejectApplication(ctx, actor, req.GetId(), req.GetReason())
	if err != nil {
		return nil, err
	}
	return toApplicationItem(item), nil
}

// ListApplications 返回当前身份可见的申请。
func (s *Server) ListApplications(ctx context.Context, _ *fleetlogv1.ListApplicationsRequest) (*fleetlogv1.ListApplicationsResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.svc.ListApplications(ctx, actor)
	if err != nil {
		return nil, err
	}
	response := &fleetlogv1.ListApplicationsResponse{Items: make([]*fleetlogv1.ApplicationItem, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toApplicationItem(item))
	}
	return response, nil
}

// CreateActivity 创建发布活动。
func (s *Server) CreateActivity(ctx context.Context, req *fleetlogv1.CreateActivityRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.CreateActivity(ctx, actor, dto.CreateActivityCommand{
		ApplicationID: req.GetApplicationId(),
		PilotVehicles: req.GetPilotVehicles(),
	})
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// StartPilot 执行试点下发。
func (s *Server) StartPilot(ctx context.Context, req *fleetlogv1.ActivityActionRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.StartPilot(ctx, actor, req.GetId())
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// ConfirmPromotion 确认推广。
func (s *Server) ConfirmPromotion(ctx context.Context, req *fleetlogv1.ActivityActionRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.ConfirmPromotion(ctx, actor, req.GetId())
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// StopActivity 停止活动未发布部分。
func (s *Server) StopActivity(ctx context.Context, req *fleetlogv1.StopActivityRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.StopActivity(ctx, actor, dto.StopActivityCommand{ID: req.GetId(), Reason: req.GetReason()})
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// ApproveRestore 负责人批准恢复。
func (s *Server) ApproveRestore(ctx context.Context, req *fleetlogv1.ActivityActionRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.ApproveRestore(ctx, actor, req.GetId())
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// StartRestore 运营发起恢复下发。
func (s *Server) StartRestore(ctx context.Context, req *fleetlogv1.ActivityActionRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.StartRestore(ctx, actor, req.GetId())
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// MarkCompleted 负责人标记活动完成。
func (s *Server) MarkCompleted(ctx context.Context, req *fleetlogv1.ActivityActionRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.MarkCompleted(ctx, actor, req.GetId())
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// MarkRestored 负责人标记恢复完成。
func (s *Server) MarkRestored(ctx context.Context, req *fleetlogv1.ActivityActionRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.MarkRestored(ctx, actor, req.GetId())
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// GetActivity 返回活动详情与逐车核查数据。
func (s *Server) GetActivity(ctx context.Context, req *fleetlogv1.GetActivityRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.GetActivityDetail(ctx, actor, req.GetId())
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// GetCurrentActivity 返回当前未结束活动。
func (s *Server) GetCurrentActivity(ctx context.Context, _ *fleetlogv1.CurrentActivityRequest) (*fleetlogv1.ActivityDetailResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	detail, err := s.svc.GetCurrentActivity(ctx, actor)
	if err != nil {
		return nil, err
	}
	return toActivityDetail(detail), nil
}

// ListActivities 返回全部活动。
func (s *Server) ListActivities(ctx context.Context, _ *fleetlogv1.ListActivitiesRequest) (*fleetlogv1.ListActivitiesResponse, error) {
	if _, err := s.auth.actor(ctx); err != nil {
		return nil, err
	}
	items, err := s.svc.ListActivities(ctx)
	if err != nil {
		return nil, err
	}
	response := &fleetlogv1.ListActivitiesResponse{Items: make([]*fleetlogv1.ActivityItem, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toActivityItem(item))
	}
	return response, nil
}

// ListVehicles 返回车辆期望值与生效值。
func (s *Server) ListVehicles(ctx context.Context, _ *fleetlogv1.ListVehiclesRequest) (*fleetlogv1.ListVehiclesResponse, error) {
	if _, err := s.auth.actor(ctx); err != nil {
		return nil, err
	}
	items, err := s.svc.ListVehicles(ctx)
	if err != nil {
		return nil, err
	}
	response := &fleetlogv1.ListVehiclesResponse{Items: make([]*fleetlogv1.VehicleItem, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toVehicleItem(item))
	}
	return response, nil
}

// SetVehicleOnline 切换模拟车辆在线状态。
func (s *Server) SetVehicleOnline(ctx context.Context, req *fleetlogv1.SetVehicleOnlineRequest) (*fleetlogv1.VehicleResponse, error) {
	actor, err := s.auth.actor(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.svc.SetVehicleOnline(ctx, actor, req.GetId(), req.GetOnline())
	if err != nil {
		return nil, err
	}
	return &fleetlogv1.VehicleResponse{Vehicle: toVehicleItem(item)}, nil
}

// PullConfig 处理车辆拉取。
func (s *Server) PullConfig(ctx context.Context, req *fleetlogv1.PullConfigRequest) (*fleetlogv1.PullConfigResponse, error) {
	vehicleID, err := s.auth.vehicle(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.svc.PullConfig(ctx, vehicleID, dto.PullConfigCommand{VehicleID: req.GetVehicleId()})
	if err != nil {
		return nil, err
	}
	return &fleetlogv1.PullConfigResponse{
		VehicleId:       result.VehicleID,
		Version:         int32(result.Version),
		IntervalSeconds: int32(result.Interval),
		Online:          result.Online,
		PulledAt:        result.PulledAt,
		ReleaseStatus:   result.Status,
		HasRelease:      result.HasRelease,
	}, nil
}

// ReportReceipt 处理车端回执判定。
func (s *Server) ReportReceipt(ctx context.Context, req *fleetlogv1.ReportReceiptRequest) (*fleetlogv1.ReceiptResultResponse, error) {
	vehicleID, err := s.auth.vehicle(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.svc.ReportReceipt(ctx, vehicleID, dto.ReportReceiptCommand{
		VehicleID:     req.GetVehicleId(),
		Version:       int(req.GetVersion()),
		Result:        req.GetResult(),
		FailureReason: req.GetFailureReason(),
	})
	if err != nil {
		return nil, err
	}
	return &fleetlogv1.ReceiptResultResponse{
		Accepted:          result.Accepted,
		Reason:            result.Reason,
		Message:           result.Message,
		VehicleId:         result.VehicleID,
		Version:           int32(result.Version),
		ReleaseStatus:     result.ReleaseStatus,
		ActivityStatus:    result.ActivityStatus,
		EffectiveInterval: int32(result.EffectiveInterval),
		EffectiveVersion:  int32(result.EffectiveVersion),
	}, nil
}

// ListRejectedReceipts 返回被拒回执。
func (s *Server) ListRejectedReceipts(ctx context.Context, req *fleetlogv1.ListRejectedReceiptsRequest) (*fleetlogv1.ListRejectedReceiptsResponse, error) {
	if _, err := s.auth.actor(ctx); err != nil {
		return nil, err
	}
	items, err := s.svc.ListRejectedReceipts(ctx, req.GetActivityId(), req.GetVehicleId())
	if err != nil {
		return nil, err
	}
	response := &fleetlogv1.ListRejectedReceiptsResponse{Items: make([]*fleetlogv1.RejectedReceiptItem, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toRejectedReceiptItem(item))
	}
	return response, nil
}

// ListAudits 返回审计记录。
func (s *Server) ListAudits(ctx context.Context, req *fleetlogv1.ListAuditsRequest) (*fleetlogv1.ListAuditsResponse, error) {
	if _, err := s.auth.actor(ctx); err != nil {
		return nil, err
	}
	items, err := s.svc.ListAudits(ctx, req.GetActivityId(), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	response := &fleetlogv1.ListAuditsResponse{Items: make([]*fleetlogv1.AuditItem, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toAuditItem(item))
	}
	return response, nil
}
