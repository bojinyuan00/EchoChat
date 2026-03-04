package constants

// 群聊状态
const (
	GroupStatusNormal    = 1 // 正常
	GroupStatusDissolved = 2 // 已解散
)

// GroupStatusMap 群聊状态中文映射
var GroupStatusMap = map[int]string{
	GroupStatusNormal:    "正常",
	GroupStatusDissolved: "已解散",
}

// 群成员角色（im_conversation_members.role）
const (
	GroupRoleNormal = 0 // 普通成员
	GroupRoleAdmin  = 1 // 管理员
	GroupRoleOwner  = 2 // 群主
)

// GroupRoleMap 群成员角色中文映射
var GroupRoleMap = map[int]string{
	GroupRoleNormal: "普通成员",
	GroupRoleAdmin:  "管理员",
	GroupRoleOwner:  "群主",
}

// 入群申请状态（im_group_join_requests.status）
const (
	JoinRequestStatusPending  = 0 // 待审批
	JoinRequestStatusApproved = 1 // 通过
	JoinRequestStatusRejected = 2 // 拒绝
)

// JoinRequestStatusMap 入群申请状态中文映射
var JoinRequestStatusMap = map[int]string{
	JoinRequestStatusPending:  "待审批",
	JoinRequestStatusApproved: "通过",
	JoinRequestStatusRejected: "拒绝",
}

// 群聊默认配置
const (
	GroupDefaultMaxMembers = 200 // 默认最大成员数
)
