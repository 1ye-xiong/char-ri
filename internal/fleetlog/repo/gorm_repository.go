package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

// GormRepository 使用 Gorm 持久化 fleetlog 聚合。
type GormRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewGormRepository 创建 fleetlog 仓储。
func NewGormRepository(db *gorm.DB, loggers ...*zap.Logger) *GormRepository {
	log := zap.NewNop()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	return &GormRepository{db: db, log: log}
}

// Transaction 在一个数据库事务内执行用例，并注入事务版仓储。
func (r *GormRepository) Transaction(ctx context.Context, fn func(tx entity.Repository) error) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&GormRepository{db: tx, log: r.log})
	})
	if err != nil {
		r.log.Warn("fleetlog transaction failed", zap.Error(err))
	}
	return err
}

// ListVehicles 返回车队全部车辆。
func (r *GormRepository) ListVehicles(ctx context.Context) ([]*entity.Vehicle, error) {
	var records []vehicleModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.Vehicle, 0, len(records))
	for i := range records {
		items = append(items, toVehicle(&records[i]))
	}
	return items, nil
}

// FindVehicle 按编号加载车辆。
func (r *GormRepository) FindVehicle(ctx context.Context, id string) (*entity.Vehicle, error) {
	return r.findVehicle(r.db.WithContext(ctx), id, false)
}

// FindVehicleForUpdate 在事务内加行锁加载车辆，保证版本分配原子递增。
func (r *GormRepository) FindVehicleForUpdate(ctx context.Context, id string) (*entity.Vehicle, error) {
	return r.findVehicle(r.db.WithContext(ctx), id, true)
}

func (r *GormRepository) findVehicle(db *gorm.DB, id string, forUpdate bool) (*entity.Vehicle, error) {
	var record vehicleModel
	query := db
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.Where("id = ?", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toVehicle(&record), nil
}

// SaveVehicle 新增或更新车辆。
func (r *GormRepository) SaveVehicle(ctx context.Context, vehicle *entity.Vehicle) error {
	record := fromVehicle(vehicle)
	return r.db.WithContext(ctx).Save(record).Error
}

// FindApplication 按 ID 加载申请。
func (r *GormRepository) FindApplication(ctx context.Context, id string) (*entity.Application, error) {
	var record applicationModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toApplication(&record), nil
}

// ListApplications 返回全部申请。
func (r *GormRepository) ListApplications(ctx context.Context) ([]*entity.Application, error) {
	var records []applicationModel
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.Application, 0, len(records))
	for i := range records {
		items = append(items, toApplication(&records[i]))
	}
	return items, nil
}

// SaveApplication 新增或更新申请。
func (r *GormRepository) SaveApplication(ctx context.Context, application *entity.Application) error {
	return r.db.WithContext(ctx).Save(fromApplication(application)).Error
}

// AppendApplicationSnapshot 追加申请历史版本。
func (r *GormRepository) AppendApplicationSnapshot(ctx context.Context, snapshot *entity.ApplicationSnapshot) error {
	record := &applicationSnapshotModel{
		ID:            snapshot.ID,
		ApplicationID: snapshot.ApplicationID,
		VersionNo:     snapshot.VersionNo,
		Reason:        snapshot.Reason,
		Interval:      snapshot.Interval,
		Status:        snapshot.Status,
		RejectReason:  snapshot.RejectReason,
		DecidedBy:     snapshot.DecidedBy,
		CreatedAt:     snapshot.CreatedAt,
	}
	return r.db.WithContext(ctx).Create(record).Error
}

// ListApplicationSnapshots 返回申请历史版本。
func (r *GormRepository) ListApplicationSnapshots(ctx context.Context, applicationID string) ([]*entity.ApplicationSnapshot, error) {
	var records []applicationSnapshotModel
	if err := r.db.WithContext(ctx).Where("application_id = ?", applicationID).Order("version_no asc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.ApplicationSnapshot, 0, len(records))
	for i := range records {
		record := &records[i]
		items = append(items, &entity.ApplicationSnapshot{
			ID:            record.ID,
			ApplicationID: record.ApplicationID,
			VersionNo:     record.VersionNo,
			Reason:        record.Reason,
			Interval:      record.Interval,
			Status:        record.Status,
			RejectReason:  record.RejectReason,
			DecidedBy:     record.DecidedBy,
			CreatedAt:     record.CreatedAt,
		})
	}
	return items, nil
}

// GetFleetStateForUpdate 加锁读取单行车队状态，用于串行化活动创建与结束。
func (r *GormRepository) GetFleetStateForUpdate(ctx context.Context) (*entity.FleetState, error) {
	var record fleetStateModel
	db := r.db.WithContext(ctx)
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", entity.FleetStateID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		seed := &fleetStateModel{ID: entity.FleetStateID, UpdatedAt: time.Now().UTC()}
		if createErr := db.Create(seed).Error; createErr != nil && !isDuplicateKey(createErr) {
			return nil, createErr
		}
		err = db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", entity.FleetStateID).First(&record).Error
	}
	if err != nil {
		return nil, err
	}
	return &entity.FleetState{ID: record.ID, OpenActivityID: record.OpenActivityID, UpdatedAt: record.UpdatedAt}, nil
}

// SaveFleetState 更新车队状态。
func (r *GormRepository) SaveFleetState(ctx context.Context, state *entity.FleetState) error {
	return r.db.WithContext(ctx).Save(&fleetStateModel{
		ID:             state.ID,
		OpenActivityID: state.OpenActivityID,
		UpdatedAt:      state.UpdatedAt,
	}).Error
}

// FindActivity 按 ID 加载活动。
func (r *GormRepository) FindActivity(ctx context.Context, id string) (*entity.Activity, error) {
	var record activityModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toActivity(&record), nil
}

// ListActivities 返回全部活动。
func (r *GormRepository) ListActivities(ctx context.Context) ([]*entity.Activity, error) {
	var records []activityModel
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.Activity, 0, len(records))
	for i := range records {
		items = append(items, toActivity(&records[i]))
	}
	return items, nil
}

// SaveActivity 新增或更新活动。
func (r *GormRepository) SaveActivity(ctx context.Context, activity *entity.Activity) error {
	return r.db.WithContext(ctx).Save(fromActivity(activity)).Error
}

// ListBaselines 返回活动的基线快照。
func (r *GormRepository) ListBaselines(ctx context.Context, activityID string) ([]*entity.Baseline, error) {
	var records []baselineModel
	if err := r.db.WithContext(ctx).Where("activity_id = ?", activityID).Order("vehicle_id asc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.Baseline, 0, len(records))
	for i := range records {
		record := &records[i]
		items = append(items, &entity.Baseline{
			ActivityID: record.ActivityID,
			VehicleID:  record.VehicleID,
			Interval:   record.Interval,
			Version:    record.Version,
			CreatedAt:  record.CreatedAt,
		})
	}
	return items, nil
}

// SaveBaseline 保存一条基线快照。
func (r *GormRepository) SaveBaseline(ctx context.Context, baseline *entity.Baseline) error {
	return r.db.WithContext(ctx).Save(&baselineModel{
		ActivityID: baseline.ActivityID,
		VehicleID:  baseline.VehicleID,
		Interval:   baseline.Interval,
		Version:    baseline.Version,
		CreatedAt:  baseline.CreatedAt,
	}).Error
}

// ListReleasesByActivity 返回活动的全部发布记录。
func (r *GormRepository) ListReleasesByActivity(ctx context.Context, activityID string) ([]*entity.Release, error) {
	var records []releaseModel
	if err := r.db.WithContext(ctx).Where("activity_id = ?", activityID).Order("vehicle_id asc, version asc").Find(&records).Error; err != nil {
		return nil, err
	}
	return toReleases(records), nil
}

// ListLatestReleasesByVehicle 返回每台车最新发布的版本。
func (r *GormRepository) ListLatestReleasesByVehicle(ctx context.Context) (map[string]*entity.Release, error) {
	var records []releaseModel
	if err := r.db.WithContext(ctx).Order("version asc").Find(&records).Error; err != nil {
		return nil, err
	}
	output := make(map[string]*entity.Release, len(records))
	for i := range records {
		release := toRelease(&records[i])
		current, ok := output[release.VehicleID]
		if !ok || release.Version > current.Version {
			output[release.VehicleID] = release
		}
	}
	return output, nil
}

// FindRelease 按车辆与版本加载发布记录。
func (r *GormRepository) FindRelease(ctx context.Context, vehicleID string, version int) (*entity.Release, error) {
	var record releaseModel
	err := r.db.WithContext(ctx).Where("vehicle_id = ? AND version = ?", vehicleID, version).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toRelease(&record), nil
}

// ListWaitingReleasesBefore 返回下发时间早于截止时刻且仍等待回执的发布记录。
func (r *GormRepository) ListWaitingReleasesBefore(ctx context.Context, deadline time.Time) ([]*entity.Release, error) {
	var records []releaseModel
	if err := r.db.WithContext(ctx).
		Where("status = ? AND dispatched_at <= ?", entity.ReleaseWaiting, deadline).
		Order("dispatched_at asc").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return toReleases(records), nil
}

// SaveRelease 新增或更新发布记录。
func (r *GormRepository) SaveRelease(ctx context.Context, release *entity.Release) error {
	return r.db.WithContext(ctx).Save(fromRelease(release)).Error
}

// SaveReceipt 保存被接受的回执。
func (r *GormRepository) SaveReceipt(ctx context.Context, receipt *entity.Receipt) error {
	return r.db.WithContext(ctx).Create(&receiptModel{
		ID:            receipt.ID,
		ReleaseID:     receipt.ReleaseID,
		ActivityID:    receipt.ActivityID,
		VehicleID:     receipt.VehicleID,
		Version:       receipt.Version,
		Result:        receipt.Result,
		FailureReason: receipt.FailureReason,
		ReportedAt:    receipt.ReportedAt,
		AcceptedAt:    receipt.AcceptedAt,
	}).Error
}

// ListReceiptsByActivity 返回活动内被接受的回执。
func (r *GormRepository) ListReceiptsByActivity(ctx context.Context, activityID string) ([]*entity.Receipt, error) {
	var records []receiptModel
	if err := r.db.WithContext(ctx).Where("activity_id = ?", activityID).Order("reported_at asc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.Receipt, 0, len(records))
	for i := range records {
		record := &records[i]
		items = append(items, &entity.Receipt{
			ID:            record.ID,
			ReleaseID:     record.ReleaseID,
			ActivityID:    record.ActivityID,
			VehicleID:     record.VehicleID,
			Version:       record.Version,
			Result:        record.Result,
			FailureReason: record.FailureReason,
			ReportedAt:    record.ReportedAt,
			AcceptedAt:    record.AcceptedAt,
		})
	}
	return items, nil
}

// SaveRejectedReceipt 保存被拒回执。
func (r *GormRepository) SaveRejectedReceipt(ctx context.Context, receipt *entity.RejectedReceipt) error {
	return r.db.WithContext(ctx).Create(&rejectedReceiptModel{
		ID:         receipt.ID,
		ActivityID: receipt.ActivityID,
		VehicleID:  receipt.VehicleID,
		Version:    receipt.Version,
		Reason:     receipt.Reason,
		Detail:     receipt.Detail,
		ReportedAt: receipt.ReportedAt,
	}).Error
}

// ListRejectedReceipts 查询被拒回执，支持按活动或车辆过滤。
func (r *GormRepository) ListRejectedReceipts(ctx context.Context, activityID string, vehicleID string) ([]*entity.RejectedReceipt, error) {
	query := r.db.WithContext(ctx).Model(&rejectedReceiptModel{})
	if activityID != "" {
		query = query.Where("activity_id = ?", activityID)
	}
	if vehicleID != "" {
		query = query.Where("vehicle_id = ?", vehicleID)
	}
	var records []rejectedReceiptModel
	if err := query.Order("reported_at desc").Limit(500).Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.RejectedReceipt, 0, len(records))
	for i := range records {
		record := &records[i]
		items = append(items, &entity.RejectedReceipt{
			ID:         record.ID,
			ActivityID: record.ActivityID,
			VehicleID:  record.VehicleID,
			Version:    record.Version,
			Reason:     record.Reason,
			Detail:     record.Detail,
			ReportedAt: record.ReportedAt,
		})
	}
	return items, nil
}

// SaveTimeoutEvent 保存超时事件。
func (r *GormRepository) SaveTimeoutEvent(ctx context.Context, event *entity.TimeoutEvent) error {
	return r.db.WithContext(ctx).Create(&timeoutEventModel{
		ID:         event.ID,
		ReleaseID:  event.ReleaseID,
		ActivityID: event.ActivityID,
		VehicleID:  event.VehicleID,
		Version:    event.Version,
		TimedOutAt: event.TimedOutAt,
		ResolvedAt: event.ResolvedAt,
	}).Error
}

// ResolveTimeoutEvents 把该发布记录的超时事件标记为已解决（迟到成功）。
func (r *GormRepository) ResolveTimeoutEvents(ctx context.Context, releaseID string, resolvedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&timeoutEventModel{}).
		Where("release_id = ? AND resolved_at IS NULL", releaseID).
		Update("resolved_at", resolvedAt).Error
}

// ListTimeoutEventsByActivity 返回活动的超时事件历史。
func (r *GormRepository) ListTimeoutEventsByActivity(ctx context.Context, activityID string) ([]*entity.TimeoutEvent, error) {
	var records []timeoutEventModel
	if err := r.db.WithContext(ctx).Where("activity_id = ?", activityID).Order("timed_out_at asc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.TimeoutEvent, 0, len(records))
	for i := range records {
		record := &records[i]
		items = append(items, &entity.TimeoutEvent{
			ID:         record.ID,
			ReleaseID:  record.ReleaseID,
			ActivityID: record.ActivityID,
			VehicleID:  record.VehicleID,
			Version:    record.Version,
			TimedOutAt: record.TimedOutAt,
			ResolvedAt: record.ResolvedAt,
		})
	}
	return items, nil
}

// SavePull 保存车辆拉取记录。
func (r *GormRepository) SavePull(ctx context.Context, pull *entity.PullRecord) error {
	return r.db.WithContext(ctx).Create(&pullRecordModel{
		ID:        pull.ID,
		VehicleID: pull.VehicleID,
		Version:   pull.Version,
		Interval:  pull.Interval,
		ReleaseID: pull.ReleaseID,
		PulledAt:  pull.PulledAt,
	}).Error
}

// ListPullsByVehicle 返回车辆最近的拉取记录。
func (r *GormRepository) ListPullsByVehicle(ctx context.Context, vehicleID string, limit int) ([]*entity.PullRecord, error) {
	var records []pullRecordModel
	query := r.db.WithContext(ctx).Where("vehicle_id = ?", vehicleID).Order("pulled_at desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.PullRecord, 0, len(records))
	for i := range records {
		record := &records[i]
		items = append(items, &entity.PullRecord{
			ID:        record.ID,
			VehicleID: record.VehicleID,
			Version:   record.Version,
			Interval:  record.Interval,
			ReleaseID: record.ReleaseID,
			PulledAt:  record.PulledAt,
		})
	}
	return items, nil
}

// SaveAudit 保存审计记录。
func (r *GormRepository) SaveAudit(ctx context.Context, audit *entity.Audit) error {
	return r.db.WithContext(ctx).Create(&auditModel{
		ID:         audit.ID,
		ActorID:    audit.ActorID,
		ActorRole:  audit.ActorRole,
		Action:     audit.Action,
		TargetType: audit.TargetType,
		TargetID:   audit.TargetID,
		Detail:     audit.Detail,
		CreatedAt:  audit.CreatedAt,
	}).Error
}

// ListAudits 返回审计记录，按活动过滤时同时包含该活动下发记录的审计。
func (r *GormRepository) ListAudits(ctx context.Context, activityID string, limit int) ([]*entity.Audit, error) {
	query := r.db.WithContext(ctx).Model(&auditModel{})
	if activityID != "" {
		var releaseIDs []string
		if err := r.db.WithContext(ctx).Model(&releaseModel{}).Where("activity_id = ?", activityID).Pluck("id", &releaseIDs).Error; err != nil {
			return nil, err
		}
		if len(releaseIDs) > 0 {
			query = query.Where("target_id = ? OR target_id IN ?", activityID, releaseIDs)
		} else {
			query = query.Where("target_id = ?", activityID)
		}
	}
	var records []auditModel
	if err := query.Order("created_at desc").Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.Audit, 0, len(records))
	for i := range records {
		record := &records[i]
		items = append(items, &entity.Audit{
			ID:         record.ID,
			ActorID:    record.ActorID,
			ActorRole:  record.ActorRole,
			Action:     record.Action,
			TargetType: record.TargetType,
			TargetID:   record.TargetID,
			Detail:     record.Detail,
			CreatedAt:  record.CreatedAt,
		})
	}
	return items, nil
}

func isDuplicateKey(err error) bool {
	return err != nil && (errors.Is(err, gorm.ErrDuplicatedKey) ||
		containsAny(err.Error(), "Duplicate entry", "UNIQUE constraint failed", "duplicate key"))
}

func containsAny(value string, keys ...string) bool {
	for _, key := range keys {
		if len(value) >= len(key) && indexOf(value, key) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(value string, key string) int {
	for i := 0; i+len(key) <= len(value); i++ {
		if value[i:i+len(key)] == key {
			return i
		}
	}
	return -1
}

func toVehicle(record *vehicleModel) *entity.Vehicle {
	return &entity.Vehicle{
		ID:                record.ID,
		Online:            record.Online,
		EffectiveInterval: record.EffectiveInterval,
		EffectiveVersion:  record.EffectiveVersion,
		CurrentVersion:    record.CurrentVersion,
		UpdatedAt:         record.UpdatedAt,
	}
}

func fromVehicle(vehicle *entity.Vehicle) *vehicleModel {
	return &vehicleModel{
		ID:                vehicle.ID,
		Online:            vehicle.Online,
		EffectiveInterval: vehicle.EffectiveInterval,
		EffectiveVersion:  vehicle.EffectiveVersion,
		CurrentVersion:    vehicle.CurrentVersion,
		UpdatedAt:         vehicle.UpdatedAt,
	}
}

func toApplication(record *applicationModel) *entity.Application {
	return &entity.Application{
		ID:            record.ID,
		VersionNo:     record.VersionNo,
		RequesterID:   record.RequesterID,
		RequesterName: record.RequesterName,
		Reason:        record.Reason,
		Interval:      record.Interval,
		Status:        record.Status,
		RejectReason:  record.RejectReason,
		DecidedBy:     record.DecidedBy,
		SubmittedAt:   record.SubmittedAt,
		DecidedAt:     record.DecidedAt,
		ActivityID:    record.ActivityID,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

func fromApplication(application *entity.Application) *applicationModel {
	return &applicationModel{
		ID:            application.ID,
		VersionNo:     application.VersionNo,
		RequesterID:   application.RequesterID,
		RequesterName: application.RequesterName,
		Reason:        application.Reason,
		Interval:      application.Interval,
		Status:        application.Status,
		RejectReason:  application.RejectReason,
		DecidedBy:     application.DecidedBy,
		SubmittedAt:   application.SubmittedAt,
		DecidedAt:     application.DecidedAt,
		ActivityID:    application.ActivityID,
		CreatedAt:     application.CreatedAt,
		UpdatedAt:     application.UpdatedAt,
	}
}

func toActivity(record *activityModel) *entity.Activity {
	return &entity.Activity{
		ID:                record.ID,
		ApplicationID:     record.ApplicationID,
		Interval:          record.Interval,
		PilotVehicles:     decodeList(record.PilotVehicles),
		PromotionVehicles: decodeList(record.PromotionVehicles),
		Status:            record.Status,
		CreatedBy:         record.CreatedBy,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
		StoppedBy:         record.StoppedBy,
		StoppedAt:         record.StoppedAt,
		StopScope:         record.StopScope,
		RestoreApprovedBy: record.RestoreApprovedBy,
		RestoreApprovedAt: record.RestoreApprovedAt,
		RestoreStartedBy:  record.RestoreStartedBy,
		RestoreStartedAt:  record.RestoreStartedAt,
		CompletedBy:       record.CompletedBy,
		CompletedAt:       record.CompletedAt,
		RestoredBy:        record.RestoredBy,
		RestoredAt:        record.RestoredAt,
	}
}

func fromActivity(activity *entity.Activity) *activityModel {
	return &activityModel{
		ID:                activity.ID,
		ApplicationID:     activity.ApplicationID,
		Interval:          activity.Interval,
		PilotVehicles:     encodeList(activity.PilotVehicles),
		PromotionVehicles: encodeList(activity.PromotionVehicles),
		Status:            activity.Status,
		CreatedBy:         activity.CreatedBy,
		CreatedAt:         activity.CreatedAt,
		UpdatedAt:         activity.UpdatedAt,
		StoppedBy:         activity.StoppedBy,
		StoppedAt:         activity.StoppedAt,
		StopScope:         activity.StopScope,
		RestoreApprovedBy: activity.RestoreApprovedBy,
		RestoreApprovedAt: activity.RestoreApprovedAt,
		RestoreStartedBy:  activity.RestoreStartedBy,
		RestoreStartedAt:  activity.RestoreStartedAt,
		CompletedBy:       activity.CompletedBy,
		CompletedAt:       activity.CompletedAt,
		RestoredBy:        activity.RestoredBy,
		RestoredAt:        activity.RestoredAt,
	}
}

func toReleases(records []releaseModel) []*entity.Release {
	items := make([]*entity.Release, 0, len(records))
	for i := range records {
		items = append(items, toRelease(&records[i]))
	}
	return items
}

func toRelease(record *releaseModel) *entity.Release {
	return &entity.Release{
		ID:                record.ID,
		ActivityID:        record.ActivityID,
		VehicleID:         record.VehicleID,
		Version:           record.Version,
		Batch:             record.Batch,
		ExpectedInterval:  record.ExpectedInterval,
		BaselineInterval:  record.BaselineInterval,
		Status:            record.Status,
		FailureReason:     record.FailureReason,
		DispatchedAt:      record.DispatchedAt,
		ReceiptAt:         record.ReceiptAt,
		TimeoutAt:         record.TimeoutAt,
		Late:              record.Late,
		DuplicateReceipts: record.DuplicateReceipts,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

func fromRelease(release *entity.Release) *releaseModel {
	return &releaseModel{
		ID:                release.ID,
		ActivityID:        release.ActivityID,
		VehicleID:         release.VehicleID,
		Version:           release.Version,
		Batch:             release.Batch,
		ExpectedInterval:  release.ExpectedInterval,
		BaselineInterval:  release.BaselineInterval,
		Status:            release.Status,
		FailureReason:     release.FailureReason,
		DispatchedAt:      release.DispatchedAt,
		ReceiptAt:         release.ReceiptAt,
		TimeoutAt:         release.TimeoutAt,
		Late:              release.Late,
		DuplicateReceipts: release.DuplicateReceipts,
		CreatedAt:         release.CreatedAt,
		UpdatedAt:         release.UpdatedAt,
	}
}

func encodeList(items []string) string {
	if items == nil {
		items = []string{}
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func decodeList(value string) []string {
	items := make([]string, 0)
	if value == "" {
		return items
	}
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return []string{}
	}
	return items
}
