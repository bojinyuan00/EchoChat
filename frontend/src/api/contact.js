/**
 * 联系人模块 API
 *
 * 对应后端路由：/api/v1/contacts/*
 * 接口文档见：docs/api/frontend/contact.md
 */

import { get, post, put, del } from '@/utils/request'

/** 获取好友列表 */
const getFriendList = (groupId) => {
  const params = groupId ? { group_id: groupId } : {}
  return get('/api/v1/contacts', params)
}

/** 发送好友申请 */
const sendFriendRequest = (targetId, message = '') => {
  return post('/api/v1/contacts/request', { target_id: targetId, message })
}

/** 接受好友申请 */
const acceptRequest = (requestId) => {
  return post('/api/v1/contacts/accept', { request_id: requestId })
}

/** 拒绝好友申请 */
const rejectRequest = (requestId) => {
  return post('/api/v1/contacts/reject', { request_id: requestId })
}

/** 获取待处理的好友申请 */
const getPendingRequests = () => {
  return get('/api/v1/contacts/requests')
}

/** 删除好友 */
const deleteFriend = (friendId) => {
  return del(`/api/v1/contacts/${friendId}`)
}

/** 更新好友备注 */
const updateRemark = (friendId, remark) => {
  return put(`/api/v1/contacts/${friendId}/remark`, { remark })
}

/** 拉黑用户 */
const blockUser = (targetId) => {
  return post('/api/v1/contacts/block', { target_id: targetId })
}

/** 取消拉黑 */
const unblockUser = (userId) => {
  return del(`/api/v1/contacts/block/${userId}`)
}

/** 获取黑名单 */
const getBlockList = () => {
  return get('/api/v1/contacts/block')
}

/** 获取分组列表 */
const getGroups = () => {
  return get('/api/v1/contacts/groups')
}

/** 创建分组 */
const createGroup = (name) => {
  return post('/api/v1/contacts/groups', { name })
}

/** 修改分组 */
const updateGroup = (groupId, name, sortOrder) => {
  const data = { name }
  if (sortOrder !== undefined) data.sort_order = sortOrder
  return put(`/api/v1/contacts/groups/${groupId}`, data)
}

/** 删除分组 */
const deleteGroup = (groupId) => {
  return del(`/api/v1/contacts/groups/${groupId}`)
}

/** 移动好友到分组 */
const moveToGroup = (friendId, groupId) => {
  return put(`/api/v1/contacts/${friendId}/group`, { group_id: groupId })
}

/** 搜索用户 */
const searchUsers = (keyword, page = 1, pageSize = 20) => {
  return get('/api/v1/users/search', { keyword, page, page_size: pageSize })
}

/** 好友推荐 */
const getRecommendFriends = () => {
  return get('/api/v1/contacts/recommend')
}

export default {
  getFriendList,
  sendFriendRequest,
  acceptRequest,
  rejectRequest,
  getPendingRequests,
  deleteFriend,
  updateRemark,
  blockUser,
  unblockUser,
  getBlockList,
  getGroups,
  createGroup,
  updateGroup,
  deleteGroup,
  moveToGroup,
  searchUsers,
  getRecommendFriends
}
