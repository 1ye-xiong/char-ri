package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/BwCloudWeGo/bw-cli/internal/gateway/handler"
)

// registerFleetLogRoutes 注册 /api/v1/fleetlog 接口。
func registerFleetLogRoutes(v1 *gin.RouterGroup, fleetLogHandler *handler.FleetLogHandler) {
	routes := v1.Group("/fleetlog")
	routes.POST("/login", fleetLogHandler.Login)

	routes.GET("/applications", fleetLogHandler.ListApplications)
	routes.POST("/applications", fleetLogHandler.CreateApplication)
	routes.PUT("/applications/:id", fleetLogHandler.UpdateApplication)
	routes.POST("/applications/:id/submit", fleetLogHandler.SubmitApplication)
	routes.POST("/applications/:id/approve", fleetLogHandler.ApproveApplication)
	routes.POST("/applications/:id/reject", fleetLogHandler.RejectApplication)

	routes.GET("/activities", fleetLogHandler.ListActivities)
	routes.POST("/activities", fleetLogHandler.CreateActivity)
	routes.GET("/current-activity", fleetLogHandler.GetCurrentActivity)
	routes.GET("/activities/:id", fleetLogHandler.GetActivity)
	routes.POST("/activities/:id/pilot", fleetLogHandler.StartPilot)
	routes.POST("/activities/:id/promote", fleetLogHandler.ConfirmPromotion)
	routes.POST("/activities/:id/stop", fleetLogHandler.StopActivity)
	routes.POST("/activities/:id/restore/approve", fleetLogHandler.ApproveRestore)
	routes.POST("/activities/:id/restore/start", fleetLogHandler.StartRestore)
	routes.POST("/activities/:id/complete", fleetLogHandler.MarkCompleted)
	routes.POST("/activities/:id/restored", fleetLogHandler.MarkRestored)

	routes.GET("/vehicles", fleetLogHandler.ListVehicles)
	routes.POST("/vehicles/:id/online", fleetLogHandler.SetVehicleOnline)
	routes.POST("/vehicle/pull", fleetLogHandler.PullConfig)
	routes.POST("/vehicle/receipts", fleetLogHandler.ReportReceipt)
	routes.GET("/rejected-receipts", fleetLogHandler.ListRejectedReceipts)
	routes.GET("/audits", fleetLogHandler.ListAudits)
}

// registerFleetLogPages 注册可操作页面。
func registerFleetLogPages(r *gin.Engine) {
	r.GET("/fleetlog", func(c *gin.Context) {
		c.HTML(http.StatusOK, "console.html", nil)
	})
	r.GET("/fleetlog/vehicles", func(c *gin.Context) {
		c.HTML(http.StatusOK, "vehicles.html", nil)
	})
}
