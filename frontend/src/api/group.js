/**
 * 群聊模块 API
 *
 * 对应后端路由：/api/v1/groups/*
 * 群消息发送/撤回通过 WebSocket 进行，此处仅包含 REST API
 */

import { get, post, put, del } from '@/utils/request'

/** 创建群聊 */
const createGroup = (name, memberIds, avatar = '') => {
  return post('/api/v1/groups', { name, member_ids: memberIds, avatar })
}

/** 获取群详情 */
const getGroupDetail = (groupId) => {
  return get(`/api/v1/groups/${groupId}`)
}

/** 更新群信息（名称/头像/公告/可搜索性） */
const updateGroup = (groupId, data) => {
  return put(`/api/v1/groups/${groupId}`, data)
}

/** 解散群聊 */
const dissolveGroup = (groupId) => {
  return del(`/api/v1/groups/${groupId}`)
}

/** 获取群成员列表 */
const getMembers = (groupId) => {
  return get(`/api/v1/groups/${groupId}/members`)
}

/** 邀请用户入群 */
const inviteMembers = (groupId, userIds) => {
  return post(`/api/v1/groups/${groupId}/members`, { user_ids: userIds })
}

/** 踢出群成员 */
const kickMember = (groupId, userId) => {
  return del(`/api/v1/groups/${groupId}/members/${userId}`)
}

/** 退出群聊 */
const leaveGroup = (groupId) => {
  return del(`/api/v1/groups/${groupId}/members/me`)
}

/** 转让群主 */
const transferOwner = (groupId, newOwnerId) => {
  return put(`/api/v1/groups/${groupId}/transfer`, { new_owner_id: newOwnerId })
}

/** 设置/取消管理员 */
const setMemberRole = (groupId, userId, role) => {
  return put(`/api/v1/groups/${groupId}/members/${userId}/role`, { role })
}

/** 禁言/解除禁言成员 */
const muteMember = (groupId, userId, isMuted) => {
  return put(`/api/v1/groups/${groupId}/members/${userId}/mute`, { is_muted: isMuted })
}

/** 设置/取消全体禁言 */
const setAllMuted = (groupId, isMuted) => {
  return put(`/api/v1/groups/${groupId}/mute`, { is_muted: isMuted })
}

/** 修改群内昵称 */
const updateNickname = (groupId, nickname) => {
  return put(`/api/v1/groups/${groupId}/members/me/nickname`, { nickname })
}

/** 提交入群申请 */
const submitJoinRequest = (groupId, message = '') => {
  return post(`/api/v1/groups/${groupId}/join-requests`, { message })
}

/** 获取入群申请列表（群主/管理员） */
const getJoinRequests = (groupId) => {
  return get(`/api/v1/groups/${groupId}/join-requests`)
}

/** 审批入群申请 */
const reviewJoinRequest = (groupId, requestId, action) => {
  return put(`/api/v1/groups/${groupId}/join-requests/${requestId}`, { action })
}

/** 搜索公开群 */
const searchGroups = (keyword, page = 1, pageSize = 20) => {
  return get('/api/v1/groups/search', { keyword, page, page_size: pageSize })
}

/** 设置/取消免打扰 */
const setDoNotDisturb = (conversationId, isDoNotDisturb) => {
  return put(`/api/v1/im/conversations/${conversationId}/dnd`, { is_do_not_disturb: isDoNotDisturb })
}

export default {
  createGroup,
  getGroupDetail,
  updateGroup,
  dissolveGroup,
  getMembers,
  inviteMembers,
  kickMember,
  leaveGroup,
  transferOwner,
  setMemberRole,
  muteMember,
  setAllMuted,
  updateNickname,
  submitJoinRequest,
  getJoinRequests,
  reviewJoinRequest,
  searchGroups,
  setDoNotDisturb
}
