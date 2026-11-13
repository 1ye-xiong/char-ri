package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
)

// PullConfig 返回该车当前最新已发布版本；从未发布过时返回初始 v1/60 秒。
func (s *Service) PullConfig(ctx context.Context, tokenVehicleID string, cmd dto.PullConfigCommand) (*dto.PullResultDTO, error) {
	if strings.TrimSpace(tokenVehicleID) == "" {
		return nil, apperrors.Unauthorized("fleetlog_vehicle_token_required", "缺少车辆凭证")
	}
	claimed := strings.TrimSpace(cmd.VehicleID)
	if claimed != "" && claimed != tokenVehicleID {
		return nil, apperrors.Unauthorized("fleetlog_vehicle_mismatch", "车辆凭证与请求车辆不一致")
	}
	vehicle, err := s.repo.FindVehicle(ctx, tokenVehicleID)
	if err != nil {
		return nil, toAppError(err)
	}
	if !vehicle.Online {
		return nil, apperrors.Conflict("fleetlog_vehicle_offline", "车辆离线，无法拉取配置")
	}
	latestMap, err := s.repo.ListLatestReleasesByVehicle(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	latest := latestMap[vehicle.ID]
	version := entity.DefaultVehicleVersion
	interval := entity.DefaultInterval
	releaseID := ""
	status := ""
	if latest != nil {
		version = latest.Version
		interval = latest.ExpectedInterval
		releaseID = latest.ID
		status = latest.Status
	}
	now := s.now()
	pull := entity.NewPullRecord(vehicle.ID, version, interval, releaseID, now)
	if err := s.repo.SavePull(ctx, pull); err != nil {
		return nil, toAppError(err)
	}
	vehicleActor := entity.Actor{ID: vehicle.ID, Name: vehicle.ID, Role: entity.RoleVehicle}
	if err := s.audit(ctx, s.repo, vehicleActor, "config_pulled", "vehicle", vehicle.ID,
		fmt.Sprintf("拉取配置 v%d / %d 秒", version, interval)); err != nil {
		return nil, toAppError(err)
	}
	return &dto.PullResultDTO{
		VehicleID:  vehicle.ID,
		Version:    version,
		Interval:   interval,
		Online:     vehicle.Online,
		PulledAt:   dto.FormatTime(now),
		Status:     status,
		HasRelease: latest != nil,
	}, nil
}

// ReportReceipt 按 5.5 节判定顺序处理车端回执。
func (s *Service) ReportReceipt(ctx context.Context, tokenVehicleID string, cmd dto.ReportReceiptCommand) (*dto.ReceiptResultDTO, error) {
	if strings.TrimSpace(tokenVehicleID) == "" {
		return nil, apperrors.Unauthorized("fleetlog_vehicle_token_required", "缺少车辆凭证")
	}
	now := s.now()
	claimed := strings.TrimSpace(cmd.VehicleID)
	if claimed == "" {
		claimed = tokenVehicleID
	}
	if claimed != tokenVehicleID {
		rejected := entity.NewRejectedReceipt(claimed, cmd.Version, entity.RejectCrossVehicle,
			fmt.Sprintf("凭证车辆 %s 上报了 %s 的回执", tokenVehicleID, claimed), now)
		if err := s.repo.SaveRejectedReceipt(ctx, rejected); err != nil {
			return nil, toAppError(err)
		}
		return rejectedResult(claimed, cmd.Version, rejected.Reason, "跨车回执已拒绝并记录"), nil
	}
	if reason := invalidPayloadReason(cmd); reason != "" {
		rejected := entity.NewRejectedReceipt(tokenVehicleID, cmd.Version, entity.RejectInvalidPayload, reason, now)
		if err := s.repo.SaveRejectedReceipt(ctx, rejected); err != nil {
			return nil, toAppError(err)
		}
		return rejectedResult(tokenVehicleID, cmd.Version, rejected.Reason, "回执内容不完整："+reason), nil
	}

	var output *dto.ReceiptResultDTO
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		vehicle, err := tx.FindVehicleForUpdate(ctx, tokenVehicleID)
		if err != nil {
			return err
		}
		release, err := tx.FindRelease(ctx, tokenVehicleID, cmd.Version)
		if errors.Is(err, entity.ErrNotFound) {
			rejected := entity.NewRejectedReceipt(tokenVehicleID, cmd.Version, entity.RejectUnknownVersion, "该车从未发布过此版本", now)
			if err := tx.SaveRejectedReceipt(ctx, rejected); err != nil {
				return err
			}
			output = rejectedResult(tokenVehicleID, cmd.Version, rejected.Reason, "未知版本回执已拒绝并记录")
			return nil
		}
		if err != nil {
			return err
		}
		if cmd.Version < vehicle.CurrentVersion {
			rejected := entity.NewRejectedReceipt(tokenVehicleID, cmd.Version, entity.RejectStaleVersion,
				fmt.Sprintf("当前期望版本 v%d，旧版本回执不能覆盖新生效值", vehicle.CurrentVersion), now)
			rejected.ActivityID = release.ActivityID
			if err := tx.SaveRejectedReceipt(ctx, rejected); err != nil {
				return err
			}
			output = rejectedResult(tokenVehicleID, cmd.Version, rejected.Reason, "过期回执已拒绝并记录")
			return nil
		}
		if release.Status == entity.ReleaseSuccess {
			release.Duplicate()
			if err := tx.SaveRelease(ctx, release); err != nil {
				return err
			}
			rejected := entity.NewRejectedReceipt(tokenVehicleID, cmd.Version, entity.RejectDuplicate,
				"当前版本已有被接受的成功回执，不重复记账", now)
			rejected.ActivityID = release.ActivityID
			if err := tx.SaveRejectedReceipt(ctx, rejected); err != nil {
				return err
			}
			output = &dto.ReceiptResultDTO{
				Accepted:          false,
				Reason:            entity.RejectDuplicate,
				Message:           "重复回执已记录，未重复记账",
				VehicleID:         tokenVehicleID,
				Version:           cmd.Version,
				ReleaseStatus:     release.Status,
				EffectiveInterval: vehicle.EffectiveInterval,
				EffectiveVersion:  vehicle.EffectiveVersion,
			}
			return nil
		}

		if cmd.Result == entity.ReceiptFailure {
			reason := strings.TrimSpace(cmd.FailureReason)
			release.MarkFailure(reason, now)
			if err := tx.SaveRelease(ctx, release); err != nil {
				return err
			}
			if err := tx.SaveReceipt(ctx, entity.NewReceipt(release, entity.ReceiptFailure, reason, now)); err != nil {
				return err
			}
			if err := s.audit(ctx, tx, vehicleActor(vehicle), "receipt_failure", "release", release.ID,
				fmt.Sprintf("v%d 应用失败：%s", release.Version, reason)); err != nil {
				return err
			}
		} else {
			release.MarkSuccess(now)
			vehicle.EffectiveInterval = release.ExpectedInterval
			vehicle.EffectiveVersion = release.Version
			vehicle.UpdatedAt = now
			if err := tx.SaveVehicle(ctx, vehicle); err != nil {
				return err
			}
			if err := tx.SaveRelease(ctx, release); err != nil {
				return err
			}
			if err := tx.SaveReceipt(ctx, entity.NewReceipt(release, entity.ReceiptSuccess, "", now)); err != nil {
				return err
			}
			if err := tx.ResolveTimeoutEvents(ctx, release.ID, now); err != nil {
				return err
			}
			if err := s.audit(ctx, tx, vehicleActor(vehicle), "receipt_success", "release", release.ID,
				fmt.Sprintf("v%d 应用成功，生效值 %d 秒", release.Version, release.ExpectedInterval)); err != nil {
				return err
			}
		}

		activityStatus := ""
		if activity, err := tx.FindActivity(ctx, release.ActivityID); err == nil {
			if activity.Status == entity.ActivityPiloting {
				if passed, _ := s.pilotsSucceeded(ctx, tx, activity); passed {
					if err := activity.MarkPilotPassed(now); err != nil {
						return err
					}
					if err := tx.SaveActivity(ctx, activity); err != nil {
						return err
					}
					if err := s.audit(ctx, tx, entity.Actor{ID: "system", Name: "system"}, "pilot_passed", "activity", activity.ID,
						"2 台试点车当前版本均成功，试点通过"); err != nil {
						return err
					}
				}
			}
			activityStatus = activity.Status
		}

		message := "成功回执已接受并更新生效值"
		if release.Status == entity.ReleaseFailed {
			message = "失败回执已记录，生效值保持不变"
		}
		output = &dto.ReceiptResultDTO{
			Accepted:          true,
			Message:           message,
			VehicleID:         tokenVehicleID,
			Version:           cmd.Version,
			ReleaseStatus:     release.Status,
			ActivityStatus:    activityStatus,
			EffectiveInterval: vehicle.EffectiveInterval,
			EffectiveVersion:  vehicle.EffectiveVersion,
		}
		return nil
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return output, nil
}

// SweepTimeouts 扫描所有等待中的发布记录，把超过阈值的置为超时并写入超时事件。
func (s *Service) SweepTimeouts(ctx context.Context) (int, error) {
	now := s.now()
	releases, err := s.repo.ListWaitingReleasesBefore(ctx, now.Add(-s.timeout))
	if err != nil {
		return 0, toAppError(err)
	}
	seen := make(map[string]bool, len(releases))
	count := 0
	for _, release := range releases {
		if seen[release.ID] {
			continue
		}
		seen[release.ID] = true
		marked, err := s.markTimeout(ctx, release)
		if err != nil {
			return count, err
		}
		if marked {
			count++
		}
	}
	return count, nil
}

func (s *Service) refreshTimeouts(ctx context.Context, activityID string) {
	releases, err := s.repo.ListReleasesByActivity(ctx, activityID)
	if err != nil {
		return
	}
	now := s.now()
	for _, release := range releases {
		if !release.TimedOut(now, s.timeout) {
			continue
		}
		if _, err := s.markTimeout(ctx, release); err != nil {
			s.log.Warn("refresh timeout failed", zap.String("release_id", release.ID), zap.Error(err))
		}
	}
}

func (s *Service) markTimeout(ctx context.Context, release *entity.Release) (bool, error) {
	marked := false
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		current, err := tx.FindRelease(ctx, release.VehicleID, release.Version)
		if err != nil {
			return err
		}
		if !current.TimedOut(s.now(), s.timeout) {
			return nil
		}
		at := s.now()
		if !current.MarkTimeout(at) {
			return nil
		}
		if err := tx.SaveRelease(ctx, current); err != nil {
			return err
		}
		if err := tx.SaveTimeoutEvent(ctx, entity.NewTimeoutEvent(current, at)); err != nil {
			return err
		}
		if err := s.audit(ctx, tx, entity.Actor{ID: "system", Name: "system"}, "release_timeout", "release", current.ID,
			fmt.Sprintf("%s v%d 超时未收到有效回执", current.VehicleID, current.Version)); err != nil {
			return err
		}
		marked = true
		return nil
	})
	if err != nil {
		return false, toAppError(err)
	}
	return marked, nil
}

// SetVehicleOnline 由运营切换模拟车辆在线状态。
func (s *Service) SetVehicleOnline(ctx context.Context, actor entity.Actor, vehicleID string, online bool) (*dto.VehicleDTO, error) {
	if err := requireRole(actor, entity.RoleOperator); err != nil {
		return nil, toAppError(err)
	}
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		vehicle, err := tx.FindVehicleForUpdate(ctx, vehicleID)
		if err != nil {
			return err
		}
		vehicle.Online = online
		vehicle.UpdatedAt = s.now()
		if err := tx.SaveVehicle(ctx, vehicle); err != nil {
			return err
		}
		state := "离线"
		if online {
			state = "在线"
		}
		return s.audit(ctx, tx, actor, "vehicle_online_changed", "vehicle", vehicle.ID, "切换为"+state)
	})
	if err != nil {
		return nil, toAppError(err)
	}
	vehicles, err := s.repo.ListVehicles(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	latestMap, err := s.repo.ListLatestReleasesByVehicle(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	for _, vehicle := range vehicles {
		if vehicle.ID == vehicleID {
			return dto.NewVehicleDTO(vehicle, latestMap[vehicle.ID]), nil
		}
	}
	return nil, apperrors.NotFound("fleetlog_vehicle_not_found", "车辆不存在")
}

func invalidPayloadReason(cmd dto.ReportReceiptCommand) string {
	if cmd.Version <= 0 {
		return "缺少版本号"
	}
	if cmd.Result != entity.ReceiptSuccess && cmd.Result != entity.ReceiptFailure {
		return "结果必须为 success 或 failure"
	}
	if cmd.Result == entity.ReceiptFailure && strings.TrimSpace(cmd.FailureReason) == "" {
		return "失败回执必须填写失败原因"
	}
	return ""
}

func rejectedResult(vehicleID string, version int, reason string, message string) *dto.ReceiptResultDTO {
	return &dto.ReceiptResultDTO{
		Accepted:      false,
		Reason:        reason,
		Message:       message,
		VehicleID:     vehicleID,
		Version:       version,
		ReleaseStatus: "",
	}
}

func vehicleActor(vehicle *entity.Vehicle) entity.Actor {
	return entity.Actor{ID: vehicle.ID, Name: vehicle.ID, Role: entity.RoleVehicle}
}
