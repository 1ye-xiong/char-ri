package service

import (
	"context"
	"sort"

	"go.uber.org/zap"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

// CreateApplication 创建申请草稿。
func (s *Service) CreateApplication(ctx context.Context, actor entity.Actor, cmd dto.CreateApplicationCommand) (*dto.ApplicationDTO, error) {
	if err := requireRole(actor, entity.RoleRequester); err != nil {
		return nil, toAppError(err)
	}
	now := s.now()
	application := entity.NewApplication(actor, cmd.Reason, cmd.Interval, now)
	if err := s.repo.SaveApplication(ctx, application); err != nil {
		return nil, toAppError(err)
	}
	if err := s.audit(ctx, s.repo, actor, "application_created", "application", application.ID, "创建申请草稿"); err != nil {
		return nil, toAppError(err)
	}
	s.log.Info("fleetlog application created", zap.String("application_id", application.ID), zap.String("actor", actor.ID))
	return s.applicationDTO(ctx, s.repo, application, actor)
}

// UpdateApplication 修改草稿或已退回申请。
func (s *Service) UpdateApplication(ctx context.Context, actor entity.Actor, cmd dto.UpdateApplicationCommand) (*dto.ApplicationDTO, error) {
	if err := requireRole(actor, entity.RoleRequester); err != nil {
		return nil, toAppError(err)
	}
	var output *dto.ApplicationDTO
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		application, err := tx.FindApplication(ctx, cmd.ID)
		if err != nil {
			return err
		}
		if err := application.Edit(actor, cmd.Reason, cmd.Interval, s.now()); err != nil {
			return err
		}
		if err := tx.SaveApplication(ctx, application); err != nil {
			return err
		}
		if err := s.audit(ctx, tx, actor, "application_updated", "application", application.ID, "修改申请内容"); err != nil {
			return err
		}
		output, err = s.applicationDTO(ctx, tx, application, actor)
		return err
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return output, nil
}

// SubmitApplication 提交草稿或退回申请；退回重提生成新版本并保留旧快照。
func (s *Service) SubmitApplication(ctx context.Context, actor entity.Actor, id string) (*dto.ApplicationDTO, error) {
	if err := requireRole(actor, entity.RoleRequester); err != nil {
		return nil, toAppError(err)
	}
	var output *dto.ApplicationDTO
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		application, err := tx.FindApplication(ctx, id)
		if err != nil {
			return err
		}
		if application.RequesterID != actor.ID {
			return entity.ErrNotRequester
		}
		now := s.now()
		if application.Status == entity.AppStatusRejected {
			previous := application.Snapshot(now)
			if err := tx.AppendApplicationSnapshot(ctx, previous); err != nil {
				return err
			}
		}
		if err := application.Submit(now); err != nil {
			return err
		}
		if err := tx.SaveApplication(ctx, application); err != nil {
			return err
		}
		if err := s.audit(ctx, tx, actor, "application_submitted", "application", application.ID, "提交申请审批"); err != nil {
			return err
		}
		output, err = s.applicationDTO(ctx, tx, application, actor)
		return err
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return output, nil
}

// ApproveApplication 负责人批准申请。
func (s *Service) ApproveApplication(ctx context.Context, actor entity.Actor, id string) (*dto.ApplicationDTO, error) {
	return s.decideApplication(ctx, actor, id, true, "")
}

// RejectApplication 负责人退回申请。
func (s *Service) RejectApplication(ctx context.Context, actor entity.Actor, id string, reason string) (*dto.ApplicationDTO, error) {
	return s.decideApplication(ctx, actor, id, false, reason)
}

func (s *Service) decideApplication(ctx context.Context, actor entity.Actor, id string, approve bool, reason string) (*dto.ApplicationDTO, error) {
	if err := requireRole(actor, entity.RoleApprover); err != nil {
		return nil, toAppError(err)
	}
	var output *dto.ApplicationDTO
	err := s.repo.Transaction(ctx, func(tx entity.Repository) error {
		application, err := tx.FindApplication(ctx, id)
		if err != nil {
			return err
		}
		now := s.now()
		if approve {
			err = application.Approve(actor, now)
		} else {
			err = application.Reject(actor, reason, now)
		}
		if err != nil {
			return err
		}
		if err := tx.SaveApplication(ctx, application); err != nil {
			return err
		}
		action := "application_approved"
		detail := "批准申请"
		if !approve {
			action = "application_rejected"
			detail = "退回申请：" + reason
		}
		if err := s.audit(ctx, tx, actor, action, "application", application.ID, detail); err != nil {
			return err
		}
		output, err = s.applicationDTO(ctx, tx, application, actor)
		return err
	})
	if err != nil {
		return nil, toAppError(err)
	}
	return output, nil
}

// ListApplications 按角色返回可见申请：申请人仅本人，其余角色可见全部。
func (s *Service) ListApplications(ctx context.Context, actor entity.Actor) ([]*dto.ApplicationDTO, error) {
	items, err := s.repo.ListApplications(ctx)
	if err != nil {
		return nil, toAppError(err)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	output := make([]*dto.ApplicationDTO, 0, len(items))
	for _, item := range items {
		if actor.Is(entity.RoleRequester) && item.RequesterID != actor.ID {
			continue
		}
		view, err := s.applicationDTO(ctx, s.repo, item, actor)
		if err != nil {
			return nil, toAppError(err)
		}
		output = append(output, view)
	}
	return output, nil
}

func (s *Service) applicationDTO(ctx context.Context, repo entity.Repository, application *entity.Application, actor entity.Actor) (*dto.ApplicationDTO, error) {
	history, err := repo.ListApplicationSnapshots(ctx, application.ID)
	if err != nil {
		return nil, err
	}
	sort.Slice(history, func(i, j int) bool { return history[i].VersionNo < history[j].VersionNo })
	editable := (application.Status == entity.AppStatusDraft || application.Status == entity.AppStatusRejected) && application.RequesterID == actor.ID
	decidable := application.Status == entity.AppStatusPending && actor.Is(entity.RoleApprover) && application.RequesterID != actor.ID
	return dto.NewApplicationDTO(application, history, editable, editable, decidable), nil
}
