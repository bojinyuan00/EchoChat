/**
 * 用户管理 API 模块
 *
 * 对应后端路由：/api/v1/admin/users, /api/v1/admin/roles
 * 所有接口需要 JWT + admin 角色权限
 *
 * 接口列表：
 * - GET    /api/v1/admin/users            用户列表（分页 + 搜索 + 筛选）
 * - GET    /api/v1/admin/users/:id        用户详情
 * - PUT    /api/v1/admin/users/:id/status 更新用户状态
 * - PUT    /api/v1/admin/users/:id/roles  批量设置角色
 * - POST   /api/v1/admin/users            管理员创建用户
 * - GET    /api/v1/admin/roles            获取所有角色列表
 */
import request from '@/utils/request'

/**
 * 获取用户列表
 * @param {Object} params - 查询参数
 * @param {number} params.page - 页码（从 1 开始）
 * @param {number} params.page_size - 每页数量
 * @param {string} [params.keyword] - 搜索关键词（用户名/邮箱）
 * @param {number} [params.status] - 状态筛选：1=正常, 2=禁用
 * @returns {Promise<{data: {total: number, list: Array}}>}
 */
export function getUserList(params) {
  return request({
    url: '/api/v1/admin/users',
    method: 'get',
    params
  })
}

/**
 * 获取用户详情
 * @param {number} id - 用户 ID
 * @returns {Promise<{data: Object}>}
 */
export function getUserDetail(id) {
  return request({
    url: `/api/v1/admin/users/${id}`,
    method: 'get'
  })
}

/**
 * 更新用户状态
 * @param {number} id - 用户 ID
 * @param {number} status - 目标状态：1=正常, 2=禁用
 * @returns {Promise}
 */
export function updateUserStatus(id, status) {
  return request({
    url: `/api/v1/admin/users/${id}/status`,
    method: 'put',
    data: { status }
  })
}

/**
 * 批量设置用户角色（先清后设）
 * @param {number} id - 用户 ID
 * @param {string[]} roleCodes - 角色代码列表
 * @returns {Promise}
 */
export function setUserRoles(id, roleCodes) {
  return request({
    url: `/api/v1/admin/users/${id}/roles`,
    method: 'put',
    data: { role_codes: roleCodes }
  })
}

/**
 * 获取所有角色列表（含 level 等级信息）
 * @returns {Promise<{data: Array<{code: string, name: string, level: number}>}>}
 */
export function getAllRoles() {
  return request({
    url: '/api/v1/admin/roles',
    method: 'get'
  })
}

/**
 * 管理员创建用户
 * @param {Object} data - 用户信息
 * @param {string} data.username - 用户名
 * @param {string} data.email - 邮箱
 * @param {string} data.password - 初始密码
 * @param {string} [data.nickname] - 昵称
 * @param {string} [data.role_code] - 初始角色
 * @returns {Promise<{data: Object}>}
 */
export function createUser(data) {
  return request({
    url: '/api/v1/admin/users',
    method: 'post',
    data
  })
}
