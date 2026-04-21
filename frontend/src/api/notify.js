/**
 * 通知中心 API
 *
 * 对应后端路由：/api/v1/notifications/*
 * 接口文档见：docs/api/frontend/notify.md
 */

import { get, put } from '@/utils/request'

/**
 * 获取通知列表（游标分页）
 * @param {Object} [params]
 * @param {string} [params.category] - 分类过滤：all|friend|group|meeting|system
 * @param {boolean} [params.is_read] - 已读状态：true/false，不传代表全部
 * @param {number} [params.before_id] - 游标，返回 id 小于该值的通知
 * @param {number} [params.limit] - 页大小（默认 20，最大 100）
 */
const getNotifications = (params = {}) => {
  return get('/api/v1/notifications', params)
}

/** 获取未读数统计（总数 + 分类统计） */
const getUnreadCount = () => {
  return get('/api/v1/notifications/unread-count')
}

/**
 * 标记单条通知已读
 * @param {number} id - 通知 ID
 */
const markRead = (id) => {
  return put(`/api/v1/notifications/${id}/read`)
}

/**
 * 批量标记已读（可按分类过滤）
 *
 * 说明：后端通过 query 参数接收 category（`PUT /api/v1/notifications/read-all?category=friend`）
 *      为保持契约一致，此处将 category 拼接到 URL，避免放入 request body 导致后端读不到。
 *
 * @param {string} [category] - 分类过滤：friend|group|meeting|system，不传代表全部
 */
const markAllRead = (category) => {
  const url = category
    ? `/api/v1/notifications/read-all?category=${encodeURIComponent(category)}`
    : '/api/v1/notifications/read-all'
  return put(url)
}

export default {
  getNotifications,
  getUnreadCount,
  markRead,
  markAllRead
}
