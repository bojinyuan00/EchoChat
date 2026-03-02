/**
 * 管理后台认证 API
 *
 * 登录接口：/api/v1/admin/auth/login（管理端专用，后端检查管理员角色）
 * 其他接口：/api/v1/auth/*（与前台共用，通过 JWT 中的 client_type 区分）
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
 * POST /api/v1/auth/logout
 */
export const logout = () => {
  return request.post('/api/v1/auth/logout')
}

/**
 * 获取管理员个人信息
 * GET /api/v1/auth/profile
 */
export const getProfile = () => {
  return request.get('/api/v1/auth/profile')
}
