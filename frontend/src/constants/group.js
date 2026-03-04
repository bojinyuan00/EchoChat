/**
 * 群聊相关常量定义
 *
 * 与后端 constants/group.go 保持一致：
 * - GroupRoleNormal = 0（普通成员）
 * - GroupRoleAdmin  = 1（管理员）
 * - GroupRoleOwner  = 2（群主）
 */

/** 群成员角色（im_conversation_members.role） */
export const GROUP_ROLE = {
  MEMBER: 0,
  ADMIN: 1,
  OWNER: 2
}

/** 群成员角色中文映射 */
export const GROUP_ROLE_LABEL = {
  [GROUP_ROLE.MEMBER]: '普通成员',
  [GROUP_ROLE.ADMIN]: '管理员',
  [GROUP_ROLE.OWNER]: '群主'
}

/** 群聊状态 */
export const GROUP_STATUS = {
  NORMAL: 1,
  DISSOLVED: 2
}

/** 入群申请状态 */
export const JOIN_REQUEST_STATUS = {
  PENDING: 0,
  APPROVED: 1,
  REJECTED: 2
}
