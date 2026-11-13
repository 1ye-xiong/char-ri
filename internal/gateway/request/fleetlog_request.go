package request

// FleetLogLoginRequest 是 fleetlog 登录载荷。
type FleetLogLoginRequest struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// CreateFleetLogApplicationRequest 是新建申请载荷。
type CreateFleetLogApplicationRequest struct {
	Reason          string `json:"reason"`
	IntervalSeconds int32  `json:"interval_seconds"`
}

// UpdateFleetLogApplicationRequest 是修改申请载荷。
type UpdateFleetLogApplicationRequest struct {
	Reason          string `json:"reason"`
	IntervalSeconds int32  `json:"interval_seconds"`
}

// RejectFleetLogApplicationRequest 是退回申请载荷。
type RejectFleetLogApplicationRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// CreateFleetLogActivityRequest 是创建活动载荷。
type CreateFleetLogActivityRequest struct {
	ApplicationID string   `json:"application_id" binding:"required"`
	PilotVehicles []string `json:"pilot_vehicles"`
}

// StopFleetLogActivityRequest 是停止活动载荷。
type StopFleetLogActivityRequest struct {
	Reason string `json:"reason"`
}

// SetFleetLogVehicleOnlineRequest 是切换车辆在线状态载荷。
type SetFleetLogVehicleOnlineRequest struct {
	Online bool `json:"online"`
}

// FleetLogVehiclePullRequest 是车辆拉取载荷。
type FleetLogVehiclePullRequest struct {
	VehicleID string `json:"vehicle_id"`
}

// FleetLogVehicleReceiptRequest 是车辆回执载荷。
type FleetLogVehicleReceiptRequest struct {
	VehicleID     string `json:"vehicle_id"`
	Version       int32  `json:"version"`
	Result        string `json:"result"`
	FailureReason string `json:"failure_reason"`
}
