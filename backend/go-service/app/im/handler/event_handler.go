// Package handler 提供 IM 模块的 WebSocket 事件处理器
// 通过 Hub.RegisterEvent 注册到事件路由表，处理 im.* 系列事件
package handler

import (
	"context"
	"encoding/json"

	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/im/service"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"go.uber.org/zap"
)

// EventHandler IM 模块的 WS 事件处理器
type EventHandler struct {
	imService *service.IMService
	hub       *ws.Hub
}

// NewEventHandler 创建 EventHandler 实例并注册事件到 Hub 路由表
func NewEventHandler(imService *service.IMService, hub *ws.Hub) *EventHandler {
	h := &EventHandler{
		imService: imService,
		hub:       hub,
	}
	h.registerEvents()
	return h
}

// registerEvents 将 IM 事件处理函数注册到 Hub 路由表
func (h *EventHandler) registerEvents() {
	h.hub.RegisterEvent("im.message.send", h.handleSendMessage)
	h.hub.RegisterEvent("im.message.recall", h.handleRecallMessage)
	h.hub.RegisterEvent("im.conversation.read", h.handleMarkRead)
	h.hub.RegisterEvent("im.group.read", h.handleGroupRead)
	h.hub.RegisterEvent("im.typing", h.handleTyping)
}

// handleSendMessage 处理发送消息事件
func (h *EventHandler) handleSendMessage(client *ws.Client, msg *ws.Message) {
	funcName := "handler.event_handler.handleSendMessage"
	ctx := context.Background()

	var req dto.SendMessageRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		logs.Warn(ctx, funcName, "解析消息请求失败",
			zap.Int64("user_id", client.UserID), zap.Error(err))
		h.sendACK(client, msg, -1, "请求参数格式错误", nil)
		return
	}

	msgDTO, err := h.imService.SendMessage(ctx, client.UserID, &req)
	if err != nil {
		if err == service.ErrDuplicateMsg && msgDTO != nil {
			h.sendACK(client, msg, 0, "ok", msgDTO)
			return
		}
		logs.Warn(ctx, funcName, "发送消息失败",
			zap.Int64("user_id", client.UserID), zap.Error(err))
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}

	h.sendACK(client, msg, 0, "ok", msgDTO)
}

// handleRecallMessage 处理撤回消息事件
func (h *EventHandler) handleRecallMessage(client *ws.Client, msg *ws.Message) {
	funcName := "handler.event_handler.handleRecallMessage"
	ctx := context.Background()

	var req dto.RecallMessageRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		logs.Warn(ctx, funcName, "解析撤回请求失败",
			zap.Int64("user_id", client.UserID), zap.Error(err))
		h.sendACK(client, msg, -1, "请求参数格式错误", nil)
		return
	}

	if err := h.imService.RecallMessage(ctx, client.UserID, req.MessageID); err != nil {
		logs.Warn(ctx, funcName, "撤回消息失败",
			zap.Int64("user_id", client.UserID), zap.Error(err))
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}

	h.sendACK(client, msg, 0, "ok", nil)
}

// handleMarkRead 处理标记已读事件
func (h *EventHandler) handleMarkRead(client *ws.Client, msg *ws.Message) {
	funcName := "handler.event_handler.handleMarkRead"
	ctx := context.Background()

	var req dto.MarkReadRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		logs.Warn(ctx, funcName, "解析已读请求失败",
			zap.Int64("user_id", client.UserID), zap.Error(err))
		h.sendACK(client, msg, -1, "请求参数格式错误", nil)
		return
	}

	if err := h.imService.MarkRead(ctx, client.UserID, req.ConversationID); err != nil {
		logs.Warn(ctx, funcName, "标记已读失败",
			zap.Int64("user_id", client.UserID), zap.Error(err))
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}

	h.sendACK(client, msg, 0, "ok", nil)
}

// handleGroupRead 处理群聊消息标记已读事件
func (h *EventHandler) handleGroupRead(client *ws.Client, msg *ws.Message) {
	funcName := "handler.event_handler.handleGroupRead"
	ctx := context.Background()

	var req dto.MarkGroupReadRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		logs.Warn(ctx, funcName, "解析群已读请求失败",
			zap.Int64("user_id", client.UserID), zap.Error(err))
		h.sendACK(client, msg, -1, "请求参数格式错误", nil)
		return
	}

	if err := h.imService.MarkGroupMessagesRead(ctx, client.UserID, &req); err != nil {
		logs.Warn(ctx, funcName, "群已读标记失败",
			zap.Int64("user_id", client.UserID), zap.Error(err))
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}

	h.sendACK(client, msg, 0, "ok", nil)
}

// handleTyping 处理正在输入事件（通过 PubSub 转发给对方，支持跨实例）
func (h *EventHandler) handleTyping(client *ws.Client, msg *ws.Message) {
	funcName := "handler.event_handler.handleTyping"
	ctx := context.Background()

	var req dto.TypingRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		logs.Warn(ctx, funcName, "解析输入状态请求失败",
			zap.Int64("user_id", client.UserID), zap.Error(err))
		return
	}

	h.imService.PushTypingNotification(ctx, req.ConversationID, client.UserID)
}

// sendACK 发送 ACK 响应给客户端
func (h *EventHandler) sendACK(client *ws.Client, msg *ws.Message, code int, message string, data interface{}) {
	resp := ws.NewResponse(msg.Event, msg.Seq, code, message, data)
	bytes, err := ws.MarshalResponse(resp)
	if err != nil {
		logs.Error(context.Background(), "handler.event_handler.sendACK", "序列化 ACK 失败",
			zap.Error(err))
		return
	}
	client.Send(bytes)
}
