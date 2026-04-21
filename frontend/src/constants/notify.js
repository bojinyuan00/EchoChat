/**
 * 通知模块前端常量
 *
 * 与后端 backend/go-service/app/constants/notify.go 一致
 * 新增/修改通知类型时必须同步维护两端常量
 */

// 通知类型（与后端 NotifyType* 保持一致）
export const NOTIFY_TYPE_FRIEND_REQUEST = 'friend_request'
export const NOTIFY_TYPE_FRIEND_ACCEPTED = 'friend_accepted'
export const NOTIFY_TYPE_FRIEND_REJECTED = 'friend_rejected'

export const NOTIFY_TYPE_GROUP_INVITE = 'group_invite'
export const NOTIFY_TYPE_GROUP_JOIN_REQUEST = 'group_join_request'
export const NOTIFY_TYPE_GROUP_JOIN_APPROVED = 'group_join_approved'
export const NOTIFY_TYPE_GROUP_JOIN_REJECTED = 'group_join_rejected'
export const NOTIFY_TYPE_GROUP_KICKED = 'group_kicked'
export const NOTIFY_TYPE_GROUP_ROLE_CHANGED = 'group_role_changed'

export const NOTIFY_TYPE_SYSTEM_BROADCAST = 'system_broadcast'

export const NOTIFY_TYPE_MEETING_INVITE = 'meeting_invite'
export const NOTIFY_TYPE_MEETING_REMINDER = 'meeting_reminder'

// 通知分类（前端 5 Tab）
export const NOTIFY_CATEGORY_ALL = 'all'
export const NOTIFY_CATEGORY_FRIEND = 'friend'
export const NOTIFY_CATEGORY_GROUP = 'group'
export const NOTIFY_CATEGORY_MEETING = 'meeting'
export const NOTIFY_CATEGORY_SYSTEM = 'system'

/** Tab 列表（顺序即 UI 展示顺序） */
export const NOTIFY_CATEGORY_TABS = [
  { key: NOTIFY_CATEGORY_ALL, label: '全部' },
  { key: NOTIFY_CATEGORY_FRIEND, label: '好友' },
  { key: NOTIFY_CATEGORY_GROUP, label: '群聊' },
  { key: NOTIFY_CATEGORY_MEETING, label: '会议' },
  { key: NOTIFY_CATEGORY_SYSTEM, label: '系统' }
]

/** 通知类型中文映射（默认标题兜底） */
export const NOTIFY_TYPE_LABEL = {
  [NOTIFY_TYPE_FRIEND_REQUEST]: '好友申请',
  [NOTIFY_TYPE_FRIEND_ACCEPTED]: '好友申请通过',
  [NOTIFY_TYPE_FRIEND_REJECTED]: '好友申请被拒',
  [NOTIFY_TYPE_GROUP_INVITE]: '群聊邀请',
  [NOTIFY_TYPE_GROUP_JOIN_REQUEST]: '入群申请',
  [NOTIFY_TYPE_GROUP_JOIN_APPROVED]: '入群申请通过',
  [NOTIFY_TYPE_GROUP_JOIN_REJECTED]: '入群申请被拒',
  [NOTIFY_TYPE_GROUP_KICKED]: '移出群聊',
  [NOTIFY_TYPE_GROUP_ROLE_CHANGED]: '群角色变更',
  [NOTIFY_TYPE_SYSTEM_BROADCAST]: '系统通知',
  [NOTIFY_TYPE_MEETING_INVITE]: '会议邀请',
  [NOTIFY_TYPE_MEETING_REMINDER]: '会议提醒'
}

/** 通知图标（emoji 占位；后续可替换为 SVG） */
export const NOTIFY_TYPE_ICON = {
  [NOTIFY_TYPE_FRIEND_REQUEST]: '\uD83D\uDC65', // 👥
  [NOTIFY_TYPE_FRIEND_ACCEPTED]: '\u2705',       // ✅
  [NOTIFY_TYPE_FRIEND_REJECTED]: '\u274C',       // ❌
  [NOTIFY_TYPE_GROUP_INVITE]: '\uD83C\uDF89',    // 🎉
  [NOTIFY_TYPE_GROUP_JOIN_REQUEST]: '\uD83D\uDCE5', // 📥
  [NOTIFY_TYPE_GROUP_JOIN_APPROVED]: '\u2705',      // ✅
  [NOTIFY_TYPE_GROUP_JOIN_REJECTED]: '\u274C',      // ❌
  [NOTIFY_TYPE_GROUP_KICKED]: '\uD83D\uDEAA',       // 🚪
  [NOTIFY_TYPE_GROUP_ROLE_CHANGED]: '\uD83D\uDD11', // 🔑
  [NOTIFY_TYPE_SYSTEM_BROADCAST]: '\uD83D\uDCE2',   // 📢
  [NOTIFY_TYPE_MEETING_INVITE]: '\uD83D\uDCC5',     // 📅
  [NOTIFY_TYPE_MEETING_REMINDER]: '\u23F0'          // ⏰
}

/** 默认图标 */
export const NOTIFY_DEFAULT_ICON = '\uD83D\uDD14' // 🔔

/** 图标背景颜色（按分类） */
export const NOTIFY_CATEGORY_COLOR = {
  [NOTIFY_CATEGORY_FRIEND]: '#DBEAFE',  // 蓝浅
  [NOTIFY_CATEGORY_GROUP]: '#DCFCE7',   // 绿浅
  [NOTIFY_CATEGORY_MEETING]: '#FEF3C7', // 黄浅
  [NOTIFY_CATEGORY_SYSTEM]: '#E0E7FF'   // 紫浅
}

/**
 * 判断该通知是否支持内联操作（接受/拒绝按钮）
 * 仅 group_invite 和 group_join_request 支持，对方待处理状态
 * @param {string} type
 * @returns {boolean}
 */
export const supportsInlineAction = (type) => {
  return type === NOTIFY_TYPE_GROUP_INVITE || type === NOTIFY_TYPE_GROUP_JOIN_REQUEST
}
