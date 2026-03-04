/**
 * 群聊状态 Store
 *
 * 管理群聊信息、群成员、入群申请，提供：
 * - 当前群详情和成员列表
 * - 群管理操作（邀请/踢人/禁言/设管理员/解散等）
 * - WebSocket 群管理事件监听（group.* 系列）
 *
 * 对应后端 API：/api/v1/groups/*
 * 对应 WS 事件：group.created / group.member.join / group.member.leave / group.member.kicked
 *               group.info.update / group.dissolved / group.mute.update / group.role.update
 *               group.owner.transfer / group.join.request / group.join.approved
 */

import { defineStore } from 'pinia'
import { ref } from 'vue'
import groupApi from '@/api/group'
import wsService from '@/services/websocket'
import { useChatStore } from '@/store/chat'

export const useGroupStore = defineStore('group', () => {
  /** 当前查看的群详情 */
  const currentGroup = ref(null)

  /** 当前群的成员列表 */
  const currentMembers = ref([])

  /** 当前群的入群申请列表 */
  const joinRequests = ref([])

  /** 群搜索结果 */
  const searchResults = ref([])
  const searchTotal = ref(0)

  // ==================== Actions：群信息 ====================

  /** 获取群详情 */
  const fetchGroupDetail = async (groupId) => {
    const res = await groupApi.getGroupDetail(groupId)
    if (res.data) {
      currentGroup.value = res.data
    }
    return currentGroup.value
  }

  /** 创建群聊 */
  const createGroup = async (name, memberIds, avatar = '') => {
    const res = await groupApi.createGroup(name, memberIds, avatar)
    return res.data
  }

  /** 更新群信息 */
  const updateGroup = async (groupId, data) => {
    await groupApi.updateGroup(groupId, data)
    if (currentGroup.value && currentGroup.value.id === groupId) {
      Object.assign(currentGroup.value, data)
    }
  }

  /** 解散群聊 */
  const dissolveGroup = async (groupId) => {
    await groupApi.dissolveGroup(groupId)
    currentGroup.value = null
  }

  // ==================== Actions：群成员管理 ====================

  /** 获取群成员列表 */
  const fetchMembers = async (groupId) => {
    const res = await groupApi.getMembers(groupId)
    if (res.data) {
      currentMembers.value = res.data.list || res.data || []
    }
    return currentMembers.value
  }

  /** 邀请入群 */
  const inviteMembers = async (groupId, userIds) => {
    await groupApi.inviteMembers(groupId, userIds)
  }

  /** 踢出群成员 */
  const kickMember = async (groupId, userId) => {
    await groupApi.kickMember(groupId, userId)
    currentMembers.value = currentMembers.value.filter(m => m.user_id !== userId)
  }

  /** 退出群聊 */
  const leaveGroup = async (groupId) => {
    await groupApi.leaveGroup(groupId)
    currentGroup.value = null
  }

  /** 转让群主 */
  const transferOwner = async (groupId, newOwnerId) => {
    await groupApi.transferOwner(groupId, newOwnerId)
  }

  /** 设置/取消管理员 */
  const setMemberRole = async (groupId, userId, role) => {
    await groupApi.setMemberRole(groupId, userId, role)
    const member = currentMembers.value.find(m => m.user_id === userId)
    if (member) member.role = role
  }

  /** 禁言/解除禁言 */
  const muteMember = async (groupId, userId, isMuted) => {
    await groupApi.muteMember(groupId, userId, isMuted)
    const member = currentMembers.value.find(m => m.user_id === userId)
    if (member) member.is_muted = isMuted
  }

  /** 全体禁言 */
  const setAllMuted = async (groupId, isMuted) => {
    await groupApi.setAllMuted(groupId, isMuted)
    if (currentGroup.value && currentGroup.value.id === groupId) {
      currentGroup.value.is_all_muted = isMuted
    }
  }

  /** 修改群昵称 */
  const updateNickname = async (groupId, nickname) => {
    await groupApi.updateNickname(groupId, nickname)
  }

  /** 设置/取消免打扰 */
  const setDoNotDisturb = async (conversationId, isDoNotDisturb) => {
    await groupApi.setDoNotDisturb(conversationId, isDoNotDisturb)
  }

  // ==================== Actions：入群申请 ====================

  /** 提交入群申请 */
  const submitJoinRequest = async (groupId, message = '') => {
    await groupApi.submitJoinRequest(groupId, message)
  }

  /** 获取入群申请列表 */
  const fetchJoinRequests = async (groupId) => {
    const res = await groupApi.getJoinRequests(groupId)
    if (res.data) {
      joinRequests.value = res.data.list || res.data || []
    }
    return joinRequests.value
  }

  /** 审批入群申请 */
  const reviewJoinRequest = async (groupId, requestId, action) => {
    await groupApi.reviewJoinRequest(groupId, requestId, action)
    joinRequests.value = joinRequests.value.filter(r => r.id !== requestId)
  }

  // ==================== Actions：搜索 ====================

  /**
   * 搜索公开群
   * @param {string} keyword - 搜索关键词
   * @param {number} page - 页码
   * @param {number} pageSize - 每页条数
   * @param {boolean} append - 是否追加到现有列表（分页加载更多时为 true）
   */
  const searchGroups = async (keyword, page = 1, pageSize = 20, append = false) => {
    const res = await groupApi.searchGroups(keyword, page, pageSize)
    if (res.data) {
      const newList = res.data.list || []
      searchResults.value = append ? [...searchResults.value, ...newList] : newList
      searchTotal.value = res.data.total || 0
    }
    return { list: searchResults.value, total: searchTotal.value }
  }

  // ==================== WS 事件监听 ====================

  let _wsInitialized = false

  /** 初始化 WebSocket 群管理事件监听（幂等） */
  const initWsListeners = () => {
    if (_wsInitialized) return
    _wsInitialized = true

    wsService.on('group.created', _onGroupCreated)
    wsService.on('group.member.join', _onMemberJoin)
    wsService.on('group.member.leave', _onMemberLeave)
    wsService.on('group.member.kicked', _onMemberKicked)
    wsService.on('group.info.update', _onInfoUpdate)
    wsService.on('group.dissolved', _onDissolved)
    wsService.on('group.mute.update', _onMuteUpdate)
    wsService.on('group.role.update', _onRoleUpdate)
    wsService.on('group.owner.transfer', _onOwnerTransfer)
    wsService.on('group.join.request', _onJoinRequest)
    wsService.on('group.join.approved', _onJoinApproved)
  }

  /** 群聊创建通知 → 刷新会话列表 */
  const _onGroupCreated = () => {
    const chatStore = useChatStore()
    chatStore.fetchConversations()
  }

  /** 新成员加入 → 刷新成员和会话 */
  const _onMemberJoin = (msg) => {
    if (!msg || !msg.data) return
    const { group_id } = msg.data
    if (currentGroup.value && currentGroup.value.id === group_id) {
      fetchMembers(group_id)
    }
    const chatStore = useChatStore()
    chatStore.fetchConversations()
  }

  /** 成员退出 → 刷新成员 */
  const _onMemberLeave = (msg) => {
    if (!msg || !msg.data) return
    const { group_id, user_id } = msg.data
    if (currentGroup.value && currentGroup.value.id === group_id) {
      currentMembers.value = currentMembers.value.filter(m => m.user_id !== user_id)
    }
  }

  /** 成员被踢 → 刷新或跳转 */
  const _onMemberKicked = (msg) => {
    if (!msg || !msg.data) return
    const { group_id, user_id } = msg.data
    if (currentGroup.value && currentGroup.value.id === group_id) {
      currentMembers.value = currentMembers.value.filter(m => m.user_id !== user_id)
    }
    const chatStore = useChatStore()
    chatStore.fetchConversations()
  }

  /** 群信息变更 → 更新当前群详情 + 刷新会话 */
  const _onInfoUpdate = (msg) => {
    if (!msg || !msg.data) return
    const { group_id, updates } = msg.data
    if (currentGroup.value && currentGroup.value.id === group_id && updates) {
      Object.assign(currentGroup.value, updates)
    }
    const chatStore = useChatStore()
    chatStore.fetchConversations()
  }

  /** 群解散 → 清空当前群 + 刷新会话 */
  const _onDissolved = (msg) => {
    if (!msg || !msg.data) return
    const { group_id } = msg.data
    if (currentGroup.value && currentGroup.value.id === group_id) {
      currentGroup.value = null
      uni.showToast({ title: '该群已被解散', icon: 'none' })
    }
    const chatStore = useChatStore()
    chatStore.fetchConversations()
  }

  /** 禁言变更 → 更新成员 / 群信息 */
  const _onMuteUpdate = (msg) => {
    if (!msg || !msg.data) return
    const { group_id, user_id, is_muted, is_all_muted } = msg.data
    if (currentGroup.value && currentGroup.value.id === group_id) {
      if (is_all_muted !== undefined) {
        currentGroup.value.is_all_muted = is_all_muted
      }
      if (user_id) {
        const member = currentMembers.value.find(m => m.user_id === user_id)
        if (member) member.is_muted = is_muted
      }
    }
  }

  /** 角色变更 → 更新成员角色 */
  const _onRoleUpdate = (msg) => {
    if (!msg || !msg.data) return
    const { group_id, user_id, role } = msg.data
    if (currentGroup.value && currentGroup.value.id === group_id) {
      const member = currentMembers.value.find(m => m.user_id === user_id)
      if (member) member.role = role
    }
  }

  /** 群主转让 → 刷新群详情和成员 */
  const _onOwnerTransfer = (msg) => {
    if (!msg || !msg.data) return
    const { group_id } = msg.data
    if (currentGroup.value && currentGroup.value.id === group_id) {
      fetchGroupDetail(group_id)
      fetchMembers(group_id)
    }
  }

  /** 新入群申请 → 提示通知 */
  const _onJoinRequest = (msg) => {
    if (!msg || !msg.data) return
    uni.showToast({ title: '收到新的入群申请', icon: 'none' })
  }

  /** 入群申请通过 → 刷新会话列表 */
  const _onJoinApproved = () => {
    const chatStore = useChatStore()
    chatStore.fetchConversations()
    uni.showToast({ title: '入群申请已通过', icon: 'none' })
  }

  return {
    currentGroup,
    currentMembers,
    joinRequests,
    searchResults,
    searchTotal,
    fetchGroupDetail,
    createGroup,
    updateGroup,
    dissolveGroup,
    fetchMembers,
    inviteMembers,
    kickMember,
    leaveGroup,
    transferOwner,
    setMemberRole,
    muteMember,
    setAllMuted,
    updateNickname,
    setDoNotDisturb,
    submitJoinRequest,
    fetchJoinRequests,
    reviewJoinRequest,
    searchGroups,
    initWsListeners
  }
})
