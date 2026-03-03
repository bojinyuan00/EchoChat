/**
 * IM 即时通讯模块 API
 *
 * 对应后端路由：/api/v1/im/*
 * 消息发送/撤回/标记已读通过 WebSocket 进行，此处仅包含 REST API
 */

import { get, put, del } from '@/utils/request'

/** 获取会话列表 */
const getConversations = () => {
  return get('/api/v1/im/conversations')
}

/** 获取历史消息（游标分页） */
const getHistoryMessages = (conversationId, beforeId = 0, limit = 30) => {
  return get('/api/v1/im/messages', {
    conversation_id: conversationId,
    before_id: beforeId,
    limit
  })
}

/** 置顶/取消置顶会话 */
const pinConversation = (conversationId, isPinned) => {
  return put(`/api/v1/im/conversations/${conversationId}/pin`, { is_pinned: isPinned })
}

/** 删除会话 */
const deleteConversation = (conversationId) => {
  return del(`/api/v1/im/conversations/${conversationId}`)
}

/** 清空聊天记录 */
const clearHistory = (conversationId) => {
  return del(`/api/v1/im/conversations/${conversationId}/messages`)
}

/** 全局消息搜索 */
const searchMessages = (keyword, limit = 50) => {
  return get('/api/v1/im/messages/search', { keyword, limit })
}

/** 获取全局未读消息总数 */
const getTotalUnread = () => {
  return get('/api/v1/im/unread')
}

export default {
  getConversations,
  getHistoryMessages,
  pinConversation,
  deleteConversation,
  clearHistory,
  searchMessages,
  getTotalUnread
}
