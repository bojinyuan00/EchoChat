/**
 * 群组管理 API 模块（管理端）
 *
 * 对应后端路由：/api/v1/admin/groups/*
 * 所有接口需要 JWT + admin 角色权限
 *
 * 接口列表：
 * - GET    /api/v1/admin/groups       群组列表（分页 + 搜索）
 * - GET    /api/v1/admin/groups/:id   群组详情（含成员列表）
 * - DELETE /api/v1/admin/groups/:id   解散群聊
 */
import request from '@/utils/request'

/**
 * 获取群组列表
 * @param {Object} params - 查询参数
 * @param {number} params.page - 页码（从 1 开始）
 * @param {number} params.page_size - 每页数量
 * @param {string} [params.keyword] - 搜索关键词（群名称）
 * @returns {Promise<{data: {total: number, list: Array}}>}
 */
export function getGroupList(params) {
  return request({
    url: '/api/v1/admin/groups',
    method: 'get',
    params
  })
}

/**
 * 获取群组详情（含成员列表）
 * @param {number} id - 群组 ID
 * @returns {Promise<{data: Object}>}
 */
export function getGroupDetail(id) {
  return request({
    url: `/api/v1/admin/groups/${id}`,
    method: 'get'
  })
}

/**
 * 解散群聊
 * @param {number} id - 群组 ID
 * @returns {Promise}
 */
export function dissolveGroup(id) {
  return request({
    url: `/api/v1/admin/groups/${id}`,
    method: 'delete'
  })
}
