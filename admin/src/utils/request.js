/**
 * HTTP 请求封装（管理后台）
 *
 * 基于 Axios 封装，提供：
 * - 基础 URL 配置（开发环境通过 Vite proxy 代理）
 * - 请求拦截：自动附加 Authorization Header
 * - 响应拦截：统一处理错误码（401 跳转登录等）
 *
 * 对应后端 API 规范见 docs/api/README.md
 */
import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'

/** Axios 实例 */
const service = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' }
})

/**
 * 请求拦截器
 * 自动从 localStorage 读取 Token 并附加到 Authorization Header
 */
service.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('admin_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

/**
 * 响应拦截器
 * 统一处理后端 { code, message, data } 响应格式
 */
service.interceptors.response.use(
  (response) => {
    const { data } = response
    if (data.code === 0) {
      return data
    }
    ElMessage.error(data.message || '请求失败')
    return Promise.reject(data)
  },
  (error) => {
    if (error.response) {
      const { status, data } = error.response
      if (status === 401) {
        ElMessage.error('登录已过期，请重新登录')
        localStorage.removeItem('admin_token')
        localStorage.removeItem('admin_user')
        router.push('/login')
      } else if (status === 403) {
        ElMessage.error(data?.message || '没有访问权限')
      } else {
        ElMessage.error(data?.message || `请求错误 (${status})`)
      }
    } else {
      ElMessage.error('网络异常，请检查网络连接')
    }
    return Promise.reject(error)
  }
)

export default service
