// Package controller 会议模块 HTTP / WS 入口
package controller

import (
	"context"
	"encoding/json"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/meeting/service"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"go.uber.org/zap"
)

// MeetingWSHandler 会议 WS 事件入口
// Task 6 落地：注册设计 §6.3 的 11 个 meeting.* 事件到 Hub 路由表
// 薄层设计：
//   - 仅负责 JSON Unmarshal → 调 SignalService 业务方法 → 组装 ACK 响应
//   - 权限校验全部下沉到 SignalService（service 层统一 assertIsActiveParticipant / assertIsHost）
//   - 广播副作用由 SignalService 自行发起（ACK 不携带广播数据）
type MeetingWSHandler struct {
	signalSvc *service.MeetingSignalService
	hub       *ws.Hub
}

// NewMeetingWSHandler 构造 WS Handler 并自动注册事件到 Hub
// 与 im.handler.EventHandler 保持一致的风格：构造时注册，启动时由 DI 容器持有
func NewMeetingWSHandler(signalSvc *service.MeetingSignalService, hub *ws.Hub) *MeetingWSHandler {
	h := &MeetingWSHandler{
		signalSvc: signalSvc,
		hub:       hub,
	}
	h.registerEvents()
	return h
}

// registerEvents 将所有 meeting.* C→S 事件注册到 Hub 路由表
// S→C 事件（ended/joined/left/host.changed/producer.new 等）不在此处注册，由 broadcaster 发出
func (h *MeetingWSHandler) registerEvents() {
	// 房间组
	h.hub.RegisterEvent(constants.MeetingWSEventRoomJoin, h.handleRoomJoin)
	h.hub.RegisterEvent(constants.MeetingWSEventRoomLeave, h.handleRoomLeave)
	// 成员组
	h.hub.RegisterEvent(constants.MeetingWSEventMemberStateChange, h.handleMemberStateChange)
	// 媒体组（5 个）
	h.hub.RegisterEvent(constants.MeetingWSEventTransportCreate, h.handleTransportCreate)
	h.hub.RegisterEvent(constants.MeetingWSEventTransportConnect, h.handleTransportConnect)
	h.hub.RegisterEvent(constants.MeetingWSEventProduceStart, h.handleProduceStart)
	h.hub.RegisterEvent(constants.MeetingWSEventConsumeStart, h.handleConsumeStart)
	h.hub.RegisterEvent(constants.MeetingWSEventProducerClose, h.handleProducerClose)
}

// ====== 通用工具 ======

// simpleRoomPayload 房间组仅需 room_code 的请求体
type simpleRoomPayload struct {
	RoomCode string `json:"room_code"`
}

// sendACK 统一发送 ACK 响应
func (h *MeetingWSHandler) sendACK(client *ws.Client, msg *ws.Message, code int, message string, data interface{}) {
	resp := ws.NewResponse(msg.Event, msg.Seq, code, message, data)
	bytes, err := ws.MarshalResponse(resp)
	if err != nil {
		logs.Error(context.Background(), "controller.meeting_ws_handler.sendACK", "序列化 ACK 失败",
			zap.String("event", msg.Event), zap.Error(err))
		return
	}
	client.Send(bytes)
}

// unmarshal 通用反序列化 + 错误 ACK
func (h *MeetingWSHandler) unmarshal(client *ws.Client, msg *ws.Message, target interface{}) bool {
	if err := json.Unmarshal(msg.Data, target); err != nil {
		logs.Warn(nil, "controller.meeting_ws_handler.unmarshal", "反序列化失败",
			zap.String("event", msg.Event),
			zap.Int64("user_id", client.UserID),
			zap.Error(err))
		h.sendACK(client, msg, -1, "请求参数格式错误", nil)
		return false
	}
	return true
}

// ====== 房间组 ======

// handleRoomJoin 处理 meeting.room.join
func (h *MeetingWSHandler) handleRoomJoin(client *ws.Client, msg *ws.Message) {
	var payload simpleRoomPayload
	if !h.unmarshal(client, msg, &payload) {
		return
	}
	if err := h.signalSvc.OnRoomJoin(context.Background(), client.UserID, payload.RoomCode); err != nil {
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}
	h.sendACK(client, msg, 0, "ok", nil)
}

// handleRoomLeave 处理 meeting.room.leave
func (h *MeetingWSHandler) handleRoomLeave(client *ws.Client, msg *ws.Message) {
	var payload simpleRoomPayload
	if !h.unmarshal(client, msg, &payload) {
		return
	}
	if err := h.signalSvc.OnRoomLeave(context.Background(), client.UserID, payload.RoomCode); err != nil {
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}
	h.sendACK(client, msg, 0, "ok", nil)
}

// ====== 成员组 ======

// handleMemberStateChange 处理 meeting.member.state.changed
func (h *MeetingWSHandler) handleMemberStateChange(client *ws.Client, msg *ws.Message) {
	var payload service.MemberStateChangePayload
	if !h.unmarshal(client, msg, &payload) {
		return
	}
	if err := h.signalSvc.OnMemberStateChanged(context.Background(), client.UserID, &payload); err != nil {
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}
	h.sendACK(client, msg, 0, "ok", nil)
}

// ====== 媒体组 ======

// handleTransportCreate 处理 meeting.transport.create
func (h *MeetingWSHandler) handleTransportCreate(client *ws.Client, msg *ws.Message) {
	var payload service.TransportCreatePayload
	if !h.unmarshal(client, msg, &payload) {
		return
	}
	info, err := h.signalSvc.OnTransportCreate(context.Background(), client.UserID, &payload)
	if err != nil {
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}
	h.sendACK(client, msg, 0, "ok", info)
}

// handleTransportConnect 处理 meeting.transport.connect
func (h *MeetingWSHandler) handleTransportConnect(client *ws.Client, msg *ws.Message) {
	var payload service.TransportConnectPayload
	if !h.unmarshal(client, msg, &payload) {
		return
	}
	if err := h.signalSvc.OnTransportConnect(context.Background(), client.UserID, &payload); err != nil {
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}
	h.sendACK(client, msg, 0, "ok", nil)
}

// handleProduceStart 处理 meeting.produce.start
func (h *MeetingWSHandler) handleProduceStart(client *ws.Client, msg *ws.Message) {
	var payload service.ProduceStartPayload
	if !h.unmarshal(client, msg, &payload) {
		return
	}
	result, err := h.signalSvc.OnProduceStart(context.Background(), client.UserID, &payload)
	if err != nil {
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}
	h.sendACK(client, msg, 0, "ok", result)
}

// handleConsumeStart 处理 meeting.consume.start
func (h *MeetingWSHandler) handleConsumeStart(client *ws.Client, msg *ws.Message) {
	var payload service.ConsumeStartPayload
	if !h.unmarshal(client, msg, &payload) {
		return
	}
	info, err := h.signalSvc.OnConsumeStart(context.Background(), client.UserID, &payload)
	if err != nil {
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}
	h.sendACK(client, msg, 0, "ok", info)
}

// handleProducerClose 处理 meeting.producer.close
func (h *MeetingWSHandler) handleProducerClose(client *ws.Client, msg *ws.Message) {
	var payload service.ProducerClosePayload
	if !h.unmarshal(client, msg, &payload) {
		return
	}
	if err := h.signalSvc.OnProducerClose(context.Background(), client.UserID, &payload); err != nil {
		h.sendACK(client, msg, -1, err.Error(), nil)
		return
	}
	h.sendACK(client, msg, 0, "ok", nil)
}
