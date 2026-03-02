/**
 * 管理后台认证 API
 *
 * 对应后端路由：/api/v1/auth/*
 * 管理员登录使用与普通用户相同的登录接口，
 * 权限区分由 JWT 中的 role 字段和 RequireRole 中间件控制
 */
import request from '@/utils/request'

/**
 * 管理员登录
 * POST /api/v1/auth/login
 *
 * @param {Object} data - { account, password }
 * @returns {Promise<Object>} { token, refresh_token, expires_in, user }
 */
export const login = (data) => {
  return request.post('/api/v1/auth/login', data)
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
