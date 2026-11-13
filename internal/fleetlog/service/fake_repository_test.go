package service

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}

type fakeData struct {
	vehicles     map[string]*entity.Vehicle
	applications map[string]*entity.Application
	snapshots    map[string][]*entity.ApplicationSnapshot
	activities   map[string]*entity.Activity
	baselines    map[string][]*entity.Baseline
	releases     map[string]map[int]*entity.Release
	receipts     []*entity.Receipt
	rejected     []*entity.RejectedReceipt
	timeouts     []*entity.TimeoutEvent
	pulls        []*entity.PullRecord
	audits       []*entity.Audit
	state        entity.FleetState
}

func newFakeData() *fakeData {
	data := &fakeData{
		vehicles:     map[string]*entity.Vehicle{},
		applications: map[string]*entity.Application{},
		snapshots:    map[string][]*entity.ApplicationSnapshot{},
		activities:   map[string]*entity.Activity{},
		baselines:    map[string][]*entity.Baseline{},
		releases:     map[string]map[int]*entity.Release{},
		state:        entity.FleetState{ID: entity.FleetStateID},
	}
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	for index := 1; index <= entity.FleetVehicleCount; index++ {
		id := "V" + string(rune('0'+index))
		data.vehicles[id] = &entity.Vehicle{
			ID:                id,
			Online:            id != "V2",
			EffectiveInterval: entity.DefaultInterval,
			EffectiveVersion:  entity.DefaultVehicleVersion,
			CurrentVersion:    entity.DefaultVehicleVersion,
			UpdatedAt:         now,
		}
	}
	return data
}

type fakeRepo struct {
	mu     sync.Mutex
	data   *fakeData
	noLock bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{data: newFakeData()}
}

func (r *fakeRepo) begin() func() {
	if r.noLock {
		return func() {}
	}
	r.mu.Lock()
	return r.mu.Unlock
}

func (r *fakeRepo) Transaction(ctx context.Context, fn func(tx entity.Repository) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn(&fakeRepo{data: r.data, noLock: true})
}

func copyVehicle(item *entity.Vehicle) *entity.Vehicle {
	clone := *item
	return &clone
}

func copyApplication(item *entity.Application) *entity.Application {
	clone := *item
	return &clone
}

func copyActivity(item *entity.Activity) *entity.Activity {
	clone := *item
	clone.PilotVehicles = append([]string{}, item.PilotVehicles...)
	clone.PromotionVehicles = append([]string{}, item.PromotionVehicles...)
	return &clone
}

func copyRelease(item *entity.Release) *entity.Release {
	clone := *item
	return &clone
}

func (r *fakeRepo) ListVehicles(ctx context.Context) ([]*entity.Vehicle, error) {
	defer r.begin()()
	items := make([]*entity.Vehicle, 0, len(r.data.vehicles))
	for _, item := range r.data.vehicles {
		items = append(items, copyVehicle(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *fakeRepo) FindVehicle(ctx context.Context, id string) (*entity.Vehicle, error) {
	defer r.begin()()
	item, ok := r.data.vehicles[id]
	if !ok {
		return nil, entity.ErrNotFound
	}
	return copyVehicle(item), nil
}

func (r *fakeRepo) FindVehicleForUpdate(ctx context.Context, id string) (*entity.Vehicle, error) {
	return r.FindVehicle(ctx, id)
}

func (r *fakeRepo) SaveVehicle(ctx context.Context, vehicle *entity.Vehicle) error {
	defer r.begin()()
	r.data.vehicles[vehicle.ID] = copyVehicle(vehicle)
	return nil
}

func (r *fakeRepo) FindApplication(ctx context.Context, id string) (*entity.Application, error) {
	defer r.begin()()
	item, ok := r.data.applications[id]
	if !ok {
		return nil, entity.ErrNotFound
	}
	return copyApplication(item), nil
}

func (r *fakeRepo) ListApplications(ctx context.Context) ([]*entity.Application, error) {
	defer r.begin()()
	items := make([]*entity.Application, 0, len(r.data.applications))
	for _, item := range r.data.applications {
		items = append(items, copyApplication(item))
	}
	return items, nil
}

func (r *fakeRepo) SaveApplication(ctx context.Context, application *entity.Application) error {
	defer r.begin()()
	r.data.applications[application.ID] = copyApplication(application)
	return nil
}

func (r *fakeRepo) AppendApplicationSnapshot(ctx context.Context, snapshot *entity.ApplicationSnapshot) error {
	defer r.begin()()
	clone := *snapshot
	r.data.snapshots[snapshot.ApplicationID] = append(r.data.snapshots[snapshot.ApplicationID], &clone)
	return nil
}

func (r *fakeRepo) ListApplicationSnapshots(ctx context.Context, applicationID string) ([]*entity.ApplicationSnapshot, error) {
	defer r.begin()()
	items := make([]*entity.ApplicationSnapshot, 0, len(r.data.snapshots[applicationID]))
	for _, item := range r.data.snapshots[applicationID] {
		clone := *item
		items = append(items, &clone)
	}
	return items, nil
}

func (r *fakeRepo) GetFleetStateForUpdate(ctx context.Context) (*entity.FleetState, error) {
	defer r.begin()()
	clone := r.data.state
	return &clone, nil
}

func (r *fakeRepo) SaveFleetState(ctx context.Context, state *entity.FleetState) error {
	defer r.begin()()
	r.data.state = *state
	return nil
}

func (r *fakeRepo) FindActivity(ctx context.Context, id string) (*entity.Activity, error) {
	defer r.begin()()
	item, ok := r.data.activities[id]
	if !ok {
		return nil, entity.ErrNotFound
	}
	return copyActivity(item), nil
}

func (r *fakeRepo) ListActivities(ctx context.Context) ([]*entity.Activity, error) {
	defer r.begin()()
	items := make([]*entity.Activity, 0, len(r.data.activities))
	for _, item := range r.data.activities {
		items = append(items, copyActivity(item))
	}
	return items, nil
}

func (r *fakeRepo) SaveActivity(ctx context.Context, activity *entity.Activity) error {
	defer r.begin()()
	r.data.activities[activity.ID] = copyActivity(activity)
	return nil
}

func (r *fakeRepo) ListBaselines(ctx context.Context, activityID string) ([]*entity.Baseline, error) {
	defer r.begin()()
	items := make([]*entity.Baseline, 0, len(r.data.baselines[activityID]))
	for _, item := range r.data.baselines[activityID] {
		clone := *item
		items = append(items, &clone)
	}
	return items, nil
}

func (r *fakeRepo) SaveBaseline(ctx context.Context, baseline *entity.Baseline) error {
	defer r.begin()()
	clone := *baseline
	r.data.baselines[baseline.ActivityID] = append(r.data.baselines[baseline.ActivityID], &clone)
	return nil
}

func (r *fakeRepo) ListReleasesByActivity(ctx context.Context, activityID string) ([]*entity.Release, error) {
	defer r.begin()()
	items := make([]*entity.Release, 0)
	for _, byVersion := range r.data.releases {
		for _, release := range byVersion {
			if release.ActivityID == activityID {
				items = append(items, copyRelease(release))
			}
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].VehicleID == items[j].VehicleID {
			return items[i].Version < items[j].Version
		}
		return items[i].VehicleID < items[j].VehicleID
	})
	return items, nil
}

func (r *fakeRepo) ListLatestReleasesByVehicle(ctx context.Context) (map[string]*entity.Release, error) {
	defer r.begin()()
	output := map[string]*entity.Release{}
	for vehicleID, byVersion := range r.data.releases {
		for _, release := range byVersion {
			current, ok := output[vehicleID]
			if !ok || release.Version > current.Version {
				output[vehicleID] = copyRelease(release)
			}
		}
	}
	return output, nil
}

func (r *fakeRepo) FindRelease(ctx context.Context, vehicleID string, version int) (*entity.Release, error) {
	defer r.begin()()
	byVersion, ok := r.data.releases[vehicleID]
	if !ok {
		return nil, entity.ErrNotFound
	}
	release, ok := byVersion[version]
	if !ok {
		return nil, entity.ErrNotFound
	}
	return copyRelease(release), nil
}

func (r *fakeRepo) ListWaitingReleasesBefore(ctx context.Context, deadline time.Time) ([]*entity.Release, error) {
	defer r.begin()()
	items := make([]*entity.Release, 0)
	for _, byVersion := range r.data.releases {
		for _, release := range byVersion {
			if release.Status == entity.ReleaseWaiting && !release.DispatchedAt.After(deadline) {
				items = append(items, copyRelease(release))
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].DispatchedAt.Before(items[j].DispatchedAt) })
	return items, nil
}

func (r *fakeRepo) SaveRelease(ctx context.Context, release *entity.Release) error {
	defer r.begin()()
	if r.data.releases[release.VehicleID] == nil {
		r.data.releases[release.VehicleID] = map[int]*entity.Release{}
	}
	r.data.releases[release.VehicleID][release.Version] = copyRelease(release)
	return nil
}

func (r *fakeRepo) SaveReceipt(ctx context.Context, receipt *entity.Receipt) error {
	defer r.begin()()
	clone := *receipt
	r.data.receipts = append(r.data.receipts, &clone)
	return nil
}

func (r *fakeRepo) ListReceiptsByActivity(ctx context.Context, activityID string) ([]*entity.Receipt, error) {
	defer r.begin()()
	items := make([]*entity.Receipt, 0)
	for _, receipt := range r.data.receipts {
		if receipt.ActivityID == activityID {
			clone := *receipt
			items = append(items, &clone)
		}
	}
	return items, nil
}

func (r *fakeRepo) SaveRejectedReceipt(ctx context.Context, receipt *entity.RejectedReceipt) error {
	defer r.begin()()
	clone := *receipt
	r.data.rejected = append(r.data.rejected, &clone)
	return nil
}

func (r *fakeRepo) ListRejectedReceipts(ctx context.Context, activityID string, vehicleID string) ([]*entity.RejectedReceipt, error) {
	defer r.begin()()
	items := make([]*entity.RejectedReceipt, 0)
	for _, receipt := range r.data.rejected {
		if activityID != "" && receipt.ActivityID != activityID {
			continue
		}
		if vehicleID != "" && receipt.VehicleID != vehicleID {
			continue
		}
		clone := *receipt
		items = append(items, &clone)
	}
	return items, nil
}

func (r *fakeRepo) SaveTimeoutEvent(ctx context.Context, event *entity.TimeoutEvent) error {
	defer r.begin()()
	clone := *event
	r.data.timeouts = append(r.data.timeouts, &clone)
	return nil
}

func (r *fakeRepo) ResolveTimeoutEvents(ctx context.Context, releaseID string, resolvedAt time.Time) error {
	defer r.begin()()
	for _, event := range r.data.timeouts {
		if event.ReleaseID == releaseID && event.ResolvedAt == nil {
			value := resolvedAt
			event.ResolvedAt = &value
		}
	}
	return nil
}

func (r *fakeRepo) ListTimeoutEventsByActivity(ctx context.Context, activityID string) ([]*entity.TimeoutEvent, error) {
	defer r.begin()()
	items := make([]*entity.TimeoutEvent, 0)
	for _, event := range r.data.timeouts {
		if event.ActivityID == activityID {
			clone := *event
			items = append(items, &clone)
		}
	}
	return items, nil
}

func (r *fakeRepo) SavePull(ctx context.Context, pull *entity.PullRecord) error {
	defer r.begin()()
	clone := *pull
	r.data.pulls = append(r.data.pulls, &clone)
	return nil
}

func (r *fakeRepo) ListPullsByVehicle(ctx context.Context, vehicleID string, limit int) ([]*entity.PullRecord, error) {
	defer r.begin()()
	items := make([]*entity.PullRecord, 0)
	for _, pull := range r.data.pulls {
		if pull.VehicleID == vehicleID {
			clone := *pull
			items = append(items, &clone)
		}
	}
	return items, nil
}

func (r *fakeRepo) SaveAudit(ctx context.Context, audit *entity.Audit) error {
	defer r.begin()()
	clone := *audit
	r.data.audits = append(r.data.audits, &clone)
	return nil
}

func (r *fakeRepo) ListAudits(ctx context.Context, activityID string, limit int) ([]*entity.Audit, error) {
	defer r.begin()()
	items := make([]*entity.Audit, 0)
	for _, audit := range r.data.audits {
		if activityID != "" && audit.TargetID != activityID && audit.TargetType != "release" {
			continue
		}
		clone := *audit
		items = append(items, &clone)
	}
	return items, nil
}
