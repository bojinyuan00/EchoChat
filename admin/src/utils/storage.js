/**
 * 本地存储工具（管理后台）
 *
 * 管理端使用 localStorage，Key 前缀 admin_ 与前台隔离
 */

const KEYS = {
  TOKEN: 'admin_token',
  REFRESH_TOKEN: 'admin_refresh_token',
  USER_INFO: 'admin_user'
}

/** 保存 Token */
export const saveToken = (token, refreshToken) => {
  localStorage.setItem(KEYS.TOKEN, token)
  if (refreshToken) localStorage.setItem(KEYS.REFRESH_TOKEN, refreshToken)
}

/** 获取 Access Token */
export const getToken = () => localStorage.getItem(KEYS.TOKEN)

/** 获取 Refresh Token */
export const getRefreshToken = () => localStorage.getItem(KEYS.REFRESH_TOKEN)

/** 清除 Token */
export const removeToken = () => {
  localStorage.removeItem(KEYS.TOKEN)
  localStorage.removeItem(KEYS.REFRESH_TOKEN)
}

/** 保存用户信息 */
export const saveUserInfo = (user) => {
  localStorage.setItem(KEYS.USER_INFO, JSON.stringify(user))
}

/** 获取用户信息 */
export const getUserInfo = () => {
  try {
    return JSON.parse(localStorage.getItem(KEYS.USER_INFO))
  } catch {
    return null
  }
}

/** 清除用户信息 */
export const removeUserInfo = () => {
  localStorage.removeItem(KEYS.USER_INFO)
}

/** 清除所有管理端缓存 */
export const clearAll = () => {
  removeToken()
  removeUserInfo()
}
