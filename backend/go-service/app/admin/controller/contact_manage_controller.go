package controller

import (
	"strconv"

	"github.com/echochat/backend/app/admin/service"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

type ContactManageController struct {
	contactManageService *service.ContactManageService
}

func NewContactManageController(contactManageService *service.ContactManageService) *ContactManageController {
	return &ContactManageController{contactManageService: contactManageService}
}

// GetAllContacts GET /api/v1/admin/contacts
func (ctl *ContactManageController) GetAllContacts(c *gin.Context) {
	ctx := c.Request.Context()
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	contacts, total, err := ctl.contactManageService.GetAllFriendships(ctx, page, pageSize)
	if err != nil {
		utils.ResponseError(c, "获取好友关系列表失败")
		return
	}
	utils.ResponseOK(c, gin.H{
		"list":  contacts,
		"total": total,
	})
}

// DeleteContact DELETE /api/v1/admin/contacts/:id
func (ctl *ContactManageController) DeleteContact(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "ID 格式错误")
		return
	}

	if err := ctl.contactManageService.DeleteFriendship(ctx, id); err != nil {
		utils.ResponseError(c, "删除好友关系失败")
		return
	}
	utils.ResponseOK(c, nil)
}
