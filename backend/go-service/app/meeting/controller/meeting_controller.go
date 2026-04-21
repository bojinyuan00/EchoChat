// Package controller 提供 meeting 模块的 HTTP 接口
package controller

import (
	"net/http"

	"github.com/echochat/backend/app/meeting/service"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

// MeetingController 会议 REST 控制器
// Task 4 骨架阶段：所有 handler 均返回 501 Not Implemented，便于路由自测与前端 mock
// Task 5/6/7 将在此填充请求解析、业务调用与错误映射
type MeetingController struct {
	meetingService *service.MeetingService
}

// NewMeetingController 创建 MeetingController 实例
func NewMeetingController(meetingService *service.MeetingService) *MeetingController {
	return &MeetingController{meetingService: meetingService}
}

// responseNotImplemented Task 4 占位响应，Task 5/6/7 逐接口替换为真实实现
func responseNotImplemented(c *gin.Context, endpoint string) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"code":     http.StatusNotImplemented,
		"message":  "接口尚未实现",
		"endpoint": endpoint,
	})
}

// requireUserID 统一的当前用户取值，失败直接写 401 并返回 false
func requireUserID(c *gin.Context) (int64, bool) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return 0, false
	}
	return userID, true
}

// CreateRoom POST /api/v1/meeting/rooms
func (ctl *MeetingController) CreateRoom(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "POST /api/v1/meeting/rooms")
}

// GetRoom GET /api/v1/meeting/rooms/:code
func (ctl *MeetingController) GetRoom(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "GET /api/v1/meeting/rooms/:code")
}

// JoinRoom POST /api/v1/meeting/rooms/:code/join
func (ctl *MeetingController) JoinRoom(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "POST /api/v1/meeting/rooms/:code/join")
}

// LeaveRoom POST /api/v1/meeting/rooms/:code/leave
func (ctl *MeetingController) LeaveRoom(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "POST /api/v1/meeting/rooms/:code/leave")
}

// EndRoom POST /api/v1/meeting/rooms/:code/end
func (ctl *MeetingController) EndRoom(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "POST /api/v1/meeting/rooms/:code/end")
}

// ListMyMeetings GET /api/v1/meeting/rooms?role=host|participant
func (ctl *MeetingController) ListMyMeetings(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "GET /api/v1/meeting/rooms")
}

// TransferHost POST /api/v1/meeting/rooms/:code/transfer-host
func (ctl *MeetingController) TransferHost(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "POST /api/v1/meeting/rooms/:code/transfer-host")
}

// KickMember POST /api/v1/meeting/rooms/:code/kick
func (ctl *MeetingController) KickMember(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "POST /api/v1/meeting/rooms/:code/kick")
}

// InviteUsers POST /api/v1/meeting/rooms/:code/invite
func (ctl *MeetingController) InviteUsers(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "POST /api/v1/meeting/rooms/:code/invite")
}

// RedeemInvite POST /api/v1/meeting/invites/:token/redeem
func (ctl *MeetingController) RedeemInvite(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "POST /api/v1/meeting/invites/:token/redeem")
}

// SendChat POST /api/v1/meeting/rooms/:code/chats
func (ctl *MeetingController) SendChat(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "POST /api/v1/meeting/rooms/:code/chats")
}

// ListChats GET /api/v1/meeting/rooms/:code/chats?after_id=&limit=
func (ctl *MeetingController) ListChats(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	responseNotImplemented(c, "GET /api/v1/meeting/rooms/:code/chats")
}
