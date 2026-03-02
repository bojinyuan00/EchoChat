/**
 * 管理后台用户 Store
 *
 * 管理管理员认证状态、个人信息
 * 使用 Pinia 3.x + Composition API 风格
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, logout as logoutApi, getProfile as getProfileApi } from '@/api/auth'
import { saveToken, getToken, removeToken, saveUserInfo, getUserInfo, removeUserInfo, clearAll } from '@/utils/storage'

export const useUserStore = defineStore('admin-user', () => {
  /** Access Token */
  const token = ref(getToken() || '')

  /** 管理员信息 */
  const userInfo = ref(getUserInfo() || null)

  /** 是否已登录 */
  const isLoggedIn = computed(() => !!token.value)

  /** 管理员用户名 */
  const username = computed(() => userInfo.value?.username || '')

  /** 角色列表 */
  const roles = computed(() => userInfo.value?.roles || [])

  /**
   * 管理员登录
   * @param {Object} loginData - { account, password }
   */
  const login = async (loginData) => {
    const res = await loginApi(loginData)
    token.value = res.data.token
    userInfo.value = res.data.user
    saveToken(res.data.token, res.data.refresh_token)
    saveUserInfo(res.data.user)
    return res.data
  }

  /** 退出登录 */
  const logout = async () => {
    try {
      await logoutApi()
    } catch {
      // 即使后端请求失败也清除本地状态
    }
    token.value = ''
    userInfo.value = null
    clearAll()
  }

  /** 获取/刷新管理员信息 */
  const getProfile = async () => {
    const res = await getProfileApi()
    userInfo.value = res.data
    saveUserInfo(res.data)
    return res.data
  }

  return {
    token,
    userInfo,
    isLoggedIn,
    username,
    roles,
    login,
    logout,
    getProfile
  }
})
