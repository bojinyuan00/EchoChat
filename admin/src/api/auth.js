/**
 * 管理后台认证 API
 *
 * 所有接口均使用管理端专属路由：/api/v1/admin/auth/*
 * 后端对管理端路由做 JWT + admin 角色双重检查
 *
 * Token 在 Redis 中按 client_type 隔离存储：
 * - 管理端：echo:auth:token:admin:{user_id}
 * - 前台：echo:auth:token:frontend:{user_id}
 */
import request from '@/utils/request'

/**
 * 管理员登录
 * POST /api/v1/admin/auth/login
 *
 * @param {Object} data - { account, password }
 * @returns {Promise<Object>} { token, refresh_token, expires_in, user }
 */
export const login = (data) => {
  return request.post('/api/v1/admin/auth/login', data)
}

/**
 * 管理员登出
 * POST /api/v1/admin/auth/logout
 */
export const logout = () => {
  return request.post('/api/v1/admin/auth/logout')
}

/**
 * 获取管理员个人信息
 * GET /api/v1/admin/auth/profile
 */
export const getProfile = () => {
  return request.get('/api/v1/admin/auth/profile')
}
