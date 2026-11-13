package service

import (
	"context"
	"sort"
	"strings"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

// GetActivityDetail 汇总活动看板、逐车核查、被拒回执、超时历史与审计。
func (s *Service) GetActivityDetail(ctx context.Context, actor entity.Actor, activityID string) (*dto.ActivityDetailDTO, error) {
	s.refreshTimeouts(ctx, activityID)
	activity, err := s.repo.FindActivity(ctx, activityID)
	if err != nil {
		return nil, toAppError(err)
	}
	vehicles, err := s.repo.ListVehicles(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	releases, err := s.repo.ListReleasesByActivity(ctx, activity.ID)
	if err != nil {
		return nil, toAppError(err)
	}
	baselines, err := s.repo.ListBaselines(ctx, activity.ID)
	if err != nil {
		return nil, toAppError(err)
	}
	overallLatest, err := s.repo.ListLatestReleasesByVehicle(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	rejected, err := s.repo.ListRejectedReceipts(ctx, activity.ID, "")
	if err != nil {
		return nil, toAppError(err)
	}
	timeouts, err := s.repo.ListTimeoutEventsByActivity(ctx, activity.ID)
	if err != nil {
		return nil, toAppError(err)
	}
	audits, err := s.repo.ListAudits(ctx, activity.ID, 200)
	if err != nil {
		return nil, toAppError(err)
	}

	latest := latestByVehicle(releases)
	baselineMap := baselineByVehicle(baselines)
	sort.Slice(vehicles, func(i, j int) bool { return vehicles[i].ID < vehicles[j].ID })

	rows := make([]*dto.VehicleRowDTO, 0, len(vehicles))
	summary := &dto.SummaryDTO{Total: len(vehicles)}
	for _, vehicle := range vehicles {
		row := &dto.VehicleRowDTO{
			VehicleID:         vehicle.ID,
			BaselineInterval:  entity.DefaultInterval,
			ExpectedInterval:  vehicle.ExpectedInterval(overallLatest[vehicle.ID]),
			EffectiveInterval: vehicle.EffectiveInterval,
			EffectiveVersion:  vehicle.EffectiveVersion,
		}
		if baseline, ok := baselineMap[vehicle.ID]; ok {
			row.BaselineInterval = baseline.Interval
			row.ExpectedInterval = baseline.Interval
		}
		release := latest[vehicle.ID]
		if release == nil {
			row.Status = entity.StatusNotReleased
			summary.NotReleased++
		} else {
			row.Released = true
			row.Batch = release.Batch
			row.Version = release.Version
			row.ExpectedInterval = release.ExpectedInterval
			row.BaselineInterval = release.BaselineInterval
			row.Status = release.Status
			row.FailureReason = release.FailureReason
			row.DispatchedAt = dto.FormatTime(release.DispatchedAt)
			row.ReceiptAt = dto.FormatTimePtr(release.ReceiptAt)
			row.TimeoutAt = dto.FormatTimePtr(release.TimeoutAt)
			row.Late = release.Late
			row.DuplicateReceipts = release.DuplicateReceipts
			summary.Released++
			switch release.Status {
			case entity.ReleaseSuccess:
				summary.Success++
				summary.Effective++
			case entity.ReleaseFailed:
				summary.Failed++
			case entity.ReleaseTimeout:
				summary.Timeout++
			case entity.ReleaseWaiting:
				summary.Waiting++
			}
		}
		rows = append(rows, row)
	}

	detail := &dto.ActivityDetailDTO{
		Activity:    dto.NewActivityDTO(activity),
		Rows:        rows,
		Summary:     summary,
		Rejected:    make([]*dto.RejectedReceiptDTO, 0, len(rejected)),
		Timeouts:    make([]*dto.TimeoutEventDTO, 0, len(timeouts)),
		Audits:      make([]*dto.AuditDTO, 0, len(audits)),
		Actions:     buildActions(activity, actor, vehicles, latest),
		TimeoutSecs: s.cfg.TimeoutSeconds,
		ServerTime:  dto.FormatTime(s.now()),
	}
	summary.RejectedReceipt = len(rejected)
	for _, item := range rejected {
		detail.Rejected = append(detail.Rejected, dto.NewRejectedReceiptDTO(item))
	}
	for _, item := range timeouts {
		detail.Timeouts = append(detail.Timeouts, dto.NewTimeoutEventDTO(item))
	}
	for _, item := range audits {
		detail.Audits = append(detail.Audits, dto.NewAuditDTO(item))
	}
	application, err := s.repo.FindApplication(ctx, activity.ApplicationID)
	if err == nil {
		view, err := s.applicationDTO(ctx, s.repo, application, actor)
		if err != nil {
			return nil, toAppError(err)
		}
		detail.Application = view
	}
	return detail, nil
}

// GetCurrentActivity 返回当前未结束活动，没有则返回 nil。
func (s *Service) GetCurrentActivity(ctx context.Context, actor entity.Actor) (*dto.ActivityDetailDTO, error) {
	items, err := s.repo.ListActivities(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	for _, activity := range items {
		if activity.IsOpen() {
			return s.GetActivityDetail(ctx, actor, activity.ID)
		}
	}
	return nil, nil
}

// ListActivities 返回全部活动（按创建时间倒序）。
func (s *Service) ListActivities(ctx context.Context) ([]*dto.ActivityDTO, error) {
	items, err := s.repo.ListActivities(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	output := make([]*dto.ActivityDTO, 0, len(items))
	for _, item := range items {
		output = append(output, dto.NewActivityDTO(item))
	}
	return output, nil
}

// ListVehicles 返回 5 台车的期望值与生效值。
func (s *Service) ListVehicles(ctx context.Context) ([]*dto.VehicleDTO, error) {
	vehicles, err := s.repo.ListVehicles(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	latest, err := s.repo.ListLatestReleasesByVehicle(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	sort.Slice(vehicles, func(i, j int) bool { return vehicles[i].ID < vehicles[j].ID })
	output := make([]*dto.VehicleDTO, 0, len(vehicles))
	for _, vehicle := range vehicles {
		output = append(output, dto.NewVehicleDTO(vehicle, latest[vehicle.ID]))
	}
	return output, nil
}

// ListRejectedReceipts 返回被拒回执，可按活动或车辆过滤。
func (s *Service) ListRejectedReceipts(ctx context.Context, activityID string, vehicleID string) ([]*dto.RejectedReceiptDTO, error) {
	items, err := s.repo.ListRejectedReceipts(ctx, activityID, vehicleID)
	if err != nil {
		return nil, toAppError(err)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ReportedAt.After(items[j].ReportedAt) })
	output := make([]*dto.RejectedReceiptDTO, 0, len(items))
	for _, item := range items {
		output = append(output, dto.NewRejectedReceiptDTO(item))
	}
	return output, nil
}

// ListAudits 返回审计记录，可按活动过滤。
func (s *Service) ListAudits(ctx context.Context, activityID string, limit int) ([]*dto.AuditDTO, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	items, err := s.repo.ListAudits(ctx, activityID, limit)
	if err != nil {
		return nil, toAppError(err)
	}
	output := make([]*dto.AuditDTO, 0, len(items))
	for _, item := range items {
		output = append(output, dto.NewAuditDTO(item))
	}
	return output, nil
}

func buildActions(activity *entity.Activity, actor entity.Actor, vehicles []*entity.Vehicle, latest map[string]*entity.Release) *dto.ActionsDTO {
	actions := &dto.ActionsDTO{
		CanStartPilot: actor.Is(entity.RoleOperator) && activity.Status == entity.ActivityPilotPending,
		CanStop:       (actor.Is(entity.RoleApprover) || actor.Is(entity.RoleOperator)) && activity.CanStop(),
	}
	pilotsOK := pilotsConfirmed(vehicles, latest, activity.PilotVehicles)
	if actor.Is(entity.RoleOperator) && activity.Status == entity.ActivityPilotPassed && pilotsOK {
		actions.CanPromote = true
	}
	if activity.Status == entity.ActivityPiloting && !pilotsOK {
		actions.PromoteBlockedReason = "试点未全部成功：" + pilotStatusSummary(activity.PilotVehicles, latest)
	}
	released := len(latest) > 0
	if actor.Is(entity.RoleApprover) && activity.Status == entity.ActivityStopped && released && !activity.RestoreApproved() {
		actions.CanApproveRestore = true
	}
	if actor.Is(entity.RoleOperator) && activity.Status == entity.ActivityStopped && activity.RestoreApproved() {
		actions.CanStartRestore = true
	}
	if actor.Is(entity.RoleApprover) && activity.Status == entity.ActivityPromoting && len(unconfirmedVehicles(vehicles, latest)) == 0 {
		actions.CanMarkCompleted = true
	}
	if actor.Is(entity.RoleApprover) && activity.Status == entity.ActivityRestoring && restoreConfirmed(vehicles, latest) {
		actions.CanMarkRestored = true
	}
	return actions
}

func pilotsConfirmed(vehicles []*entity.Vehicle, latest map[string]*entity.Release, pilots []string) bool {
	vehicleMap := make(map[string]*entity.Vehicle, len(vehicles))
	for _, vehicle := range vehicles {
		vehicleMap[vehicle.ID] = vehicle
	}
	for _, vehicleID := range pilots {
		vehicle := vehicleMap[vehicleID]
		release := latest[vehicleID]
		if vehicle == nil || release == nil || release.Version != vehicle.CurrentVersion || release.Status != entity.ReleaseSuccess {
			return false
		}
	}
	return true
}

func pilotStatusSummary(pilots []string, latest map[string]*entity.Release) string {
	parts := make([]string, 0, len(pilots))
	for _, vehicleID := range pilots {
		parts = append(parts, vehicleID+"="+releaseStatus(latest[vehicleID]))
	}
	return strings.Join(parts, "，")
}

func restoreConfirmed(vehicles []*entity.Vehicle, latest map[string]*entity.Release) bool {
	vehicleMap := make(map[string]*entity.Vehicle, len(vehicles))
	for _, vehicle := range vehicles {
		vehicleMap[vehicle.ID] = vehicle
	}
	for vehicleID, release := range latest {
		vehicle, ok := vehicleMap[vehicleID]
		if !ok {
			continue
		}
		if release.Batch != entity.BatchRestore || release.Status != entity.ReleaseSuccess || release.Version != vehicle.CurrentVersion {
			return false
		}
	}
	return len(latest) > 0
}
