/**
 * 联系人状态 Store
 *
 * 管理好友列表、好友申请、分组、黑名单和在线状态，提供：
 * - 好友列表增删改查
 * - 好友申请发送/接受/拒绝
 * - 好友分组管理
 * - 在线状态跟踪（结合 WebSocket 推送）
 *
 * 对应后端 API：/api/v1/contacts/*
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import contactApi from '@/api/contact'
import wsService from '@/services/websocket'

export const useContactStore = defineStore('contact', () => {
  /** 好友列表 */
  const friendList = ref([])

  /** 待处理的好友申请列表 */
  const pendingRequests = ref([])

  /** 好友分组列表 */
  const groups = ref([])

  /** 黑名单列表 */
  const blockList = ref([])

  /** 在线状态映射 userID -> boolean */
  const onlineMap = ref({})

  /** 未读好友申请数 */
  const pendingCount = computed(() => pendingRequests.value.length)

  // ==================== Actions ====================

  /** 获取好友列表 */
  const fetchFriends = async (groupId) => {
    const res = await contactApi.getFriendList(groupId)
    friendList.value = res.data || []
    return friendList.value
  }

  /** 获取待处理的好友申请 */
  const fetchPendingRequests = async () => {
    const res = await contactApi.getPendingRequests()
    pendingRequests.value = res.data || []
    return pendingRequests.value
  }

  /** 发送好友申请 */
  const sendRequest = async (targetId, message) => {
    await contactApi.sendFriendRequest(targetId, message)
  }

  /** 接受好友申请 */
  const acceptRequest = async (requestId) => {
    await contactApi.acceptRequest(requestId)
    pendingRequests.value = pendingRequests.value.filter(r => r.id !== requestId)
    await fetchFriends()
  }

  /** 拒绝好友申请 */
  const rejectRequest = async (requestId) => {
    await contactApi.rejectRequest(requestId)
    pendingRequests.value = pendingRequests.value.filter(r => r.id !== requestId)
  }

  /** 删除好友 */
  const deleteFriend = async (friendId) => {
    await contactApi.deleteFriend(friendId)
    friendList.value = friendList.value.filter(f => f.user_id !== friendId)
  }

  /** 更新好友备注 */
  const updateRemark = async (friendId, remark) => {
    await contactApi.updateRemark(friendId, remark)
    const friend = friendList.value.find(f => f.user_id === friendId)
    if (friend) friend.remark = remark
  }

  /** 拉黑用户 */
  const blockUser = async (targetId) => {
    await contactApi.blockUser(targetId)
    friendList.value = friendList.value.filter(f => f.user_id !== targetId)
  }

  /** 取消拉黑 */
  const unblockUser = async (userId) => {
    await contactApi.unblockUser(userId)
    blockList.value = blockList.value.filter(b => b.user_id !== userId)
  }

  /** 获取黑名单列表 */
  const fetchBlockList = async () => {
    const res = await contactApi.getBlockList()
    blockList.value = res.data || []
    return blockList.value
  }

  /** 获取分组列表 */
  const fetchGroups = async () => {
    const res = await contactApi.getGroups()
    groups.value = res.data || []
    return groups.value
  }

  /** 创建分组 */
  const createGroup = async (name) => {
    const res = await contactApi.createGroup(name)
    groups.value.push(res.data)
    return res.data
  }

  /** 更新分组 */
  const updateGroup = async (groupId, name, sortOrder) => {
    await contactApi.updateGroup(groupId, name, sortOrder)
    const group = groups.value.find(g => g.id === groupId)
    if (group) {
      group.name = name
      if (sortOrder !== undefined) group.sort_order = sortOrder
    }
  }

  /** 删除分组 */
  const deleteGroup = async (groupId) => {
    await contactApi.deleteGroup(groupId)
    groups.value = groups.value.filter(g => g.id !== groupId)
  }

  /** 移动好友到分组 */
  const moveToGroup = async (friendId, groupId) => {
    await contactApi.moveToGroup(friendId, groupId)
    const friend = friendList.value.find(f => f.user_id === friendId)
    if (friend) friend.group_id = groupId
  }

  /** 更新单个用户的在线状态 */
  const setOnline = (userId, online) => {
    onlineMap.value[userId] = online
    const friend = friendList.value.find(f => f.user_id === userId)
    if (friend) friend.is_online = online
  }

  /** 防止 WebSocket 事件监听重复注册 */
  let _wsInitialized = false

  /** 初始化 WebSocket 事件监听（幂等，多次调用只注册一次） */
  const initWsListeners = () => {
    if (_wsInitialized) return
    _wsInitialized = true

    wsService.on('notify.friend.request', () => {
      fetchPendingRequests()
    })

    wsService.on('contact.request.accepted', () => {
      fetchFriends()
    })

    wsService.on('user.status.online', (msg) => {
      if (msg && msg.data) {
        setOnline(msg.data.user_id, true)
      }
    })

    wsService.on('user.status.offline', (msg) => {
      if (msg && msg.data) {
        setOnline(msg.data.user_id, false)
      }
    })
  }

  return {
    friendList,
    pendingRequests,
    groups,
    blockList,
    onlineMap,
    pendingCount,
    fetchFriends,
    fetchPendingRequests,
    sendRequest,
    acceptRequest,
    rejectRequest,
    deleteFriend,
    updateRemark,
    blockUser,
    unblockUser,
    fetchBlockList,
    fetchGroups,
    createGroup,
    updateGroup,
    deleteGroup,
    moveToGroup,
    setOnline,
    initWsListeners
  }
})
