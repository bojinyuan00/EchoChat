/**
 * 好友关系管理 API 模块（管理端）
 *
 * 对应后端路由：/api/v1/admin/contacts/*
 * 所有接口需要 JWT + admin 角色权限
 */
import request from '@/utils/request'

/**
 * 获取所有好友关系列表（分页）
 * @param {Object} params - 查询参数
 * @param {number} params.page - 页码
 * @param {number} params.page_size - 每页数量
 * @returns {Promise<{data: {total: number, list: Array}}>}
 */
export function getAllContacts(params) {
  return request({
    url: '/api/v1/admin/contacts',
    method: 'get',
    params
  })
}

/**
 * 删除好友关系
 * @param {number} id - 好友关系 ID
 * @returns {Promise}
 */
export function deleteContact(id) {
  return request({
    url: `/api/v1/admin/contacts/${id}`,
    method: 'delete'
  })
}
