package handler

import (
	fleetlogv1 "github.com/BwCloudWeGo/bw-cli/api/gen/fleetlog/v1"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/dto"
)

func toApplicationItem(item *dto.ApplicationDTO) *fleetlogv1.ApplicationItem {
	if item == nil {
		return nil
	}
	output := &fleetlogv1.ApplicationItem{
		Id:              item.ID,
		VersionNo:       int32(item.VersionNo),
		RequesterId:     item.RequesterID,
		RequesterName:   item.RequesterName,
		Reason:          item.Reason,
		IntervalSeconds: int32(item.Interval),
		Status:          item.Status,
		RejectReason:    item.RejectReason,
		DecidedBy:       item.DecidedBy,
		SubmittedAt:     item.SubmittedAt,
		DecidedAt:       item.DecidedAt,
		CreatedAt:       item.CreatedAt,
		ActivityId:      item.ActivityID,
		CanEdit:         item.CanEdit,
		CanSubmit:       item.CanSubmit,
		CanDecide:       item.CanDecide,
		History:         make([]*fleetlogv1.ApplicationVersionItem, 0, len(item.History)),
	}
	for _, version := range item.History {
		output.History = append(output.History, &fleetlogv1.ApplicationVersionItem{
			VersionNo:       int32(version.VersionNo),
			Reason:          version.Reason,
			IntervalSeconds: int32(version.Interval),
			Status:          version.Status,
			RejectReason:    version.RejectReason,
			DecidedBy:       version.DecidedBy,
			CreatedAt:       version.CreatedAt,
		})
	}
	return output
}

func toActivityItem(item *dto.ActivityDTO) *fleetlogv1.ActivityItem {
	if item == nil {
		return nil
	}
	return &fleetlogv1.ActivityItem{
		Id:                item.ID,
		ApplicationId:     item.ApplicationID,
		IntervalSeconds:   int32(item.Interval),
		PilotVehicles:     item.PilotVehicles,
		PromotionVehicles: item.PromotionVehicles,
		Status:            item.Status,
		CreatedBy:         item.CreatedBy,
		CreatedAt:         item.CreatedAt,
		StoppedBy:         item.StoppedBy,
		StoppedAt:         item.StoppedAt,
		StopScope:         item.StopScope,
		RestoreApprovedBy: item.RestoreApprovedBy,
		RestoreApprovedAt: item.RestoreApprovedAt,
		RestoreStartedBy:  item.RestoreStartedBy,
		RestoreStartedAt:  item.RestoreStartedAt,
		CompletedBy:       item.CompletedBy,
		CompletedAt:       item.CompletedAt,
		RestoredBy:        item.RestoredBy,
		RestoredAt:        item.RestoredAt,
	}
}

func toVehicleRow(item *dto.VehicleRowDTO) *fleetlogv1.VehicleRowItem {
	return &fleetlogv1.VehicleRowItem{
		VehicleId:         item.VehicleID,
		Released:          item.Released,
		Batch:             item.Batch,
		Version:           int32(item.Version),
		BaselineInterval:  int32(item.BaselineInterval),
		ExpectedInterval:  int32(item.ExpectedInterval),
		EffectiveInterval: int32(item.EffectiveInterval),
		EffectiveVersion:  int32(item.EffectiveVersion),
		Status:            item.Status,
		FailureReason:     item.FailureReason,
		DispatchedAt:      item.DispatchedAt,
		ReceiptAt:         item.ReceiptAt,
		TimeoutAt:         item.TimeoutAt,
		Late:              item.Late,
		DuplicateReceipts: int32(item.DuplicateReceipts),
	}
}

func toSummary(item *dto.SummaryDTO) *fleetlogv1.SummaryItem {
	if item == nil {
		return nil
	}
	return &fleetlogv1.SummaryItem{
		Total:            int32(item.Total),
		Released:         int32(item.Released),
		NotReleased:      int32(item.NotReleased),
		Success:          int32(item.Success),
		Failed:           int32(item.Failed),
		Timeout:          int32(item.Timeout),
		Waiting:          int32(item.Waiting),
		RejectedReceipts: int32(item.RejectedReceipt),
		Effective:        int32(item.Effective),
	}
}

func toActions(item *dto.ActionsDTO) *fleetlogv1.ActionsItem {
	if item == nil {
		return nil
	}
	return &fleetlogv1.ActionsItem{
		CanStartPilot:        item.CanStartPilot,
		CanPromote:           item.CanPromote,
		CanStop:              item.CanStop,
		CanApproveRestore:    item.CanApproveRestore,
		CanStartRestore:      item.CanStartRestore,
		CanMarkCompleted:     item.CanMarkCompleted,
		CanMarkRestored:      item.CanMarkRestored,
		PromoteBlockedReason: item.PromoteBlockedReason,
	}
}

func toRejectedReceiptItem(item *dto.RejectedReceiptDTO) *fleetlogv1.RejectedReceiptItem {
	return &fleetlogv1.RejectedReceiptItem{
		Id:         item.ID,
		VehicleId:  item.VehicleID,
		Version:    int32(item.Version),
		Reason:     item.Reason,
		Detail:     item.Detail,
		ReportedAt: item.ReportedAt,
	}
}

func toTimeoutEventItem(item *dto.TimeoutEventDTO) *fleetlogv1.TimeoutEventItem {
	return &fleetlogv1.TimeoutEventItem{
		VehicleId:  item.VehicleID,
		Version:    int32(item.Version),
		TimedOutAt: item.TimedOutAt,
		ResolvedAt: item.ResolvedAt,
	}
}

func toAuditItem(item *dto.AuditDTO) *fleetlogv1.AuditItem {
	return &fleetlogv1.AuditItem{
		ActorId:    item.ActorID,
		ActorRole:  item.ActorRole,
		Action:     item.Action,
		TargetType: item.TargetType,
		TargetId:   item.TargetID,
		Detail:     item.Detail,
		CreatedAt:  item.CreatedAt,
	}
}

func toVehicleItem(item *dto.VehicleDTO) *fleetlogv1.VehicleItem {
	return &fleetlogv1.VehicleItem{
		Id:                  item.ID,
		Online:              item.Online,
		EffectiveInterval:   int32(item.EffectiveInterval),
		EffectiveVersion:    int32(item.EffectiveVersion),
		CurrentVersion:      int32(item.CurrentVersion),
		ExpectedInterval:    int32(item.ExpectedInterval),
		LatestReleaseStatus: item.LatestStatus,
	}
}

func toActivityDetail(item *dto.ActivityDetailDTO) *fleetlogv1.ActivityDetailResponse {
	if item == nil {
		return &fleetlogv1.ActivityDetailResponse{}
	}
	output := &fleetlogv1.ActivityDetailResponse{
		Activity:         toActivityItem(item.Activity),
		Application:      toApplicationItem(item.Application),
		Rows:             make([]*fleetlogv1.VehicleRowItem, 0, len(item.Rows)),
		Summary:          toSummary(item.Summary),
		RejectedReceipts: make([]*fleetlogv1.RejectedReceiptItem, 0, len(item.Rejected)),
		TimeoutEvents:    make([]*fleetlogv1.TimeoutEventItem, 0, len(item.Timeouts)),
		Audits:           make([]*fleetlogv1.AuditItem, 0, len(item.Audits)),
		Actions:          toActions(item.Actions),
		TimeoutSeconds:   int32(item.TimeoutSecs),
		ServerTime:       item.ServerTime,
	}
	for _, row := range item.Rows {
		output.Rows = append(output.Rows, toVehicleRow(row))
	}
	for _, rejected := range item.Rejected {
		output.RejectedReceipts = append(output.RejectedReceipts, toRejectedReceiptItem(rejected))
	}
	for _, event := range item.Timeouts {
		output.TimeoutEvents = append(output.TimeoutEvents, toTimeoutEventItem(event))
	}
	for _, audit := range item.Audits {
		output.Audits = append(output.Audits, toAuditItem(audit))
	}
	return output
}
