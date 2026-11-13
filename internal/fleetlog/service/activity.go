package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

type stopScopeItem struct {
	VehicleID string `json:"vehicle_id"`
	Version   int    `json:"version"`
	Batch     string `json:"batch"`
	Status    string `json:"status"`
}

type stopScopeSnapshot struct {
	Released    []stopScopeItem `json:"released"`
	NotReleased []string        `json:"not_released"`
	Reason      string          `json:"reason,omitempty"`
}

// CreateActivity 由已批准申请创建发布活动并选定 2 台试点车。
func (s *Service) CreateActivity(ctx context.Context, actor entity.Actor, cmd dto.CreateActivityCommand) (*dto.ActivityDetailDTO, error) {
	if err := requireRole(actor, entity.RoleOperator); err != nil {
		return nil, toAppError(err)
	}
	var activityID string
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		application, err := tx.FindApplication(ctx, cmd.ApplicationID)
		if err != nil {
			return err
		}
		if application.Status != entity.AppStatusApproved {
			return entity.ErrApplicationNotApproved
		}
		vehicles, err := tx.ListVehicles(ctx)
		if err != nil {
			return err
		}
		pilots, promotion, err := splitPilotVehicles(vehicles, cmd.PilotVehicles)
		if err != nil {
			return err
		}
		state, err := tx.GetFleetStateForUpdate(ctx)
		if err != nil {
			return err
		}
		if state.OpenActivityID != "" {
			existing, findErr := tx.FindActivity(ctx, state.OpenActivityID)
			if findErr == nil && existing.IsOpen() {
				return entity.ErrOpenActivity
			}
			state.OpenActivityID = ""
		}
		now := s.now()
		activity := entity.NewActivity(application, pilots, promotion, actor, now)
		if err := tx.SaveActivity(ctx, activity); err != nil {
			return err
		}
		for _, vehicle := range vehicles {
			baseline := &entity.Baseline{
				ActivityID: activity.ID,
				VehicleID:  vehicle.ID,
				Interval:   vehicle.EffectiveInterval,
				Version:    vehicle.EffectiveVersion,
				CreatedAt:  now,
			}
			if err := tx.SaveBaseline(ctx, baseline); err != nil {
				return err
			}
		}
		if err := application.Void(activity.ID, now); err != nil {
			return err
		}
		if err := tx.SaveApplication(ctx, application); err != nil {
			return err
		}
		state.OpenActivityID = activity.ID
		state.UpdatedAt = now
		if err := tx.SaveFleetState(ctx, state); err != nil {
			return err
		}
		if err := s.audit(ctx, tx, actor, "activity_created", "activity", activity.ID,
			fmt.Sprintf("由申请 %s 创建活动，试点 %s，推广 %s", application.ID, strings.Join(pilots, "、"), strings.Join(promotion, "、"))); err != nil {
			return err
		}
		activityID = activity.ID
		return nil
	})
	if err != nil {
		return nil, toAppError(err)
	}
	s.log.Info("fleetlog activity created", zap.String("activity_id", activityID), zap.String("actor", actor.ID))
	return s.GetActivityDetail(ctx, actor, activityID)
}

// StartPilot 仅向 2 台试点车下发配置。
func (s *Service) StartPilot(ctx context.Context, actor entity.Actor, activityID string) (*dto.ActivityDetailDTO, error) {
	if err := requireRole(actor, entity.RoleOperator); err != nil {
		return nil, toAppError(err)
	}
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		activity, err := tx.FindActivity(ctx, activityID)
		if err != nil {
			return err
		}
		if activity.Status == entity.ActivityPiloting {
			return nil
		}
		if activity.Status != entity.ActivityPilotPending {
			return entity.ErrConflict
		}
		baselines, err := tx.ListBaselines(ctx, activity.ID)
		if err != nil {
			return err
		}
		baselineMap := baselineByVehicle(baselines)
		now := s.now()
		for _, vehicleID := range activity.PilotVehicles {
			baseline := baselineInterval(baselineMap, vehicleID)
			if _, err := s.allocate(ctx, tx, activity, vehicleID, entity.BatchPilot, activity.Interval, baseline, now); err != nil {
				return err
			}
		}
		if err := activity.MarkPiloting(now); err != nil {
			return err
		}
		if err := tx.SaveActivity(ctx, activity); err != nil {
			return err
		}
		return s.audit(ctx, tx, actor, "pilot_started", "activity", activity.ID,
			fmt.Sprintf("试点下发 %s，期望间隔 %d 秒", strings.Join(activity.PilotVehicles, "、"), activity.Interval))
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return s.GetActivityDetail(ctx, actor, activityID)
}

// ConfirmPromotion 在试点全部成功后向剩余 3 台车推广。
func (s *Service) ConfirmPromotion(ctx context.Context, actor entity.Actor, activityID string) (*dto.ActivityDetailDTO, error) {
	if err := requireRole(actor, entity.RoleOperator); err != nil {
		return nil, toAppError(err)
	}
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		activity, err := tx.FindActivity(ctx, activityID)
		if err != nil {
			return err
		}
		if activity.Status == entity.ActivityPromoting {
			return nil
		}
		if activity.Status != entity.ActivityPilotPassed {
			return entity.ErrPilotNotPassed
		}
		passed, _ := s.pilotsSucceeded(ctx, tx, activity)
		if !passed {
			return entity.ErrPilotNotPassed
		}
		baselines, err := tx.ListBaselines(ctx, activity.ID)
		if err != nil {
			return err
		}
		baselineMap := baselineByVehicle(baselines)
		now := s.now()
		for _, vehicleID := range activity.PromotionVehicles {
			baseline := baselineInterval(baselineMap, vehicleID)
			if _, err := s.allocate(ctx, tx, activity, vehicleID, entity.BatchPromotion, activity.Interval, baseline, now); err != nil {
				return err
			}
		}
		if err := activity.MarkPromoting(now); err != nil {
			return err
		}
		if err := tx.SaveActivity(ctx, activity); err != nil {
			return err
		}
		return s.audit(ctx, tx, actor, "promotion_started", "activity", activity.ID,
			fmt.Sprintf("推广下发 %s", strings.Join(activity.PromotionVehicles, "、")))
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return s.GetActivityDetail(ctx, actor, activityID)
}

// StopActivity 停止未发布部分，保留已发与已生效记录。
func (s *Service) StopActivity(ctx context.Context, actor entity.Actor, cmd dto.StopActivityCommand) (*dto.ActivityDetailDTO, error) {
	if err := requireAnyRole(actor, entity.RoleApprover, entity.RoleOperator); err != nil {
		return nil, toAppError(err)
	}
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		activity, err := tx.FindActivity(ctx, cmd.ID)
		if err != nil {
			return err
		}
		if activity.Status == entity.ActivityStopped {
			return nil
		}
		if !activity.CanStop() {
			return entity.ErrConflict
		}
		releases, err := tx.ListReleasesByActivity(ctx, activity.ID)
		if err != nil {
			return err
		}
		vehicles, err := tx.ListVehicles(ctx)
		if err != nil {
			return err
		}
		scope, releasedCount, notReleasedCount := buildStopScope(releases, vehicles, cmd.Reason)
		now := s.now()
		if err := activity.Stop(actor, scope, now); err != nil {
			return err
		}
		if err := tx.SaveActivity(ctx, activity); err != nil {
			return err
		}
		detail := fmt.Sprintf("停止活动，已发 %d 台，未发 %d 台", releasedCount, notReleasedCount)
		if strings.TrimSpace(cmd.Reason) != "" {
			detail += "；原因：" + strings.TrimSpace(cmd.Reason)
		}
		return s.audit(ctx, tx, actor, "activity_stopped", "activity", activity.ID, detail)
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return s.GetActivityDetail(ctx, actor, cmd.ID)
}

// ApproveRestore 由负责人批准恢复。
func (s *Service) ApproveRestore(ctx context.Context, actor entity.Actor, activityID string) (*dto.ActivityDetailDTO, error) {
	if err := requireRole(actor, entity.RoleApprover); err != nil {
		return nil, toAppError(err)
	}
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		activity, err := tx.FindActivity(ctx, activityID)
		if err != nil {
			return err
		}
		if activity.Status != entity.ActivityStopped {
			return entity.ErrConflict
		}
		releases, err := tx.ListReleasesByActivity(ctx, activity.ID)
		if err != nil {
			return err
		}
		if len(releases) == 0 {
			return entity.ErrInvalidInput
		}
		now := s.now()
		if err := activity.ApproveRestore(actor, now); err != nil {
			return err
		}
		if err := tx.SaveActivity(ctx, activity); err != nil {
			return err
		}
		return s.audit(ctx, tx, actor, "restore_approved", "activity", activity.ID, "批准恢复下发")
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return s.GetActivityDetail(ctx, actor, activityID)
}

// StartRestore 为曾发布过配置的车辆下发更高的恢复版本，目标为活动前基线值。
func (s *Service) StartRestore(ctx context.Context, actor entity.Actor, activityID string) (*dto.ActivityDetailDTO, error) {
	if err := requireRole(actor, entity.RoleOperator); err != nil {
		return nil, toAppError(err)
	}
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		activity, err := tx.FindActivity(ctx, activityID)
		if err != nil {
			return err
		}
		if activity.Status == entity.ActivityRestoring {
			return nil
		}
		if activity.Status != entity.ActivityStopped {
			return entity.ErrConflict
		}
		if !activity.RestoreApproved() {
			return entity.ErrForbidden
		}
		releases, err := tx.ListReleasesByActivity(ctx, activity.ID)
		if err != nil {
			return err
		}
		targets := releasedVehicles(releases)
		if len(targets) == 0 {
			return entity.ErrInvalidInput
		}
		baselines, err := tx.ListBaselines(ctx, activity.ID)
		if err != nil {
			return err
		}
		baselineMap := baselineByVehicle(baselines)
		now := s.now()
		for _, vehicleID := range targets {
			baseline := baselineInterval(baselineMap, vehicleID)
			if _, err := s.allocate(ctx, tx, activity, vehicleID, entity.BatchRestore, baseline, baseline, now); err != nil {
				return err
			}
		}
		if err := activity.MarkRestoring(actor, now); err != nil {
			return err
		}
		if err := tx.SaveActivity(ctx, activity); err != nil {
			return err
		}
		return s.audit(ctx, tx, actor, "restore_started", "activity", activity.ID,
			fmt.Sprintf("恢复下发 %s，目标为活动前基线值", strings.Join(targets, "、")))
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return s.GetActivityDetail(ctx, actor, activityID)
}

// MarkCompleted 负责人在 5 台车当前期望版本全部成功后标记活动完成。
func (s *Service) MarkCompleted(ctx context.Context, actor entity.Actor, activityID string) (*dto.ActivityDetailDTO, error) {
	if err := requireRole(actor, entity.RoleApprover); err != nil {
		return nil, toAppError(err)
	}
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		activity, err := tx.FindActivity(ctx, activityID)
		if err != nil {
			return err
		}
		if activity.Status != entity.ActivityPromoting {
			return entity.ErrConflict
		}
		vehicles, err := tx.ListVehicles(ctx)
		if err != nil {
			return err
		}
		releases, err := tx.ListReleasesByActivity(ctx, activity.ID)
		if err != nil {
			return err
		}
		latest := latestByVehicle(releases)
		if pending := unconfirmedVehicles(vehicles, latest); len(pending) > 0 {
			return fmt.Errorf("%w: %s", entity.ErrUnconfirmedVehicles, strings.Join(pending, "、"))
		}
		now := s.now()
		if err := activity.MarkCompleted(actor, now); err != nil {
			return err
		}
		if err := tx.SaveActivity(ctx, activity); err != nil {
			return err
		}
		if err := s.releaseOpenActivity(ctx, tx, activity.ID, now); err != nil {
			return err
		}
		return s.audit(ctx, tx, actor, "activity_completed", "activity", activity.ID, "5 台车均确认当前期望版本成功，标记活动完成")
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return s.GetActivityDetail(ctx, actor, activityID)
}

// MarkRestored 负责人在全部需恢复车辆确认后标记恢复完成。
func (s *Service) MarkRestored(ctx context.Context, actor entity.Actor, activityID string) (*dto.ActivityDetailDTO, error) {
	if err := requireRole(actor, entity.RoleApprover); err != nil {
		return nil, toAppError(err)
	}
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		activity, err := tx.FindActivity(ctx, activityID)
		if err != nil {
			return err
		}
		if activity.Status != entity.ActivityRestoring {
			return entity.ErrConflict
		}
		vehicles, err := tx.ListVehicles(ctx)
		if err != nil {
			return err
		}
		releases, err := tx.ListReleasesByActivity(ctx, activity.ID)
		if err != nil {
			return err
		}
		vehicleMap := make(map[string]*entity.Vehicle, len(vehicles))
		for _, vehicle := range vehicles {
			vehicleMap[vehicle.ID] = vehicle
		}
		latest := latestByVehicle(releases)
		pending := make([]string, 0)
		for vehicleID, release := range latest {
			vehicle, ok := vehicleMap[vehicleID]
			if !ok {
				continue
			}
			if release.Batch != entity.BatchRestore || release.Status != entity.ReleaseSuccess || release.Version != vehicle.CurrentVersion {
				pending = append(pending, vehicleID)
			}
		}
		if len(pending) > 0 {
			sort.Strings(pending)
			return fmt.Errorf("%w: %s", entity.ErrUnconfirmedVehicles, strings.Join(pending, "、"))
		}
		now := s.now()
		if err := activity.MarkRestored(actor, now); err != nil {
			return err
		}
		if err := tx.SaveActivity(ctx, activity); err != nil {
			return err
		}
		if err := s.releaseOpenActivity(ctx, tx, activity.ID, now); err != nil {
			return err
		}
		return s.audit(ctx, tx, actor, "restore_completed", "activity", activity.ID, "需恢复车辆全部确认恢复，标记恢复完成")
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return s.GetActivityDetail(ctx, actor, activityID)
}

func (s *Service) allocate(ctx context.Context, repo entity.Repository, activity *entity.Activity, vehicleID string, batch string, expected int, baseline int, now time.Time) (*entity.Release, error) {
	vehicle, err := repo.FindVehicleForUpdate(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	release := entity.NewRelease(activity.ID, vehicleID, vehicle.CurrentVersion+1, batch, expected, baseline, now)
	vehicle.CurrentVersion = release.Version
	vehicle.UpdatedAt = now
	if err := repo.SaveVehicle(ctx, vehicle); err != nil {
		return nil, err
	}
	if err := repo.SaveRelease(ctx, release); err != nil {
		return nil, err
	}
	return release, nil
}

func (s *Service) pilotsSucceeded(ctx context.Context, repo entity.Repository, activity *entity.Activity) (bool, string) {
	releases, err := repo.ListReleasesByActivity(ctx, activity.ID)
	if err != nil {
		return false, ""
	}
	latest := latestByVehicle(releases)
	for _, vehicleID := range activity.PilotVehicles {
		vehicle, err := repo.FindVehicle(ctx, vehicleID)
		if err != nil {
			return false, vehicleID
		}
		release := latest[vehicleID]
		if release == nil || release.Version != vehicle.CurrentVersion || release.Status != entity.ReleaseSuccess {
			return false, vehicleID
		}
	}
	return true, ""
}

func (s *Service) releaseOpenActivity(ctx context.Context, repo entity.Repository, activityID string, now time.Time) error {
	state, err := repo.GetFleetStateForUpdate(ctx)
	if err != nil {
		return err
	}
	if state.OpenActivityID != activityID {
		return nil
	}
	state.OpenActivityID = ""
	state.UpdatedAt = now
	return repo.SaveFleetState(ctx, state)
}

func baselineInterval(baselines map[string]*entity.Baseline, vehicleID string) int {
	if baseline, ok := baselines[vehicleID]; ok {
		return baseline.Interval
	}
	return entity.DefaultInterval
}

func splitPilotVehicles(vehicles []*entity.Vehicle, pilots []string) ([]string, []string, error) {
	if len(pilots) != entity.PilotVehicleCount {
		return nil, nil, entity.ErrInvalidPilotSelection
	}
	known := make(map[string]bool, len(vehicles))
	for _, vehicle := range vehicles {
		known[vehicle.ID] = true
	}
	selected := make(map[string]bool, len(pilots))
	for _, vehicleID := range pilots {
		if !known[vehicleID] || selected[vehicleID] {
			return nil, nil, entity.ErrInvalidPilotSelection
		}
		selected[vehicleID] = true
	}
	promotion := make([]string, 0, len(vehicles)-len(pilots))
	for _, vehicle := range vehicles {
		if !selected[vehicle.ID] {
			promotion = append(promotion, vehicle.ID)
		}
	}
	ordered := append([]string{}, pilots...)
	return ordered, promotion, nil
}

func releasedVehicles(releases []*entity.Release) []string {
	seen := make(map[string]bool, len(releases))
	output := make([]string, 0, len(releases))
	for _, release := range releases {
		if seen[release.VehicleID] {
			continue
		}
		seen[release.VehicleID] = true
		output = append(output, release.VehicleID)
	}
	sort.Strings(output)
	return output
}

func unconfirmedVehicles(vehicles []*entity.Vehicle, latest map[string]*entity.Release) []string {
	pending := make([]string, 0)
	for _, vehicle := range vehicles {
		release := latest[vehicle.ID]
		if release == nil || release.Version != vehicle.CurrentVersion || release.Status != entity.ReleaseSuccess {
			pending = append(pending, vehicle.ID)
		}
	}
	return pending
}

func buildStopScope(releases []*entity.Release, vehicles []*entity.Vehicle, reason string) (string, int, int) {
	latest := latestByVehicle(releases)
	scope := stopScopeSnapshot{Released: make([]stopScopeItem, 0, len(latest)), NotReleased: make([]string, 0), Reason: strings.TrimSpace(reason)}
	released := make(map[string]bool, len(latest))
	for vehicleID, release := range latest {
		released[vehicleID] = true
		scope.Released = append(scope.Released, stopScopeItem{
			VehicleID: vehicleID,
			Version:   release.Version,
			Batch:     release.Batch,
			Status:    release.Status,
		})
	}
	sort.Slice(scope.Released, func(i, j int) bool { return scope.Released[i].VehicleID < scope.Released[j].VehicleID })
	for _, vehicle := range vehicles {
		if !released[vehicle.ID] {
			scope.NotReleased = append(scope.NotReleased, vehicle.ID)
		}
	}
	sort.Strings(scope.NotReleased)
	raw, err := json.Marshal(scope)
	if err != nil {
		return "", len(scope.Released), len(scope.NotReleased)
	}
	return string(raw), len(scope.Released), len(scope.NotReleased)
}
