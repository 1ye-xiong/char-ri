package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

// faultRepo 按方法名注入错误，用于验证 service 的错误传播分支。
type faultRepo struct {
	entity.Repository
	faults map[string]error
}

func (r *faultRepo) fault(method string) error {
	if r.faults == nil {
		return nil
	}
	return r.faults[method]
}

func (r *faultRepo) Transaction(ctx context.Context, fn func(tx entity.Repository) error) error {
	if err := r.fault("Transaction"); err != nil {
		return err
	}
	return fn(r)
}

func (r *faultRepo) ListVehicles(ctx context.Context) ([]*entity.Vehicle, error) {
	if err := r.fault("ListVehicles"); err != nil {
		return nil, err
	}
	return r.Repository.ListVehicles(ctx)
}

func (r *faultRepo) FindVehicle(ctx context.Context, id string) (*entity.Vehicle, error) {
	if err := r.fault("FindVehicle"); err != nil {
		return nil, err
	}
	return r.Repository.FindVehicle(ctx, id)
}

func (r *faultRepo) FindVehicleForUpdate(ctx context.Context, id string) (*entity.Vehicle, error) {
	if err := r.fault("FindVehicleForUpdate"); err != nil {
		return nil, err
	}
	return r.Repository.FindVehicleForUpdate(ctx, id)
}

func (r *faultRepo) SaveVehicle(ctx context.Context, vehicle *entity.Vehicle) error {
	if err := r.fault("SaveVehicle"); err != nil {
		return err
	}
	return r.Repository.SaveVehicle(ctx, vehicle)
}

func (r *faultRepo) FindApplication(ctx context.Context, id string) (*entity.Application, error) {
	if err := r.fault("FindApplication"); err != nil {
		return nil, err
	}
	return r.Repository.FindApplication(ctx, id)
}

func (r *faultRepo) ListApplications(ctx context.Context) ([]*entity.Application, error) {
	if err := r.fault("ListApplications"); err != nil {
		return nil, err
	}
	return r.Repository.ListApplications(ctx)
}

func (r *faultRepo) SaveApplication(ctx context.Context, application *entity.Application) error {
	if err := r.fault("SaveApplication"); err != nil {
		return err
	}
	return r.Repository.SaveApplication(ctx, application)
}

func (r *faultRepo) AppendApplicationSnapshot(ctx context.Context, snapshot *entity.ApplicationSnapshot) error {
	if err := r.fault("AppendApplicationSnapshot"); err != nil {
		return err
	}
	return r.Repository.AppendApplicationSnapshot(ctx, snapshot)
}

func (r *faultRepo) ListApplicationSnapshots(ctx context.Context, applicationID string) ([]*entity.ApplicationSnapshot, error) {
	if err := r.fault("ListApplicationSnapshots"); err != nil {
		return nil, err
	}
	return r.Repository.ListApplicationSnapshots(ctx, applicationID)
}

func (r *faultRepo) GetFleetStateForUpdate(ctx context.Context) (*entity.FleetState, error) {
	if err := r.fault("GetFleetStateForUpdate"); err != nil {
		return nil, err
	}
	return r.Repository.GetFleetStateForUpdate(ctx)
}

func (r *faultRepo) SaveFleetState(ctx context.Context, state *entity.FleetState) error {
	if err := r.fault("SaveFleetState"); err != nil {
		return err
	}
	return r.Repository.SaveFleetState(ctx, state)
}

func (r *faultRepo) FindActivity(ctx context.Context, id string) (*entity.Activity, error) {
	if err := r.fault("FindActivity"); err != nil {
		return nil, err
	}
	return r.Repository.FindActivity(ctx, id)
}

func (r *faultRepo) ListActivities(ctx context.Context) ([]*entity.Activity, error) {
	if err := r.fault("ListActivities"); err != nil {
		return nil, err
	}
	return r.Repository.ListActivities(ctx)
}

func (r *faultRepo) SaveActivity(ctx context.Context, activity *entity.Activity) error {
	if err := r.fault("SaveActivity"); err != nil {
		return err
	}
	return r.Repository.SaveActivity(ctx, activity)
}

func (r *faultRepo) ListBaselines(ctx context.Context, activityID string) ([]*entity.Baseline, error) {
	if err := r.fault("ListBaselines"); err != nil {
		return nil, err
	}
	return r.Repository.ListBaselines(ctx, activityID)
}

func (r *faultRepo) SaveBaseline(ctx context.Context, baseline *entity.Baseline) error {
	if err := r.fault("SaveBaseline"); err != nil {
		return err
	}
	return r.Repository.SaveBaseline(ctx, baseline)
}

func (r *faultRepo) ListReleasesByActivity(ctx context.Context, activityID string) ([]*entity.Release, error) {
	if err := r.fault("ListReleasesByActivity"); err != nil {
		return nil, err
	}
	return r.Repository.ListReleasesByActivity(ctx, activityID)
}

func (r *faultRepo) ListLatestReleasesByVehicle(ctx context.Context) (map[string]*entity.Release, error) {
	if err := r.fault("ListLatestReleasesByVehicle"); err != nil {
		return nil, err
	}
	return r.Repository.ListLatestReleasesByVehicle(ctx)
}

func (r *faultRepo) FindRelease(ctx context.Context, vehicleID string, version int) (*entity.Release, error) {
	if err := r.fault("FindRelease"); err != nil {
		return nil, err
	}
	return r.Repository.FindRelease(ctx, vehicleID, version)
}

func (r *faultRepo) ListWaitingReleasesBefore(ctx context.Context, deadline time.Time) ([]*entity.Release, error) {
	if err := r.fault("ListWaitingReleasesBefore"); err != nil {
		return nil, err
	}
	return r.Repository.ListWaitingReleasesBefore(ctx, deadline)
}

func (r *faultRepo) SaveRelease(ctx context.Context, release *entity.Release) error {
	if err := r.fault("SaveRelease"); err != nil {
		return err
	}
	return r.Repository.SaveRelease(ctx, release)
}

func (r *faultRepo) SaveReceipt(ctx context.Context, receipt *entity.Receipt) error {
	if err := r.fault("SaveReceipt"); err != nil {
		return err
	}
	return r.Repository.SaveReceipt(ctx, receipt)
}

func (r *faultRepo) SaveRejectedReceipt(ctx context.Context, receipt *entity.RejectedReceipt) error {
	if err := r.fault("SaveRejectedReceipt"); err != nil {
		return err
	}
	return r.Repository.SaveRejectedReceipt(ctx, receipt)
}

func (r *faultRepo) SaveTimeoutEvent(ctx context.Context, event *entity.TimeoutEvent) error {
	if err := r.fault("SaveTimeoutEvent"); err != nil {
		return err
	}
	return r.Repository.SaveTimeoutEvent(ctx, event)
}

func (r *faultRepo) ResolveTimeoutEvents(ctx context.Context, releaseID string, resolvedAt time.Time) error {
	if err := r.fault("ResolveTimeoutEvents"); err != nil {
		return err
	}
	return r.Repository.ResolveTimeoutEvents(ctx, releaseID, resolvedAt)
}

func (r *faultRepo) ListTimeoutEventsByActivity(ctx context.Context, activityID string) ([]*entity.TimeoutEvent, error) {
	if err := r.fault("ListTimeoutEventsByActivity"); err != nil {
		return nil, err
	}
	return r.Repository.ListTimeoutEventsByActivity(ctx, activityID)
}

func (r *faultRepo) SavePull(ctx context.Context, pull *entity.PullRecord) error {
	if err := r.fault("SavePull"); err != nil {
		return err
	}
	return r.Repository.SavePull(ctx, pull)
}

func (r *faultRepo) SaveAudit(ctx context.Context, audit *entity.Audit) error {
	if err := r.fault("SaveAudit"); err != nil {
		return err
	}
	return r.Repository.SaveAudit(ctx, audit)
}

func (r *faultRepo) ListAudits(ctx context.Context, activityID string, limit int) ([]*entity.Audit, error) {
	if err := r.fault("ListAudits"); err != nil {
		return nil, err
	}
	return r.Repository.ListAudits(ctx, activityID, limit)
}

func (r *faultRepo) ListRejectedReceipts(ctx context.Context, activityID string, vehicleID string) ([]*entity.RejectedReceipt, error) {
	if err := r.fault("ListRejectedReceipts"); err != nil {
		return nil, err
	}
	return r.Repository.ListRejectedReceipts(ctx, activityID, vehicleID)
}

func buildPiloting(t *testing.T, svc *Service, ctx context.Context) *dto.ActivityDetailDTO {
	t.Helper()
	detail := mustActivity(t, svc, ctx, []string{"V1", "V3"})
	detail, err := svc.StartPilot(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	return detail
}

func buildPilotPassed(t *testing.T, svc *Service, ctx context.Context) *dto.ActivityDetailDTO {
	t.Helper()
	detail := buildPiloting(t, svc, ctx)
	for _, vehicleID := range []string{"V1", "V3"} {
		_, err := svc.ReportReceipt(ctx, vehicleID, dto.ReportReceiptCommand{VehicleID: vehicleID, Version: 2, Result: entity.ReceiptSuccess})
		require.NoError(t, err)
	}
	detail, err := svc.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ActivityPilotPassed, detail.Activity.Status)
	return detail
}

func buildPromoting(t *testing.T, svc *Service, ctx context.Context) *dto.ActivityDetailDTO {
	t.Helper()
	detail := buildPilotPassed(t, svc, ctx)
	detail, err := svc.ConfirmPromotion(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	return detail
}

func buildAllSuccess(t *testing.T, svc *Service, ctx context.Context) *dto.ActivityDetailDTO {
	t.Helper()
	detail := buildPromoting(t, svc, ctx)
	for _, vehicleID := range []string{"V2", "V4", "V5"} {
		_, err := svc.ReportReceipt(ctx, vehicleID, dto.ReportReceiptCommand{VehicleID: vehicleID, Version: 2, Result: entity.ReceiptSuccess})
		require.NoError(t, err)
	}
	return detail
}

func buildStopped(t *testing.T, svc *Service, ctx context.Context) *dto.ActivityDetailDTO {
	t.Helper()
	detail := buildPiloting(t, svc, ctx)
	detail, err := svc.StopActivity(ctx, testOperator, dto.StopActivityCommand{ID: detail.Activity.ID, Reason: "演示停止"})
	require.NoError(t, err)
	return detail
}

func buildRestoreApproved(t *testing.T, svc *Service, ctx context.Context) *dto.ActivityDetailDTO {
	t.Helper()
	detail := buildStopped(t, svc, ctx)
	detail, err := svc.ApproveRestore(ctx, testApprover, detail.Activity.ID)
	require.NoError(t, err)
	return detail
}

func buildRestoring(t *testing.T, svc *Service, ctx context.Context) *dto.ActivityDetailDTO {
	t.Helper()
	detail := buildRestoreApproved(t, svc, ctx)
	detail, err := svc.StartRestore(ctx, testOperator, detail.Activity.ID)
	require.NoError(t, err)
	return detail
}

func TestServiceRepositoryFaults(t *testing.T) {
	type scenario struct {
		name  string
		fault string
		run   func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error
	}
	scenarios := []scenario{
		{"create application save", "SaveApplication", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.CreateApplication(ctx, testRequester, dto.CreateApplicationCommand{Reason: "原因", Interval: 10})
				return err
			}
		}},
		{"create application audit", "SaveAudit", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.CreateApplication(ctx, testRequester, dto.CreateApplicationCommand{Reason: "原因", Interval: 10})
				return err
			}
		}},
		{"list applications", "ListApplications", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.ListApplications(ctx, testRequester)
				return err
			}
		}},
		{"update application save", "SaveApplication", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			app, err := svc.CreateApplication(ctxOf(t), testRequester, dto.CreateApplicationCommand{Reason: "原因", Interval: 10})
			require.NoError(t, err)
			return func(ctx context.Context, s *Service) error {
				_, err := s.UpdateApplication(ctx, testRequester, dto.UpdateApplicationCommand{ID: app.ID, Reason: "新原因", Interval: 20})
				return err
			}
		}},
		{"submit snapshot", "AppendApplicationSnapshot", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			ctx := ctxOf(t)
			app, err := svc.CreateApplication(ctx, testRequester, dto.CreateApplicationCommand{Reason: "原因", Interval: 10})
			require.NoError(t, err)
			app, err = svc.SubmitApplication(ctx, testRequester, app.ID)
			require.NoError(t, err)
			_, err = svc.RejectApplication(ctx, testApprover, app.ID, "参数需要复核")
			require.NoError(t, err)
			return func(ctx context.Context, s *Service) error {
				_, err := s.SubmitApplication(ctx, testRequester, app.ID)
				return err
			}
		}},
		{"approve application save", "SaveApplication", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			ctx := ctxOf(t)
			app, err := svc.CreateApplication(ctx, testRequester, dto.CreateApplicationCommand{Reason: "待审批", Interval: 20})
			require.NoError(t, err)
			app, err = svc.SubmitApplication(ctx, testRequester, app.ID)
			require.NoError(t, err)
			return func(ctx context.Context, s *Service) error {
				_, err := s.ApproveApplication(ctx, testApprover, app.ID)
				return err
			}
		}},
		{"create activity vehicles", "ListVehicles", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			app := mustApprovedApplication(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: app.ID, PilotVehicles: []string{"V1", "V3"}})
				return err
			}
		}},
		{"create activity save", "SaveActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			app := mustApprovedApplication(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: app.ID, PilotVehicles: []string{"V1", "V3"}})
				return err
			}
		}},
		{"create activity baseline", "SaveBaseline", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			app := mustApprovedApplication(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: app.ID, PilotVehicles: []string{"V1", "V3"}})
				return err
			}
		}},
		{"create activity state", "SaveFleetState", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			app := mustApprovedApplication(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.CreateActivity(ctx, testOperator, dto.CreateActivityCommand{ApplicationID: app.ID, PilotVehicles: []string{"V1", "V3"}})
				return err
			}
		}},
		{"start pilot baselines", "ListBaselines", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.StartPilot(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"start pilot release", "SaveRelease", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.StartPilot(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"start pilot vehicle", "SaveVehicle", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.StartPilot(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"start pilot activity", "SaveActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.StartPilot(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"confirm promotion baselines", "ListBaselines", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildPilotPassed(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.ConfirmPromotion(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"confirm promotion activity", "SaveActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildPilotPassed(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.ConfirmPromotion(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"stop releases", "ListReleasesByActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildPiloting(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.StopActivity(ctx, testOperator, dto.StopActivityCommand{ID: detail.Activity.ID})
				return err
			}
		}},
		{"stop vehicles", "ListVehicles", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildPiloting(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.StopActivity(ctx, testOperator, dto.StopActivityCommand{ID: detail.Activity.ID})
				return err
			}
		}},
		{"stop activity save", "SaveActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildPiloting(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.StopActivity(ctx, testOperator, dto.StopActivityCommand{ID: detail.Activity.ID})
				return err
			}
		}},
		{"approve restore releases", "ListReleasesByActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildStopped(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.ApproveRestore(ctx, testApprover, detail.Activity.ID)
				return err
			}
		}},
		{"start restore releases", "ListReleasesByActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildRestoreApproved(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.StartRestore(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"start restore baselines", "ListBaselines", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildRestoreApproved(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.StartRestore(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"mark completed vehicles", "ListVehicles", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildAllSuccess(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.MarkCompleted(ctx, testApprover, detail.Activity.ID)
				return err
			}
		}},
		{"mark completed releases", "ListReleasesByActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildAllSuccess(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.MarkCompleted(ctx, testApprover, detail.Activity.ID)
				return err
			}
		}},
		{"mark completed state", "GetFleetStateForUpdate", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildAllSuccess(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.MarkCompleted(ctx, testApprover, detail.Activity.ID)
				return err
			}
		}},
		{"mark completed save", "SaveActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildAllSuccess(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.MarkCompleted(ctx, testApprover, detail.Activity.ID)
				return err
			}
		}},
		{"mark restored vehicles", "ListVehicles", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildRestoring(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.MarkRestored(ctx, testApprover, detail.Activity.ID)
				return err
			}
		}},
		{"mark restored releases", "ListReleasesByActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := buildRestoring(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.MarkRestored(ctx, testApprover, detail.Activity.ID)
				return err
			}
		}},
		{"pull latest", "ListLatestReleasesByVehicle", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.PullConfig(ctx, "V1", dto.PullConfigCommand{VehicleID: "V1"})
				return err
			}
		}},
		{"pull save", "SavePull", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.PullConfig(ctx, "V1", dto.PullConfigCommand{VehicleID: "V1"})
				return err
			}
		}},
		{"receipt vehicle", "FindVehicleForUpdate", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			buildPiloting(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
				return err
			}
		}},
		{"receipt find release", "FindRelease", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			buildPiloting(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
				return err
			}
		}},
		{"receipt save vehicle", "SaveVehicle", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			buildPiloting(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
				return err
			}
		}},
		{"receipt save release", "SaveRelease", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			buildPiloting(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptFailure, FailureReason: "应用失败"})
				return err
			}
		}},
		{"receipt save receipt", "SaveReceipt", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			buildPiloting(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
				return err
			}
		}},
		{"receipt resolve timeout", "ResolveTimeoutEvents", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			buildPiloting(t, svc, ctxOf(t))
			return func(ctx context.Context, s *Service) error {
				_, err := s.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V1", Version: 2, Result: entity.ReceiptSuccess})
				return err
			}
		}},
		{"receipt rejected save", "SaveRejectedReceipt", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.ReportReceipt(ctx, "V1", dto.ReportReceiptCommand{VehicleID: "V2", Version: 2, Result: entity.ReceiptSuccess})
				return err
			}
		}},
		{"sweep list", "ListWaitingReleasesBefore", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			buildPiloting(t, svc, ctxOf(t))
			clock.Advance(61 * time.Second)
			return func(ctx context.Context, s *Service) error {
				_, err := s.SweepTimeouts(ctx)
				return err
			}
		}},
		{"sweep release", "FindRelease", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			buildPiloting(t, svc, ctxOf(t))
			clock.Advance(61 * time.Second)
			return func(ctx context.Context, s *Service) error {
				_, err := s.SweepTimeouts(ctx)
				return err
			}
		}},
		{"sweep event", "SaveTimeoutEvent", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			buildPiloting(t, svc, ctxOf(t))
			clock.Advance(61 * time.Second)
			return func(ctx context.Context, s *Service) error {
				_, err := s.SweepTimeouts(ctx)
				return err
			}
		}},
		{"detail activity", "FindActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"detail vehicles", "ListVehicles", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"detail releases", "ListReleasesByActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"detail baselines", "ListBaselines", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"detail latest", "ListLatestReleasesByVehicle", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"detail rejected", "ListRejectedReceipts", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"detail timeouts", "ListTimeoutEventsByActivity", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"detail audits", "ListAudits", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"detail snapshots", "ListApplicationSnapshots", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			detail := mustActivity(t, svc, ctxOf(t), []string{"V1", "V3"})
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetActivityDetail(ctx, testOperator, detail.Activity.ID)
				return err
			}
		}},
		{"current activity", "ListActivities", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.GetCurrentActivity(ctx, testOperator)
				return err
			}
		}},
		{"list activities", "ListActivities", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.ListActivities(ctx)
				return err
			}
		}},
		{"list vehicles", "ListLatestReleasesByVehicle", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.ListVehicles(ctx)
				return err
			}
		}},
		{"list rejected", "ListRejectedReceipts", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.ListRejectedReceipts(ctx, "", "")
				return err
			}
		}},
		{"list audits", "ListAudits", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.ListAudits(ctx, "", 10)
				return err
			}
		}},
		{"set online find", "FindVehicleForUpdate", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.SetVehicleOnline(ctx, testOperator, "V2", true)
				return err
			}
		}},
		{"set online save", "SaveAudit", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.SetVehicleOnline(ctx, testOperator, "V2", true)
				return err
			}
		}},
		{"set online list", "ListVehicles", func(t *testing.T, svc *Service, repo *fakeRepo, clock *fakeClock) func(context.Context, *Service) error {
			return func(ctx context.Context, s *Service) error {
				_, err := s.SetVehicleOnline(ctx, testOperator, "V2", true)
				return err
			}
		}},
	}

	for _, tc := range scenarios {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			baseSvc, repo, clock := newTestEnv(t)
			call := tc.run(t, baseSvc, repo, clock)
			faulted, err := NewService(&faultRepo{
				Repository: repo,
				faults:     map[string]error{tc.fault: errors.New("injected repository failure")},
			}, clock, testServiceConfig(), nil)
			require.NoError(t, err)
			require.Error(t, call(context.Background(), faulted))
		})
	}
}

func ctxOf(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}
