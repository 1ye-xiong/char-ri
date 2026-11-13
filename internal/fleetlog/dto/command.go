// Package dto 定义 fleetlog 用例的入参与出参，隔离 entity 与传输层。
package dto

// LoginCommand 是登录入参。
type LoginCommand struct {
	Account  string
	Password string
}

// CreateApplicationCommand 是新建申请草稿入参。
type CreateApplicationCommand struct {
	Reason   string
	Interval int
}

// UpdateApplicationCommand 是修改草稿/退回申请入参。
type UpdateApplicationCommand struct {
	ID       string
	Reason   string
	Interval int
}

// CreateActivityCommand 是创建发布活动入参。
type CreateActivityCommand struct {
	ApplicationID string
	PilotVehicles []string
}

// StopActivityCommand 是停止活动入参。
type StopActivityCommand struct {
	ID     string
	Reason string
}

// PullConfigCommand 是车辆拉取入参。
type PullConfigCommand struct {
	VehicleID string
}

// ReportReceiptCommand 是车辆回执入参。
type ReportReceiptCommand struct {
	VehicleID     string
	Version       int
	Result        string
	FailureReason string
}
