/**
 * 认证模块 API
 *
 * 对应后端路由：/api/v1/auth/*
 * 接口文档见：docs/api/frontend/auth.md
 *
 * 所有 API 函数返回 Promise，resolve 时返回完整响应体 { code, message, data }
 */

import { post, get, put } from '@/utils/request'

/**
 * 用户注册
 * POST /api/v1/auth/register
 *
 * @param {Object} data - 注册信息
 * @param {string} data.username - 用户名（3-50 字符，全局唯一）
 * @param {string} data.email - 邮箱地址（全局唯一）
 * @param {string} data.password - 登录密码（6-50 字符）
 * @param {string} [data.nickname] - 昵称（选填，默认与用户名相同）
 * @returns {Promise<Object>} { token, refresh_token, expires_in, user }
 */
const register = (data) => {
  return post('/api/v1/auth/register', data, { needAuth: false })
}

/**
 * 用户登录
 * POST /api/v1/auth/login
 *
 * @param {Object} data - 登录信息
 * @param {string} data.account - 用户名或邮箱（自动识别）
 * @param {string} data.password - 登录密码
 * @returns {Promise<Object>} { token, refresh_token, expires_in, user }
 */
const login = (data) => {
  return post('/api/v1/auth/login', data, { needAuth: false })
}

/**
 * 退出登录
 * POST /api/v1/auth/logout
 *
 * 服务端会从 Redis 中删除该用户的 Token，使其立即失效
 * 客户端需同步清除本地存储的 Token（由 store 层处理）
 *
 * @returns {Promise<Object>}
 */
const logout = () => {
  return post('/api/v1/auth/logout')
}

/**
 * 刷新 Token
 * POST /api/v1/auth/refresh-token
 *
 * 使用 Refresh Token 获取新的 Access Token 和 Refresh Token
 * 新 Token 会覆盖 Redis 中的旧值（单设备登录）
 *
 * @param {string} refreshToken - Refresh Token
 * @returns {Promise<Object>} { token, refresh_token, expires_in, user }
 */
const refreshToken = (refreshToken) => {
  return post('/api/v1/auth/refresh-token', { refresh_token: refreshToken }, { needAuth: false })
}

/**
 * 获取个人信息
 * GET /api/v1/auth/profile
 *
 * 返回完整用户信息（含 phone、created_at 等额外字段）
 *
 * @returns {Promise<Object>} UserInfo 对象
 */
const getProfile = () => {
  return get('/api/v1/auth/profile')
}

/**
 * 更新个人信息
 * PUT /api/v1/auth/profile
 *
 * @param {Object} data - 需要更新的字段（均为可选）
 * @param {string} [data.nickname] - 新昵称
 * @param {string} [data.avatar] - 新头像 URL
 * @param {number} [data.gender] - 性别（0=未知，1=男，2=女）
 * @param {string} [data.phone] - 手机号
 * @returns {Promise<Object>} 更新后的完整 UserInfo
 */
const updateProfile = (data) => {
  return put('/api/v1/auth/profile', data)
}

/**
 * 修改密码
 * PUT /api/v1/auth/password
 *
 * @param {Object} data - 密码信息
 * @param {string} data.old_password - 原密码
 * @param {string} data.new_password - 新密码（6-50 字符）
 * @returns {Promise<Object>}
 */
const changePassword = (data) => {
  return put('/api/v1/auth/password', data)
}

export default {
  register,
  login,
  logout,
  refreshToken,
  getProfile,
  updateProfile,
  changePassword
}
