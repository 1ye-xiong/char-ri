package repo

import "time"

type vehicleModel struct {
	ID                string    `gorm:"column:id;primaryKey;size:16"`
	Online            bool      `gorm:"column:online;not null"`
	EffectiveInterval int       `gorm:"column:effective_interval;not null"`
	EffectiveVersion  int       `gorm:"column:effective_version;not null"`
	CurrentVersion    int       `gorm:"column:current_version;not null"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (vehicleModel) TableName() string { return "fleet_vehicles" }

type applicationModel struct {
	ID            string     `gorm:"column:id;primaryKey;size:36"`
	VersionNo     int        `gorm:"column:version_no;not null"`
	RequesterID   string     `gorm:"column:requester_id;size:64;index"`
	RequesterName string     `gorm:"column:requester_name;size:64"`
	Reason        string     `gorm:"column:reason;type:text"`
	Interval      int        `gorm:"column:interval_seconds;not null"`
	Status        string     `gorm:"column:status;size:20;index"`
	RejectReason  string     `gorm:"column:reject_reason;type:text"`
	DecidedBy     string     `gorm:"column:decided_by;size:64"`
	SubmittedAt   *time.Time `gorm:"column:submitted_at"`
	DecidedAt     *time.Time `gorm:"column:decided_at"`
	ActivityID    string     `gorm:"column:activity_id;size:36"`
	CreatedAt     time.Time  `gorm:"column:created_at;index"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (applicationModel) TableName() string { return "fleet_applications" }

type applicationSnapshotModel struct {
	ID            string    `gorm:"column:id;primaryKey;size:36"`
	ApplicationID string    `gorm:"column:application_id;size:36;index"`
	VersionNo     int       `gorm:"column:version_no;not null"`
	Reason        string    `gorm:"column:reason;type:text"`
	Interval      int       `gorm:"column:interval_seconds;not null"`
	Status        string    `gorm:"column:status;size:20"`
	RejectReason  string    `gorm:"column:reject_reason;type:text"`
	DecidedBy     string    `gorm:"column:decided_by;size:64"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (applicationSnapshotModel) TableName() string { return "fleet_application_snapshots" }

type activityModel struct {
	ID                string     `gorm:"column:id;primaryKey;size:36"`
	ApplicationID     string     `gorm:"column:application_id;size:36;uniqueIndex"`
	Interval          int        `gorm:"column:interval_seconds;not null"`
	PilotVehicles     string     `gorm:"column:pilot_vehicles;type:text"`
	PromotionVehicles string     `gorm:"column:promotion_vehicles;type:text"`
	Status            string     `gorm:"column:status;size:24;index"`
	CreatedBy         string     `gorm:"column:created_by;size:64"`
	CreatedAt         time.Time  `gorm:"column:created_at;index"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	StoppedBy         string     `gorm:"column:stopped_by;size:64"`
	StoppedAt         *time.Time `gorm:"column:stopped_at"`
	StopScope         string     `gorm:"column:stop_scope;type:text"`
	RestoreApprovedBy string     `gorm:"column:restore_approved_by;size:64"`
	RestoreApprovedAt *time.Time `gorm:"column:restore_approved_at"`
	RestoreStartedBy  string     `gorm:"column:restore_started_by;size:64"`
	RestoreStartedAt  *time.Time `gorm:"column:restore_started_at"`
	CompletedBy       string     `gorm:"column:completed_by;size:64"`
	CompletedAt       *time.Time `gorm:"column:completed_at"`
	RestoredBy        string     `gorm:"column:restored_by;size:64"`
	RestoredAt        *time.Time `gorm:"column:restored_at"`
}

func (activityModel) TableName() string { return "fleet_activities" }

type baselineModel struct {
	ActivityID string    `gorm:"column:activity_id;primaryKey;size:36"`
	VehicleID  string    `gorm:"column:vehicle_id;primaryKey;size:16"`
	Interval   int       `gorm:"column:interval_seconds;not null"`
	Version    int       `gorm:"column:version;not null"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (baselineModel) TableName() string { return "fleet_baselines" }

type releaseModel struct {
	ID                string     `gorm:"column:id;primaryKey;size:36"`
	ActivityID        string     `gorm:"column:activity_id;size:36;index"`
	VehicleID         string     `gorm:"column:vehicle_id;size:16;uniqueIndex:uk_release_vehicle_version,priority:1"`
	Version           int        `gorm:"column:version;not null;uniqueIndex:uk_release_vehicle_version,priority:2"`
	Batch             string     `gorm:"column:batch;size:16"`
	ExpectedInterval  int        `gorm:"column:expected_interval;not null"`
	BaselineInterval  int        `gorm:"column:baseline_interval;not null"`
	Status            string     `gorm:"column:status;size:16;index"`
	FailureReason     string     `gorm:"column:failure_reason;type:text"`
	DispatchedAt      time.Time  `gorm:"column:dispatched_at;index"`
	ReceiptAt         *time.Time `gorm:"column:receipt_at"`
	TimeoutAt         *time.Time `gorm:"column:timeout_at"`
	Late              bool       `gorm:"column:late;not null"`
	DuplicateReceipts int        `gorm:"column:duplicate_receipts;not null"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

func (releaseModel) TableName() string { return "fleet_releases" }

type receiptModel struct {
	ID            string    `gorm:"column:id;primaryKey;size:36"`
	ReleaseID     string    `gorm:"column:release_id;size:36;index"`
	ActivityID    string    `gorm:"column:activity_id;size:36;index"`
	VehicleID     string    `gorm:"column:vehicle_id;size:16;index"`
	Version       int       `gorm:"column:version;not null"`
	Result        string    `gorm:"column:result;size:16"`
	FailureReason string    `gorm:"column:failure_reason;type:text"`
	ReportedAt    time.Time `gorm:"column:reported_at"`
	AcceptedAt    time.Time `gorm:"column:accepted_at"`
}

func (receiptModel) TableName() string { return "fleet_receipts" }

type rejectedReceiptModel struct {
	ID         string    `gorm:"column:id;primaryKey;size:36"`
	ActivityID string    `gorm:"column:activity_id;size:36;index"`
	VehicleID  string    `gorm:"column:vehicle_id;size:16;index"`
	Version    int       `gorm:"column:version;not null"`
	Reason     string    `gorm:"column:reason;size:32;index"`
	Detail     string    `gorm:"column:detail;type:text"`
	ReportedAt time.Time `gorm:"column:reported_at;index"`
}

func (rejectedReceiptModel) TableName() string { return "fleet_rejected_receipts" }

type timeoutEventModel struct {
	ID         string     `gorm:"column:id;primaryKey;size:36"`
	ReleaseID  string     `gorm:"column:release_id;size:36;index"`
	ActivityID string     `gorm:"column:activity_id;size:36;index"`
	VehicleID  string     `gorm:"column:vehicle_id;size:16"`
	Version    int        `gorm:"column:version;not null"`
	TimedOutAt time.Time  `gorm:"column:timed_out_at"`
	ResolvedAt *time.Time `gorm:"column:resolved_at"`
}

func (timeoutEventModel) TableName() string { return "fleet_timeout_events" }

type pullRecordModel struct {
	ID        string    `gorm:"column:id;primaryKey;size:36"`
	VehicleID string    `gorm:"column:vehicle_id;size:16;index"`
	Version   int       `gorm:"column:version;not null"`
	Interval  int       `gorm:"column:interval_seconds;not null"`
	ReleaseID string    `gorm:"column:release_id;size:36"`
	PulledAt  time.Time `gorm:"column:pulled_at;index"`
}

func (pullRecordModel) TableName() string { return "fleet_pull_records" }

type auditModel struct {
	ID         string    `gorm:"column:id;primaryKey;size:36"`
	ActorID    string    `gorm:"column:actor_id;size:64;index"`
	ActorRole  string    `gorm:"column:actor_role;size:8"`
	Action     string    `gorm:"column:action;size:48;index"`
	TargetType string    `gorm:"column:target_type;size:24"`
	TargetID   string    `gorm:"column:target_id;size:36;index"`
	Detail     string    `gorm:"column:detail;type:text"`
	CreatedAt  time.Time `gorm:"column:created_at;index"`
}

func (auditModel) TableName() string { return "fleet_audits" }

type fleetStateModel struct {
	ID             string    `gorm:"column:id;primaryKey;size:32"`
	OpenActivityID string    `gorm:"column:open_activity_id;size:36"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (fleetStateModel) TableName() string { return "fleet_states" }
