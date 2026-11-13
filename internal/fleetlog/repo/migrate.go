package repo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

// AutoMigrate 创建或更新 fleetlog 全部表结构。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&vehicleModel{},
		&applicationModel{},
		&applicationSnapshotModel{},
		&activityModel{},
		&baselineModel{},
		&releaseModel{},
		&receiptModel{},
		&rejectedReceiptModel{},
		&timeoutEventModel{},
		&pullRecordModel{},
		&auditModel{},
		&fleetStateModel{},
	)
}

// Seed 初始化 5 台测试车辆与单行车队状态，重复执行幂等。
func Seed(ctx context.Context, db *gorm.DB) error {
	now := time.Now().UTC()
	for index := 1; index <= entity.FleetVehicleCount; index++ {
		vehicleID := fmt.Sprintf("V%d", index)
		var count int64
		if err := db.WithContext(ctx).Model(&vehicleModel{}).Where("id = ?", vehicleID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		vehicle := &vehicleModel{
			ID:                vehicleID,
			Online:            vehicleID != "V2",
			EffectiveInterval: entity.DefaultInterval,
			EffectiveVersion:  entity.DefaultVehicleVersion,
			CurrentVersion:    entity.DefaultVehicleVersion,
			UpdatedAt:         now,
		}
		if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(vehicle).Error; err != nil {
			return err
		}
	}
	var stateCount int64
	if err := db.WithContext(ctx).Model(&fleetStateModel{}).Where("id = ?", entity.FleetStateID).Count(&stateCount).Error; err != nil {
		return err
	}
	if stateCount == 0 {
		state := &fleetStateModel{ID: entity.FleetStateID, UpdatedAt: now}
		if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(state).Error; err != nil {
			return err
		}
	}
	return nil
}
