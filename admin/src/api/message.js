/**
 * 消息管理 API 模块
 *
 * 对应后端路由：/api/v1/admin/messages
 * 所有接口需要 JWT + admin 角色权限
 *
 * 接口列表：
 * - GET    /api/v1/admin/messages          消息列表（分页 + 多条件筛选）
 * - GET    /api/v1/admin/messages/:id      消息详情
 * - DELETE /api/v1/admin/messages/:id      删除消息（软删除）
 * - PUT    /api/v1/admin/messages/:id/recall  撤回消息
 * - GET    /api/v1/admin/messages/stats    消息统计
 */
import request from '@/utils/request'

/**
 * 获取消息列表
 * @param {Object} params - 查询参数
 * @param {number} [params.page] - 页码（默认 1）
 * @param {number} [params.page_size] - 每页数量（默认 20）
 * @param {string} [params.keyword] - 搜索关键词（模糊匹配消息内容）
 * @param {number} [params.type] - 消息类型筛选（1=文本/2=图片/3=语音/5=文件/10=系统）
 * @param {number} [params.sender_id] - 发送者 ID
 * @param {number} [params.conversation_id] - 会话 ID
 * @param {number} [params.status] - 消息状态（1=正常/2=已撤回/3=已删除）
 * @param {string} [params.start_time] - 开始时间（YYYY-MM-DD）
 * @param {string} [params.end_time] - 结束时间（YYYY-MM-DD）
 * @returns {Promise<{data: {total: number, list: Array, page: number, page_size: number}}>}
 */
export function getMessageList(params) {
  return request({
    url: '/api/v1/admin/messages',
    method: 'get',
    params
  })
}

/**
 * 获取消息详情
 * @param {number} id - 消息 ID
 * @returns {Promise<{data: Object}>}
 */
export function getMessageDetail(id) {
  return request({
    url: `/api/v1/admin/messages/${id}`,
    method: 'get'
  })
}

/**
 * 删除消息（软删除）
 * @param {number} id - 消息 ID
 * @returns {Promise}
 */
export function deleteMessage(id) {
  return request({
    url: `/api/v1/admin/messages/${id}`,
    method: 'delete'
  })
}

/**
 * 撤回消息
 * @param {number} id - 消息 ID
 * @returns {Promise}
 */
export function recallMessage(id) {
  return request({
    url: `/api/v1/admin/messages/${id}/recall`,
    method: 'put'
  })
}

/**
 * 获取消息统计数据
 * @param {Object} params - 查询参数
 * @param {number} [params.days] - 统计天数（默认 7，最大 90）
 * @returns {Promise<{data: {total_count, today_count, type_distribution, daily_trend, active_users, active_groups}}>}
 */
export function getMessageStats(params) {
  return request({
    url: '/api/v1/admin/messages/stats',
    method: 'get',
    params
  })
}
