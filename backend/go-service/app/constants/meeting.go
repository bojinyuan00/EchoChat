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

// 会议 WS 事件常量（设计文档 §6.3）
// 命名格式：meeting.{domain}.{action}
// 方向说明：C→S（客户端发往服务端，需 ACK）；S→C（服务端广播/定向推送，不需 ACK）
const (
	// ====== 房间组（3 个）======
	MeetingWSEventRoomJoin  = "meeting.room.join"  // C→S：加入会议后的"宣告在线"，绑定 WS 连接 ↔ roomCode
	MeetingWSEventRoomLeave = "meeting.room.leave" // C→S：主动离会（等价 REST leave，但保留信令触发入口）
	MeetingWSEventRoomEnded = "meeting.room.ended" // S→C：会议已结束（广播全员）

	// ====== 成员组（5 个）======
	MeetingWSEventMemberJoined      = "meeting.member.joined"        // S→C：新成员加入
	MeetingWSEventMemberLeft        = "meeting.member.left"          // S→C：成员离开
	MeetingWSEventMemberStateChange = "meeting.member.state.changed" // 双向：麦克风 / 摄像头状态变化（host 可带 target_user_id 强制静音他人）
	MeetingWSEventMemberKicked      = "meeting.member.kicked"        // S→C：被移出会议（定向发给被踢者）
	MeetingWSEventMemberProducerNew = "meeting.member.producer.new"  // S→C：有新 producer 可订阅（produce.start 成功后广播）
	MeetingWSEventHostChanged       = "meeting.host.changed"         // S→C：主持人变更

	// ====== 媒体组（5 个，mediasoup signaling 桥接）======
	MeetingWSEventTransportCreate  = "meeting.transport.create"  // C→S→Node：创建 Transport
	MeetingWSEventTransportConnect = "meeting.transport.connect" // C→S→Node：Transport DTLS 握手
	MeetingWSEventProduceStart     = "meeting.produce.start"     // C→S→Node：创建 Producer（广播 producer.new）
	MeetingWSEventConsumeStart     = "meeting.consume.start"     // C→S→Node：订阅远端 Producer，创建本地 Consumer
	MeetingWSEventProducerClose    = "meeting.producer.close"    // C→S→Node：关闭自己的 Producer（推流停止）

	// ====== 补充事件（非 §6.3 核心 11 事件，但业务必须）======
	MeetingWSEventChatMessage = "meeting.chat.message" // S→C：会议内文字聊天广播（REST SendChat 触发）
)

// MeetingWSClientEvents 客户端可主动发起的 WS 事件（C→S）白名单
// 用于 WS 消息路由时快速校验事件名合法性
var MeetingWSClientEvents = []string{
	MeetingWSEventRoomJoin,
	MeetingWSEventRoomLeave,
	MeetingWSEventMemberStateChange,
	MeetingWSEventTransportCreate,
	MeetingWSEventTransportConnect,
	MeetingWSEventProduceStart,
	MeetingWSEventConsumeStart,
	MeetingWSEventProducerClose,
}
