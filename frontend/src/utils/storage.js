/**
 * 本地存储工具模块
 *
 * 封装 uni.setStorageSync / uni.getStorageSync，提供：
 * - Token 存取（Access Token + Refresh Token）
 * - 用户信息缓存
 * - 通用存储操作
 *
 * 存储方式：
 * - H5 环境：localStorage
 * - 小程序/App：uni 内置存储
 * - 均通过 uni.setStorageSync/getStorageSync 统一调用
 */

// 存储 Key 常量，集中管理避免硬编码
const STORAGE_KEYS = {
  /** Access Token，用于 API 认证 */
  ACCESS_TOKEN: 'echo_access_token',
  /** Refresh Token，用于刷新 Access Token */
  REFRESH_TOKEN: 'echo_refresh_token',
  /** Access Token 过期时间戳（毫秒） */
  TOKEN_EXPIRE_TIME: 'echo_token_expire_time',
  /** 用户信息缓存 */
  USER_INFO: 'echo_user_info'
}

// ==================== Token 操作 ====================

/**
 * 保存 Token 信息（登录/注册成功后调用）
 *
 * @param {string} accessToken - Access Token
 * @param {string} refreshToken - Refresh Token
 * @param {number} expiresIn - Access Token 有效期（秒）
 */
const saveToken = (accessToken, refreshToken, expiresIn) => {
  uni.setStorageSync(STORAGE_KEYS.ACCESS_TOKEN, accessToken)
  uni.setStorageSync(STORAGE_KEYS.REFRESH_TOKEN, refreshToken)
  // 计算并存储过期时间戳，用于前端主动判断是否需要刷新
  const expireTime = Date.now() + expiresIn * 1000
  uni.setStorageSync(STORAGE_KEYS.TOKEN_EXPIRE_TIME, expireTime)
}

/**
 * 获取 Access Token
 *
 * @returns {string|null} Access Token，不存在时返回 null
 */
const getToken = () => {
  return uni.getStorageSync(STORAGE_KEYS.ACCESS_TOKEN) || null
}

/**
 * 获取 Refresh Token
 *
 * @returns {string|null} Refresh Token，不存在时返回 null
 */
const getRefreshToken = () => {
  return uni.getStorageSync(STORAGE_KEYS.REFRESH_TOKEN) || null
}

/**
 * 判断 Access Token 是否即将过期（提前 5 分钟视为过期）
 * 用于在请求前主动触发 Token 刷新
 *
 * @returns {boolean} true 表示已过期或即将过期
 */
const isTokenExpired = () => {
  const expireTime = uni.getStorageSync(STORAGE_KEYS.TOKEN_EXPIRE_TIME)
  if (!expireTime) return true
  // 提前 5 分钟（300秒）判定为过期，给刷新操作留出时间窗口
  return Date.now() >= expireTime - 300 * 1000
}

/**
 * 清除所有 Token（登出时调用）
 * 同时清除服务端 Redis 中的 Token（通过 logout API），这里只清除客户端缓存
 */
const removeToken = () => {
  uni.removeStorageSync(STORAGE_KEYS.ACCESS_TOKEN)
  uni.removeStorageSync(STORAGE_KEYS.REFRESH_TOKEN)
  uni.removeStorageSync(STORAGE_KEYS.TOKEN_EXPIRE_TIME)
}

// ==================== 用户信息操作 ====================

/**
 * 缓存用户信息
 *
 * @param {Object} userInfo - 用户信息对象（与 dto.UserInfo 结构一致）
 */
const saveUserInfo = (userInfo) => {
  uni.setStorageSync(STORAGE_KEYS.USER_INFO, JSON.stringify(userInfo))
}

/**
 * 获取缓存的用户信息
 *
 * @returns {Object|null} 用户信息对象，不存在时返回 null
 */
const getUserInfo = () => {
  const data = uni.getStorageSync(STORAGE_KEYS.USER_INFO)
  if (!data) return null
  try {
    return JSON.parse(data)
  } catch (e) {
    return null
  }
}

/**
 * 清除用户信息缓存
 */
const removeUserInfo = () => {
  uni.removeStorageSync(STORAGE_KEYS.USER_INFO)
}

// ==================== 通用存储操作 ====================

/**
 * 清除所有应用数据（重置应用状态）
 * 谨慎使用，会清除所有本地存储
 */
const clearAll = () => {
  removeToken()
  removeUserInfo()
}

export {
  STORAGE_KEYS,
  saveToken,
  getToken,
  getRefreshToken,
  isTokenExpired,
  removeToken,
  saveUserInfo,
  getUserInfo,
  removeUserInfo,
  clearAll
}
