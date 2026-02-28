// Package router 主路由汇总入口
// 不含任何具体路由定义，仅调用各模块的 RegisterRoutes 函数
// 新增模块时在 Setup 函数中添加一行调用即可
package router

import (
	"time"

	"github.com/echochat/backend/app/auth"
	"github.com/echochat/backend/app/provider"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

// Setup 注册所有路由
// 包含健康检查和各业务模块的路由注册
func Setup(engine *gin.Engine, app *provider.App) {
	// 健康检查（不经过业务中间件）
	engine.GET("/health", func(c *gin.Context) {
		utils.ResponseOK(c, gin.H{
			"status":  "ok",
			"service": "echochat",
			"time":    time.Now().Format("2006-01-02 15:04:05"),
		})
	})

	// JWT 认证中间件实例
	jwtAuth := middleware.JWTAuth(&app.Config.JWT)

	// --- 各模块路由注册 ---
	auth.RegisterRoutes(engine, app.AuthController, app.AdminAuthController, jwtAuth)

	// [未来] im.RegisterRoutes(engine, app.ImController, jwtAuth)
	// [未来] meeting.RegisterRoutes(engine, app.MeetingController, jwtAuth)
	// [未来] contact.RegisterRoutes(engine, app.ContactController, jwtAuth)
	// [未来] notify.RegisterRoutes(engine, app.NotifyController, jwtAuth)
	// [未来] admin.RegisterRoutes(engine, app.AdminController, jwtAuth)
}
