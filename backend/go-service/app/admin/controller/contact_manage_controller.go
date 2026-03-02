// Package controller 提供 admin 模块的 HTTP 接口处理
package controller

import (
	"strconv"

	"github.com/echochat/backend/app/admin/service"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ContactManageController 联系人关系管理控制器（管理端）
type ContactManageController struct {
	contactManageService *service.ContactManageService
}

// NewContactManageController 创建 ContactManageController 实例
func NewContactManageController(contactManageService *service.ContactManageService) *ContactManageController {
	return &ContactManageController{contactManageService: contactManageService}
}

// GetAllContacts 获取所有好友关系列表（分页）
// GET /api/v1/admin/contacts
func (ctl *ContactManageController) GetAllContacts(c *gin.Context) {
	funcName := "admin.contact_manage_controller.GetAllContacts"
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	contacts, total, err := ctl.contactManageService.GetAllFriendships(ctx, page, pageSize)
	if err != nil {
		logs.Error(ctx, funcName, "获取好友关系列表失败", zap.Error(err))
		utils.ResponseError(c, "获取好友关系列表失败")
		return
	}

	logs.Info(ctx, funcName, "获取好友关系列表成功",
		zap.Int64("total", total), zap.Int("page", page))
	utils.ResponseOK(c, gin.H{
		"list":  contacts,
		"total": total,
	})
}

// DeleteContact 删除好友关系
// DELETE /api/v1/admin/contacts/:id
func (ctl *ContactManageController) DeleteContact(c *gin.Context) {
	funcName := "admin.contact_manage_controller.DeleteContact"
	ctx := c.Request.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "ID 格式错误")
		return
	}

	if err := ctl.contactManageService.DeleteFriendship(ctx, id); err != nil {
		logs.Error(ctx, funcName, "删除好友关系失败",
			zap.Int64("friendship_id", id), zap.Error(err))
		utils.ResponseError(c, "删除好友关系失败")
		return
	}

	logs.Info(ctx, funcName, "删除好友关系成功", zap.Int64("friendship_id", id))
	utils.ResponseOK(c, nil)
}
