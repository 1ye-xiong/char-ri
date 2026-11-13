package entity

import (
	"context"
	"time"
)

// Repository 是 fleetlog 聚合的持久化契约。
// 事务通过 Transaction 注入带事务的 Repository 实现，service 不感知 gorm。
type Repository interface {
	Transaction(ctx context.Context, fn func(tx Repository) error) error

	ListVehicles(ctx context.Context) ([]*Vehicle, error)
	FindVehicle(ctx context.Context, id string) (*Vehicle, error)
	FindVehicleForUpdate(ctx context.Context, id string) (*Vehicle, error)
	SaveVehicle(ctx context.Context, vehicle *Vehicle) error

	FindApplication(ctx context.Context, id string) (*Application, error)
	ListApplications(ctx context.Context) ([]*Application, error)
	SaveApplication(ctx context.Context, application *Application) error
	AppendApplicationSnapshot(ctx context.Context, snapshot *ApplicationSnapshot) error
	ListApplicationSnapshots(ctx context.Context, applicationID string) ([]*ApplicationSnapshot, error)

	GetFleetStateForUpdate(ctx context.Context) (*FleetState, error)
	SaveFleetState(ctx context.Context, state *FleetState) error

	FindActivity(ctx context.Context, id string) (*Activity, error)
	ListActivities(ctx context.Context) ([]*Activity, error)
	SaveActivity(ctx context.Context, activity *Activity) error
	ListBaselines(ctx context.Context, activityID string) ([]*Baseline, error)
	SaveBaseline(ctx context.Context, baseline *Baseline) error

	ListReleasesByActivity(ctx context.Context, activityID string) ([]*Release, error)
	ListLatestReleasesByVehicle(ctx context.Context) (map[string]*Release, error)
	FindRelease(ctx context.Context, vehicleID string, version int) (*Release, error)
	ListWaitingReleasesBefore(ctx context.Context, deadline time.Time) ([]*Release, error)
	SaveRelease(ctx context.Context, release *Release) error

	SaveReceipt(ctx context.Context, receipt *Receipt) error
	ListReceiptsByActivity(ctx context.Context, activityID string) ([]*Receipt, error)
	SaveRejectedReceipt(ctx context.Context, receipt *RejectedReceipt) error
	ListRejectedReceipts(ctx context.Context, activityID string, vehicleID string) ([]*RejectedReceipt, error)
	SaveTimeoutEvent(ctx context.Context, event *TimeoutEvent) error
	ResolveTimeoutEvents(ctx context.Context, releaseID string, resolvedAt time.Time) error
	ListTimeoutEventsByActivity(ctx context.Context, activityID string) ([]*TimeoutEvent, error)
	SavePull(ctx context.Context, pull *PullRecord) error
	ListPullsByVehicle(ctx context.Context, vehicleID string, limit int) ([]*PullRecord, error)

	SaveAudit(ctx context.Context, audit *Audit) error
	ListAudits(ctx context.Context, activityID string, limit int) ([]*Audit, error)
}
