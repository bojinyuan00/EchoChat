/**
 * 用户状态管理 Store
 *
 * 管理用户认证状态和个人信息，提供：
 * - 登录/注册/登出操作
 * - Token 自动管理（存储 + 刷新）
 * - 用户信息缓存与更新
 * - 持久化存储（通过 pinia-plugin-persistedstate，H5 用 localStorage，小程序用 uni 存储）
 *
 * 对应后端 API：/api/v1/auth/*
 * 接口文档见：docs/api/frontend/auth.md
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import authApi from '@/api/auth'
import { useWebSocketStore } from '@/store/websocket'
import {
  saveToken,
  getToken,
  getRefreshToken,
  removeToken,
  saveUserInfo,
  getUserInfo,
  removeUserInfo,
  clearAll
} from '@/utils/storage'

/**
 * 用户 Store（Composition API 风格）
 *
 * state:
 * - token: Access Token
 * - userInfo: 用户信息对象
 *
 * getters:
 * - isLoggedIn: 是否已登录
 * - username: 用户名快捷访问
 * - roles: 用户角色列表
 *
 * actions:
 * - login: 用户登录
 * - register: 用户注册
 * - logout: 退出登录
 * - getProfile: 获取/刷新用户信息
 * - updateProfile: 更新个人信息
 * - changePassword: 修改密码
 * - refreshUserToken: 刷新 Token
 */
export const useUserStore = defineStore('user', () => {
  // ==================== State ====================

  /** Access Token，用于 API 认证 */
  const token = ref(getToken() || '')

  /** 用户信息对象，与后端 dto.UserInfo 结构一致 */
  const userInfo = ref(getUserInfo() || null)

  // ==================== Getters ====================

  /** 是否已登录（Token 存在视为已登录） */
  const isLoggedIn = computed(() => !!token.value)

  /** 用户名快捷访问 */
  const username = computed(() => userInfo.value?.username || '')

  /** 用户角色列表 */
  const roles = computed(() => userInfo.value?.roles || [])

  // ==================== Actions ====================

  /**
   * 处理登录/注册成功后的通用逻辑
   * 保存 Token 和用户信息到 Store 和本地存储
   *
   * @param {Object} data - 后端返回的登录响应 { token, refresh_token, expires_in, user }
   */
  const _handleAuthSuccess = (data) => {
    token.value = data.token
    userInfo.value = data.user
    saveToken(data.token, data.refresh_token, data.expires_in)
    saveUserInfo(data.user)
  }

  /**
   * 用户登录
   *
   * @param {Object} loginData - { account, password }
   * @returns {Promise<Object>} 登录响应数据
   */
  const login = async (loginData) => {
    const res = await authApi.login(loginData)
    _handleAuthSuccess(res.data)
    return res.data
  }

  /**
   * 用户注册（注册成功后自动登录）
   *
   * @param {Object} registerData - { username, email, password, nickname? }
   * @returns {Promise<Object>} 注册响应数据
   */
  const register = async (registerData) => {
    const res = await authApi.register(registerData)
    _handleAuthSuccess(res.data)
    return res.data
  }

  /**
   * 退出登录
   * 1. 调用后端 API 从 Redis 删除 Token（使 Token 立即失效）
   * 2. 清除本地存储的 Token 和用户信息
   * 3. 重置 Store 状态
   */
  const logout = async () => {
    const wsStore = useWebSocketStore()
    wsStore.disconnect()
    try {
      await authApi.logout()
    } catch (e) {
      console.warn('登出 API 调用失败，仍然清除本地状态', e)
    }
    token.value = ''
    userInfo.value = null
    clearAll()
  }

  /**
   * 获取/刷新用户信息
   * 从后端获取最新的用户资料并更新 Store 和本地缓存
   *
   * @returns {Promise<Object>} 用户信息
   */
  const getProfile = async () => {
    const res = await authApi.getProfile()
    userInfo.value = res.data
    saveUserInfo(res.data)
    return res.data
  }

  /**
   * 更新个人信息
   *
   * @param {Object} data - 需要更新的字段 { nickname?, avatar?, gender?, phone? }
   * @returns {Promise<Object>} 更新后的用户信息
   */
  const updateProfile = async (data) => {
    const res = await authApi.updateProfile(data)
    userInfo.value = res.data
    saveUserInfo(res.data)
    return res.data
  }

  /**
   * 修改密码
   *
   * @param {Object} data - { old_password, new_password }
   * @returns {Promise<Object>}
   */
  const changePassword = async (data) => {
    return await authApi.changePassword(data)
  }

  /**
   * 刷新 Token
   * 使用 Refresh Token 获取新的 Access Token
   * 新 Token 会覆盖 Redis 中的旧值（实现单设备登录）
   *
   * @returns {Promise<boolean>} 刷新是否成功
   */
  const refreshUserToken = async () => {
    const refreshTokenValue = getRefreshToken()
    if (!refreshTokenValue) return false

    try {
      const res = await authApi.refreshToken(refreshTokenValue)
      _handleAuthSuccess(res.data)
      return true
    } catch (e) {
      // Refresh Token 也过期或无效，需要重新登录
      token.value = ''
      userInfo.value = null
      clearAll()
      return false
    }
  }

  return {
    // state
    token,
    userInfo,
    // getters
    isLoggedIn,
    username,
    roles,
    // actions
    login,
    register,
    logout,
    getProfile,
    updateProfile,
    changePassword,
    refreshUserToken
  }
}, {
  /**
   * Pinia 持久化配置
   * H5 环境使用 localStorage，小程序/App 使用 uni 存储
   * 仅持久化 token 和 userInfo，避免存储过多无关数据
   */
  persist: {
    key: 'echo-user-store',
    storage: {
      getItem: (key) => uni.getStorageSync(key),
      setItem: (key, value) => uni.setStorageSync(key, value)
    },
    paths: ['token', 'userInfo']
  }
})
