package constants

// 通知类型枚举（notify_notifications.type）
// 覆盖好友 / 群聊 / 会议 / 系统四大类，其中 meeting_* 在 Phase 2e-1 仅预留枚举，由 Phase 2e-2/2e-3 业务接入
const (
	// 好友相关（contact 模块触发）
	NotifyTypeFriendRequest  = "friend_request"  // 收到好友申请
	NotifyTypeFriendAccepted = "friend_accepted" // 好友申请被接受（通知申请人）
	NotifyTypeFriendRejected = "friend_rejected" // 好友申请被拒绝（通知申请人）

	// 群聊相关（group 模块触发）
	NotifyTypeGroupInvite        = "group_invite"         // 被邀请入群
	NotifyTypeGroupJoinRequest   = "group_join_request"   // 有新入群申请（通知群管理员）
	NotifyTypeGroupJoinApproved  = "group_join_approved"  // 入群申请通过（通知申请人）
	NotifyTypeGroupJoinRejected  = "group_join_rejected"  // 入群申请被拒（通知申请人）
	NotifyTypeGroupKicked        = "group_kicked"         // 被移出群聊
	NotifyTypeGroupRoleChanged   = "group_role_changed"   // 群内角色变更（被设/取消管理员）

	// 系统广播（admin 触发）
	NotifyTypeSystemBroadcast = "system_broadcast" // 管理员全员广播

	// 会议相关（Phase 2e-1 预留枚举，Phase 2e-2/2e-3 业务接入）
	NotifyTypeMeetingInvite   = "meeting_invite"   // 会议邀请
	NotifyTypeMeetingReminder = "meeting_reminder" // 会议提醒
)

// NotifyTypeMap 通知类型中文映射
// 用于日志、管理端展示、默认标题兜底
var NotifyTypeMap = map[string]string{
	NotifyTypeFriendRequest:     "好友申请",
	NotifyTypeFriendAccepted:    "好友申请通过",
	NotifyTypeFriendRejected:    "好友申请被拒",
	NotifyTypeGroupInvite:       "群聊邀请",
	NotifyTypeGroupJoinRequest:  "入群申请",
	NotifyTypeGroupJoinApproved: "入群申请通过",
	NotifyTypeGroupJoinRejected: "入群申请被拒",
	NotifyTypeGroupKicked:       "移出群聊",
	NotifyTypeGroupRoleChanged:  "群角色变更",
	NotifyTypeSystemBroadcast:   "系统通知",
	NotifyTypeMeetingInvite:     "会议邀请",
	NotifyTypeMeetingReminder:   "会议提醒",
}

// 通知分类（用于前端 5 个 Tab 过滤）
const (
	NotifyCategoryAll     = "all"     // 全部
	NotifyCategoryFriend  = "friend"  // 好友
	NotifyCategoryGroup   = "group"   // 群聊
	NotifyCategoryMeeting = "meeting" // 会议
	NotifyCategorySystem  = "system"  // 系统
)

// NotifyCategoryOfType 根据 type 判断所属分类，用于列表按分类筛选
func NotifyCategoryOfType(notifyType string) string {
	switch notifyType {
	case NotifyTypeFriendRequest, NotifyTypeFriendAccepted, NotifyTypeFriendRejected:
		return NotifyCategoryFriend
	case NotifyTypeGroupInvite, NotifyTypeGroupJoinRequest, NotifyTypeGroupJoinApproved,
		NotifyTypeGroupJoinRejected, NotifyTypeGroupKicked, NotifyTypeGroupRoleChanged:
		return NotifyCategoryGroup
	case NotifyTypeMeetingInvite, NotifyTypeMeetingReminder:
		return NotifyCategoryMeeting
	case NotifyTypeSystemBroadcast:
		return NotifyCategorySystem
	default:
		return NotifyCategorySystem
	}
}

// NotifyTypesByCategory 分类 → type 列表映射，查询时用于构造 SQL IN 条件
var NotifyTypesByCategory = map[string][]string{
	NotifyCategoryFriend: {
		NotifyTypeFriendRequest, NotifyTypeFriendAccepted, NotifyTypeFriendRejected,
	},
	NotifyCategoryGroup: {
		NotifyTypeGroupInvite, NotifyTypeGroupJoinRequest, NotifyTypeGroupJoinApproved,
		NotifyTypeGroupJoinRejected, NotifyTypeGroupKicked, NotifyTypeGroupRoleChanged,
	},
	NotifyCategoryMeeting: {
		NotifyTypeMeetingInvite, NotifyTypeMeetingReminder,
	},
	NotifyCategorySystem: {
		NotifyTypeSystemBroadcast,
	},
}

// 通知业务对象类型（notify_notifications.target_type）
// 用于前端 deep-link 跳转路由
const (
	NotifyTargetUser    = "user"    // 跳转到用户主页/好友申请页
	NotifyTargetGroup   = "group"   // 跳转到群聊详情
	NotifyTargetMeeting = "meeting" // 跳转到会议（Phase 2e-2/2e-3）
	NotifyTargetSystem  = "system"  // 系统公告，无跳转
)

// 通知清理策略
const (
	NotifyRetentionDays = 30 // 已读通知保留天数（未读不清理）
)

// WS 事件常量
const (
	WSEventNotifyNew         = "notify.new"          // 新通知推送（S→C）
	WSEventNotifyUnreadTotal = "notify.unread.total" // 断线重连补偿未读数（S→C）
)
