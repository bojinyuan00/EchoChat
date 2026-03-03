/**
 * 即时通讯状态 Store
 *
 * 管理会话列表、消息、未读数，提供：
 * - 会话列表 CRUD + 排序（置顶优先 → 最后消息时间降序）
 * - 当前会话消息管理（发送/接收/撤回/历史加载）
 * - 全局未读数（TabBar badge 显示）
 * - WebSocket 事件监听（im.message.new / im.message.recalled / im.offline.sync）
 * - 三态消息确认（sending → sent → failed）
 *
 * 对应后端 API：/api/v1/im/*
 * 对应 WS 事件：im.message.send / im.message.recall / im.conversation.read / im.typing
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import imApi from '@/api/im'
import wsService from '@/services/websocket'

export const useChatStore = defineStore('chat', () => {
  /** 会话列表 */
  const conversationList = ref([])

  /** 当前聊天会话 ID */
  const currentConversationId = ref(null)

  /** 消息缓存 conversationId -> Message[] */
  const messagesMap = ref({})

  /** 发送中消息队列 clientMsgId -> { status, ... } */
  const pendingMessages = ref({})

  /** 全局未读消息总数（用于 TabBar badge） */
  const totalUnread = ref(0)

  /** 正在输入状态 conversationId -> boolean */
  const typingMap = ref({})

  /** 是否还有更多历史消息 conversationId -> boolean */
  const hasMoreMap = ref({})

  // ==================== Computed ====================

  /** 当前会话的消息列表 */
  const currentMessages = computed(() => {
    if (!currentConversationId.value) return []
    return messagesMap.value[currentConversationId.value] || []
  })

  /** 按排序规则的会话列表（置顶优先 → 最后消息时间降序） */
  const sortedConversations = computed(() => {
    return [...conversationList.value].sort((a, b) => {
      if (a.is_pinned !== b.is_pinned) return a.is_pinned ? -1 : 1
      const timeA = a.last_msg_time ? new Date(a.last_msg_time).getTime() : 0
      const timeB = b.last_msg_time ? new Date(b.last_msg_time).getTime() : 0
      return timeB - timeA
    })
  })

  // ==================== Actions ====================

  /** 加载会话列表 */
  const fetchConversations = async () => {
    const res = await imApi.getConversations()
    conversationList.value = (res.data && res.data.list) || []
    _recalcTotalUnread()
    return conversationList.value
  }

  /** 加载全局未读数（从后端获取，同步 Redis 数据） */
  const fetchTotalUnread = async () => {
    const res = await imApi.getTotalUnread()
    if (res.data) {
      totalUnread.value = res.data.total_unread || 0
    }
  }

  /**
   * 发送消息
   * @param {Object} params - { conversationId, targetUserId, content, type }
   * @returns {string} clientMsgId
   */
  const sendMessage = (params) => {
    const clientMsgId = _generateClientMsgId()
    const { conversationId, targetUserId, content, type = 1 } = params

    const tempMsg = {
      id: 0,
      conversation_id: conversationId || 0,
      sender_id: 0,
      type,
      content,
      status: 1,
      client_msg_id: clientMsgId,
      created_at: new Date().toISOString().replace('T', ' ').substring(0, 19),
      _sending: true
    }

    if (conversationId) {
      _appendMessage(conversationId, tempMsg)
    }

    pendingMessages.value[clientMsgId] = { status: 'sending', tempMsg }

    const seq = wsService.send('im.message.send', {
      conversation_id: conversationId || 0,
      target_user_id: targetUserId || 0,
      type,
      content,
      client_msg_id: clientMsgId
    })

    if (seq === -1) {
      pendingMessages.value[clientMsgId].status = 'failed'
      tempMsg._sending = false
      tempMsg._failed = true
    }

    return clientMsgId
  }

  /** 撤回消息 */
  const recallMessage = (messageId) => {
    wsService.send('im.message.recall', { message_id: messageId })
  }

  /** 标记会话已读 */
  const markRead = (conversationId) => {
    const conv = conversationList.value.find(c => c.id === conversationId)
    if (conv && conv.unread_count > 0) {
      totalUnread.value = Math.max(0, totalUnread.value - conv.unread_count)
      conv.unread_count = 0
    }
    wsService.send('im.conversation.read', { conversation_id: conversationId })
  }

  /** 发送正在输入通知 */
  const sendTyping = (conversationId) => {
    wsService.send('im.typing', { conversation_id: conversationId })
  }

  /** 加载历史消息 */
  const loadHistoryMessages = async (conversationId) => {
    const messages = messagesMap.value[conversationId] || []
    const beforeId = messages.length > 0 ? messages[0].id : 0

    const res = await imApi.getHistoryMessages(conversationId, beforeId)
    if (res.data) {
      const list = res.data.list || []
      hasMoreMap.value[conversationId] = res.data.has_more

      if (list.length > 0) {
        const existing = messagesMap.value[conversationId] || []
        messagesMap.value[conversationId] = [...list.reverse(), ...existing]
      }
    }
  }

  /** 置顶/取消置顶 */
  const pinConversation = async (conversationId, isPinned) => {
    await imApi.pinConversation(conversationId, isPinned)
    const conv = conversationList.value.find(c => c.id === conversationId)
    if (conv) conv.is_pinned = isPinned
  }

  /** 删除会话 */
  const deleteConversation = async (conversationId) => {
    await imApi.deleteConversation(conversationId)
    conversationList.value = conversationList.value.filter(c => c.id !== conversationId)
    if (currentConversationId.value === conversationId) {
      currentConversationId.value = null
    }
    _recalcTotalUnread()
  }

  /** 清空聊天记录 */
  const clearHistory = async (conversationId) => {
    await imApi.clearHistory(conversationId)
    messagesMap.value[conversationId] = []
    const conv = conversationList.value.find(c => c.id === conversationId)
    if (conv) {
      conv.last_msg_content = ''
      conv.last_msg_time = ''
    }
  }

  /** 设置当前会话 */
  const setCurrentConversation = (conversationId) => {
    currentConversationId.value = conversationId
    if (conversationId) {
      markRead(conversationId)
    }
  }

  // ==================== WS 事件监听 ====================

  let _wsInitialized = false

  /** 初始化 WebSocket 事件监听（幂等） */
  const initWsListeners = () => {
    if (_wsInitialized) return
    _wsInitialized = true

    wsService.on('im.message.new', _onNewMessage)
    wsService.on('im.message.recalled', _onMessageRecalled)
    wsService.on('im.message.send.ack', _onSendACK)
    wsService.on('im.offline.sync', _onOfflineSync)
    wsService.on('im.typing', _onTyping)
  }

  /** 收到新消息推送 */
  const _onNewMessage = (msg) => {
    if (!msg || !msg.data) return
    const data = msg.data

    _appendMessage(data.conversation_id, {
      id: data.id,
      conversation_id: data.conversation_id,
      sender_id: data.sender_id,
      type: data.type,
      content: data.content,
      status: 1,
      client_msg_id: data.client_msg_id || '',
      created_at: data.created_at
    })

    _updateConversationPreview(data.conversation_id, data.content, data.created_at, data.sender_id)

    if (data.conversation_id !== currentConversationId.value) {
      const conv = conversationList.value.find(c => c.id === data.conversation_id)
      if (conv) {
        conv.unread_count = (conv.unread_count || 0) + 1
      }
      totalUnread.value++
    } else {
      markRead(data.conversation_id)
    }
  }

  /** 消息撤回推送 */
  const _onMessageRecalled = (msg) => {
    if (!msg || !msg.data) return
    const { message_id, conversation_id } = msg.data
    const messages = messagesMap.value[conversation_id]
    if (messages) {
      const idx = messages.findIndex(m => m.id === message_id)
      if (idx !== -1) {
        messages[idx].status = 2
        messages[idx].content = '[消息已撤回]'
      }
    }
  }

  /** 发送消息 ACK 响应 */
  const _onSendACK = (msg) => {
    if (!msg) return
    const { code, data } = msg
    if (!data || !data.client_msg_id) return

    const pending = pendingMessages.value[data.client_msg_id]
    if (!pending) return

    if (code === 0) {
      pending.status = 'sent'
      const convId = data.conversation_id
      const messages = messagesMap.value[convId]
      if (messages) {
        const idx = messages.findIndex(m => m.client_msg_id === data.client_msg_id)
        if (idx !== -1) {
          messages[idx] = { ...messages[idx], ...data, _sending: false }
        } else {
          _appendMessage(convId, { ...data, _sending: false })
        }
      } else {
        _appendMessage(convId, { ...data, _sending: false })
      }
      _updateConversationPreview(convId, data.content, data.created_at, data.sender_id)
    } else {
      pending.status = 'failed'
      if (pending.tempMsg) {
        pending.tempMsg._sending = false
        pending.tempMsg._failed = true
      }
    }
    delete pendingMessages.value[data.client_msg_id]
  }

  /** 离线消息同步推送 */
  const _onOfflineSync = (msg) => {
    if (!msg || !msg.data) return
    const { total_unread } = msg.data
    totalUnread.value = total_unread || 0
    fetchConversations()
  }

  /** 正在输入通知 */
  const _onTyping = (msg) => {
    if (!msg || !msg.data) return
    const { conversation_id } = msg.data
    typingMap.value[conversation_id] = true
    setTimeout(() => {
      typingMap.value[conversation_id] = false
    }, 3000)
  }

  // ==================== 内部工具方法 ====================

  /** 追加消息到缓存 */
  const _appendMessage = (conversationId, message) => {
    if (!messagesMap.value[conversationId]) {
      messagesMap.value[conversationId] = []
    }
    messagesMap.value[conversationId].push(message)
  }

  /** 更新会话列表中的预览信息 */
  const _updateConversationPreview = (conversationId, content, time, senderId) => {
    const conv = conversationList.value.find(c => c.id === conversationId)
    if (conv) {
      conv.last_msg_content = content
      conv.last_msg_time = time
      conv.last_msg_sender_id = senderId
    }
  }

  /** 根据会话列表重新计算总未读数 */
  const _recalcTotalUnread = () => {
    totalUnread.value = conversationList.value.reduce((sum, c) => sum + (c.unread_count || 0), 0)
  }

  /** 生成客户端消息唯一 ID */
  const _generateClientMsgId = () => {
    return `${Date.now()}-${Math.random().toString(36).substring(2, 10)}`
  }

  return {
    conversationList,
    currentConversationId,
    messagesMap,
    pendingMessages,
    totalUnread,
    typingMap,
    hasMoreMap,
    currentMessages,
    sortedConversations,
    fetchConversations,
    fetchTotalUnread,
    sendMessage,
    recallMessage,
    markRead,
    sendTyping,
    loadHistoryMessages,
    pinConversation,
    deleteConversation,
    clearHistory,
    setCurrentConversation,
    initWsListeners
  }
})
