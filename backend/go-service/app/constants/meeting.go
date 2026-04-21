package constants

// 会议相关常量（Phase 2e-2）
// 包含：会议类型、会议状态、参与者角色、结束原因、离会原因、默认配置
// 所有枚举值严禁直接用中文字符串，中文仅用于 *Map 展示

// 会议类型（meeting_rooms.type）
const (
	MeetingTypeInstant   = 1 // 即时会议（MVP 仅此）
	MeetingTypeScheduled = 2 // 预约会议（Phase 2e-3）
)

// MeetingTypeMap 会议类型中文映射
var MeetingTypeMap = map[int]string{
	MeetingTypeInstant:   "即时会议",
	MeetingTypeScheduled: "预约会议",
}

// 会议状态（meeting_rooms.status）
const (
	MeetingStatusPending = 0 // 未开始（仅预约会议使用）
	MeetingStatusActive  = 1 // 进行中
	MeetingStatusEnded   = 2 // 已结束
)

// MeetingStatusMap 会议状态中文映射
var MeetingStatusMap = map[int]string{
	MeetingStatusPending: "未开始",
	MeetingStatusActive:  "进行中",
	MeetingStatusEnded:   "已结束",
}

// 会议参与者角色（meeting_participants.role）
// MVP 仅使用 Participant / Host 两档，CoHost 保留至第二期
const (
	MeetingRoleParticipant = 0 // 普通参会人
	MeetingRoleHost        = 1 // 主持人
	MeetingRoleCoHost      = 2 // 联合主持人（保留，MVP 不用）
)

// MeetingRoleMap 参与者角色中文映射
var MeetingRoleMap = map[int]string{
	MeetingRoleParticipant: "参会人",
	MeetingRoleHost:        "主持人",
	MeetingRoleCoHost:      "联合主持人",
}

// 会议结束原因（meeting_rooms.ended_reason）
const (
	MeetingEndedReasonHostEnded   = "host_ended"   // 主持人主动结束
	MeetingEndedReasonEmptyTTL    = "empty_ttl"    // 空房超时销毁
	MeetingEndedReasonAdminForce  = "admin_force"  // 管理员强制结束
	MeetingEndedReasonSystemError = "system_error" // 系统异常（如 Node worker died）
)

// MeetingEndedReasonMap 结束原因中文映射
var MeetingEndedReasonMap = map[string]string{
	MeetingEndedReasonHostEnded:   "主持人结束",
	MeetingEndedReasonEmptyTTL:    "空房超时",
	MeetingEndedReasonAdminForce:  "管理员强制结束",
	MeetingEndedReasonSystemError: "系统异常",
}

// 离会原因（meeting_participants.left_reason）
const (
	MeetingLeftReasonSelf       = "self"       // 主动离会
	MeetingLeftReasonKicked     = "kicked"     // 被主持人移除
	MeetingLeftReasonHostEnd    = "host_end"   // 主持人结束会议
	MeetingLeftReasonEmptyTTL   = "empty_ttl"  // 空房 TTL 触发
	MeetingLeftReasonDisconnect = "disconnect" // 网络断开（宽限期结束仍未重连）
)

// MeetingLeftReasonMap 离会原因中文映射
var MeetingLeftReasonMap = map[string]string{
	MeetingLeftReasonSelf:       "主动离开",
	MeetingLeftReasonKicked:     "被移除",
	MeetingLeftReasonHostEnd:    "会议结束",
	MeetingLeftReasonEmptyTTL:   "空房超时",
	MeetingLeftReasonDisconnect: "网络断开",
}

// 会议默认配置（应用层强制约束）
const (
	MeetingMVPMaxMembers       = 8   // MVP 阶段单会议硬上限
	MeetingRoomCodeLength      = 9   // 会议号去连字符后的长度（XXX-XXX-XXX）
	MeetingRoomCodeRetryMax    = 3   // 生成会议号冲突重试上限
	MeetingPasswordMaxAttempts = 5   // 密码连续错误锁定阈值
	MeetingPasswordLockSeconds = 600 // 密码锁定时长（10 分钟）
	MeetingHostGraceSeconds    = 120 // 主持人掉线宽限期（2 分钟）
	MeetingEmptyRoomTTLSeconds = 300 // 空房销毁 TTL（5 分钟）
	MeetingInviteTokenTTL      = 600 // 邀请链接 Token TTL（10 分钟）
	MeetingChatRetentionHours  = 24  // 会议聊天保留时长（会议结束后）
)

// 会议 WS 事件常量（设计文档 §6.3，11 个事件）
// 命名格式：meeting.{domain}.{action}
const (
	// 房间级事件
	MeetingWSEventRoomEnded      = "meeting.room.ended"      // 会议已结束
	MeetingWSEventRoomLocked     = "meeting.room.locked"     // 会议被锁定（等候室，Phase 2e-3）
	MeetingWSEventRoomHostChange = "meeting.room.host.changed" // 主持人变更

	// 成员级事件
	MeetingWSEventMemberJoined      = "meeting.member.joined"       // 新成员加入
	MeetingWSEventMemberLeft        = "meeting.member.left"         // 成员离开
	MeetingWSEventMemberStateChange = "meeting.member.state.changed" // 麦克风 / 摄像头状态变化
	MeetingWSEventMemberKicked      = "meeting.member.kicked"       // 被移出

	// 媒体级事件（Go 发起，驱动客户端 mediasoup-client 订阅/取消）
	MeetingWSEventProducerNew       = "meeting.producer.new"    // 有新 producer 可订阅
	MeetingWSEventProducerClosed    = "meeting.producer.closed" // producer 已关闭
	MeetingWSEventConsumerResumed   = "meeting.consumer.resumed" // consumer 已恢复（由服务端触发）
	MeetingWSEventChatMessage       = "meeting.chat.message"    // 会议内文字聊天
)
