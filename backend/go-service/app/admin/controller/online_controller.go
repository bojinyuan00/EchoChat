package controller

import (
	"github.com/echochat/backend/app/admin/service"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

type OnlineController struct {
	onlineManageService *service.OnlineManageService
}

func NewOnlineController(onlineManageService *service.OnlineManageService) *OnlineController {
	return &OnlineController{onlineManageService: onlineManageService}
}

// GetOnlineUsers GET /api/v1/admin/online/users
func (ctl *OnlineController) GetOnlineUsers(c *gin.Context) {
	ctx := c.Request.Context()
	userIDs, err := ctl.onlineManageService.GetOnlineUsers(ctx)
	if err != nil {
		utils.ResponseError(c, "获取在线用户列表失败")
		return
	}
	utils.ResponseOK(c, userIDs)
}

// GetOnlineCount GET /api/v1/admin/online/count
func (ctl *OnlineController) GetOnlineCount(c *gin.Context) {
	ctx := c.Request.Context()
	count, err := ctl.onlineManageService.GetOnlineCount(ctx)
	if err != nil {
		utils.ResponseError(c, "获取在线用户数失败")
		return
	}
	utils.ResponseOK(c, gin.H{"count": count})
}
