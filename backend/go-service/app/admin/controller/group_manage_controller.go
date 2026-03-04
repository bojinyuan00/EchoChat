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

// GroupManageController 管理端群组管理控制器
type GroupManageController struct {
	groupManageService *service.GroupManageService
}

// NewGroupManageController 创建群组管理控制器实例
func NewGroupManageController(svc *service.GroupManageService) *GroupManageController {
	return &GroupManageController{groupManageService: svc}
}

// GetGroupList 获取群组列表
// GET /api/v1/admin/groups?page=1&page_size=20&keyword=xxx
func (ctl *GroupManageController) GetGroupList(c *gin.Context) {
	funcName := "admin.group_manage_controller.GetGroupList"
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	list, total, err := ctl.groupManageService.ListGroups(ctx, page, pageSize, keyword)
	if err != nil {
		logs.Error(ctx, funcName, "获取群组列表失败", zap.Error(err))
		utils.ResponseError(c, "获取群组列表失败")
		return
	}

	logs.Info(ctx, funcName, "获取群组列表成功",
		zap.Int64("total", total), zap.Int("page", page))
	utils.ResponseOK(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// GetGroupDetail 获取群组详情
// GET /api/v1/admin/groups/:id
func (ctl *GroupManageController) GetGroupDetail(c *gin.Context) {
	funcName := "admin.group_manage_controller.GetGroupDetail"
	ctx := c.Request.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "ID 格式错误")
		return
	}

	detail, err := ctl.groupManageService.GetGroupDetail(ctx, id)
	if err != nil {
		logs.Error(ctx, funcName, "获取群组详情失败",
			zap.Int64("group_id", id), zap.Error(err))
		utils.ResponseError(c, "获取群组详情失败")
		return
	}

	logs.Info(ctx, funcName, "获取群组详情成功", zap.Int64("group_id", id))
	utils.ResponseOK(c, detail)
}

// DissolveGroup 解散群聊
// DELETE /api/v1/admin/groups/:id
func (ctl *GroupManageController) DissolveGroup(c *gin.Context) {
	funcName := "admin.group_manage_controller.DissolveGroup"
	ctx := c.Request.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "ID 格式错误")
		return
	}

	if err := ctl.groupManageService.DissolveGroup(ctx, id); err != nil {
		logs.Error(ctx, funcName, "解散群聊失败",
			zap.Int64("group_id", id), zap.Error(err))
		utils.ResponseError(c, "解散群聊失败")
		return
	}

	logs.Info(ctx, funcName, "解散群聊成功", zap.Int64("group_id", id))
	utils.ResponseOK(c, nil)
}
