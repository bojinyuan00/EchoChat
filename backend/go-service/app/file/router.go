// Package file 文件上传模块路由注册
package file

import (
	"github.com/echochat/backend/app/file/controller"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册文件上传模块的所有路由（需要 JWT 中间件）
func RegisterRoutes(r *gin.Engine, ctrl *controller.FileController, jwtAuth gin.HandlerFunc) {
	authed := r.Group("/api/v1")
	authed.Use(jwtAuth)
	{
		authed.POST("/upload", ctrl.Upload)
	}
}
