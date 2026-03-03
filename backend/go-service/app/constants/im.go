package constants

// 会话类型
const (
	ConversationTypePrivate = 1 // 单聊
	ConversationTypeGroup   = 2 // 群聊（Phase 2c 预留）
)

// ConversationTypeMap 会话类型中文映射
var ConversationTypeMap = map[int]string{
	ConversationTypePrivate: "单聊",
	ConversationTypeGroup:   "群聊",
}

// 消息类型
const (
	MessageTypeText  = 1 // 文本消息
	MessageTypeImage = 2 // 图片消息（预留）
	MessageTypeVoice = 3 // 语音消息（预留）
	MessageTypeVideo = 4 // 视频消息（预留）
	MessageTypeFile  = 5 // 文件消息（预留）
)

// MessageTypeMap 消息类型中文映射
var MessageTypeMap = map[int]string{
	MessageTypeText:  "文本",
	MessageTypeImage: "图片",
	MessageTypeVoice: "语音",
	MessageTypeVideo: "视频",
	MessageTypeFile:  "文件",
}

// 消息状态
const (
	MessageStatusNormal   = 1 // 正常
	MessageStatusRecalled = 2 // 已撤回
	MessageStatusDeleted  = 3 // 已删除
)

// MessageStatusMap 消息状态中文映射
var MessageStatusMap = map[int]string{
	MessageStatusNormal:   "正常",
	MessageStatusRecalled: "已撤回",
	MessageStatusDeleted:  "已删除",
}

// MessageRecallTimeLimit 消息撤回时限（秒），超过此时间不允许撤回
const MessageRecallTimeLimit = 2 * 60
