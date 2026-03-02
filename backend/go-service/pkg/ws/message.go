// Package ws 提供 WebSocket 通讯基础设施
// 包含消息协议定义、连接管理、Redis Pub/Sub 消息路由
package ws

import (
	"encoding/json"
	"time"
)

// Message 客户端 → 服务端的 WebSocket 消息
// 事件命名规范：{模块}.{对象}.{动作}，如 contact.request.send
type Message struct {
	Event string          `json:"event"`          // 事件类型
	Seq   int64           `json:"seq"`            // 客户端消息序号，用于 ACK 匹配
	Data  json.RawMessage `json:"data,omitempty"` // 业务数据（延迟解析）
	Time  string          `json:"time"`           // 发送时间
}

// Response 服务端 → 客户端的 ACK 响应
type Response struct {
	Event   string      `json:"event"`             // 事件类型（原事件 + ".ack" 后缀）
	Seq     int64       `json:"seq"`               // 对应请求的序号
	Code    int         `json:"code"`              // 状态码，0=成功
	Message string      `json:"message"`           // 状态描述
	Data    interface{} `json:"data,omitempty"`    // 响应数据
}

// PushMessage 服务端主动推送给客户端的消息
type PushMessage struct {
	Event string      `json:"event"`          // 事件类型
	Data  interface{} `json:"data,omitempty"` // 推送数据
	Time  string      `json:"time"`           // 推送时间
}

// NewPushMessage 创建一条推送消息
func NewPushMessage(event string, data interface{}) *PushMessage {
	return &PushMessage{
		Event: event,
		Data:  data,
		Time:  time.Now().Format("2006-01-02 15:04:05"),
	}
}

// NewResponse 创建一条 ACK 响应
func NewResponse(event string, seq int64, code int, message string, data interface{}) *Response {
	return &Response{
		Event:   event + ".ack",
		Seq:     seq,
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// MarshalPush 将推送消息序列化为 JSON 字节
func MarshalPush(msg *PushMessage) ([]byte, error) {
	return json.Marshal(msg)
}

// MarshalResponse 将响应消息序列化为 JSON 字节
func MarshalResponse(resp *Response) ([]byte, error) {
	return json.Marshal(resp)
}
