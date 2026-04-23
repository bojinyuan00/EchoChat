/**
 * HTTP 请求封装模块
 *
 * 基于 uni.request 封装统一的请求方法，提供：
 * - 基础 URL 配置（区分开发/生产环境）
 * - 请求拦截：自动附加 Authorization Header
 * - 响应拦截：统一处理错误码（401 自动跳转登录等）
 * - Promise 化的请求接口
 *
 * 对应后端 API 规范见 docs/api/README.md
 */

import { getToken, removeToken } from './storage'

// 基础 URL 配置
// 开发 / 生产环境都用相对路径（空串）：
// - 开发：vite 配置 server.proxy 把 /api /ws 转发到后端，同时开启 HTTPS；前端同源调用 /api/** 即可，
//   同源意味着浏览器在 HTTPS 页面下不会因混合内容拒绝请求，且 WebRTC 所需的 secure context 自然满足。
// - 生产：由 Nginx 反向代理到 Go 后端，同样同源。
// 如果需要直连其他主机（极少数场景），可通过 VITE_API_BASE 在构建期覆盖。
// eslint-disable-next-line no-undef
const BASE_URL = (typeof __VITE_API_BASE__ !== 'undefined' && __VITE_API_BASE__) || ''

// 请求超时时间（毫秒）
const TIMEOUT = 15000

/**
 * 统一请求方法
 *
 * @param {Object} options - 请求配置
 * @param {string} options.url - 请求路径（不含 BASE_URL，如 /api/v1/auth/login）
 * @param {string} [options.method='GET'] - HTTP 方法
 * @param {Object} [options.data] - 请求体数据
 * @param {Object} [options.params] - URL Query 参数
 * @param {Object} [options.header] - 自定义请求头
 * @param {boolean} [options.needAuth=true] - 是否需要自动附加 Token
 * @param {boolean} [options.silent=false] - 是否抑制失败 toast（用于内部重试 / 清理等静默流程）
 * @returns {Promise<Object>} 响应中的 data 字段（已解包）
 */
const request = (options) => {
  const silent = options.silent === true
  const toastIfNotSilent = (toast) => {
    if (!silent) uni.showToast(toast)
  }
  return new Promise((resolve, reject) => {
    // 构建请求头
    const header = {
      'Content-Type': 'application/json',
      ...options.header
    }

    // 自动附加 Authorization Header（默认需要认证）
    const needAuth = options.needAuth !== false
    if (needAuth) {
      const token = getToken()
      if (token) {
        header['Authorization'] = `Bearer ${token}`
      }
    }

    uni.request({
      url: `${BASE_URL}${options.url}`,
      method: options.method || 'GET',
      data: options.data,
      header,
      timeout: TIMEOUT,
      success: (res) => {
        const { statusCode, data } = res

        // HTTP 状态码 2xx 视为成功
        if (statusCode >= 200 && statusCode < 300) {
          // 后端统一响应格式：{ code: 0, message: "success", data: ... }
          if (data.code === 0) {
            resolve(data)
          } else {
            // code 非 0 表示业务逻辑错误
            toastIfNotSilent({
              title: data.message || '请求失败',
              icon: 'none',
              duration: 2000
            })
            reject(data)
          }
        } else if (statusCode === 401) {
          const isAuthEndpoint = /\/auth\/(login|register)$/.test(options.url)
          const message = data?.message || '登录已过期，请重新登录'
          toastIfNotSilent({ title: message, icon: 'none', duration: 2000 })
          if (!isAuthEndpoint) {
            removeToken()
            setTimeout(() => {
              uni.reLaunch({ url: '/pages/auth/login' })
            }, 1500)
          }
          reject({ code: 401, message })
        } else if (statusCode === 403) {
          toastIfNotSilent({
            title: data.message || '没有访问权限',
            icon: 'none',
            duration: 2000
          })
          reject(data)
        } else {
          // 其他 HTTP 错误（400/404/500 等）
          toastIfNotSilent({
            title: data.message || `请求错误 (${statusCode})`,
            icon: 'none',
            duration: 2000
          })
          reject(data)
        }
      },
      fail: (err) => {
        // 网络错误或请求超时
        toastIfNotSilent({
          title: '网络异常，请检查网络连接',
          icon: 'none',
          duration: 2000
        })
        reject({ code: -1, message: err.errMsg || '网络异常' })
      }
    })
  })
}

/**
 * GET 请求快捷方法
 *
 * @param {string} url - 请求路径
 * @param {Object} [data] - Query 参数
 * @param {Object} [options] - 额外配置
 * @returns {Promise<Object>}
 */
const get = (url, data, options = {}) => {
  return request({ url, method: 'GET', data, ...options })
}

/**
 * POST 请求快捷方法
 *
 * @param {string} url - 请求路径
 * @param {Object} [data] - 请求体数据
 * @param {Object} [options] - 额外配置
 * @returns {Promise<Object>}
 */
const post = (url, data, options = {}) => {
  return request({ url, method: 'POST', data, ...options })
}

/**
 * PUT 请求快捷方法
 *
 * @param {string} url - 请求路径
 * @param {Object} [data] - 请求体数据
 * @param {Object} [options] - 额外配置
 * @returns {Promise<Object>}
 */
const put = (url, data, options = {}) => {
  return request({ url, method: 'PUT', data, ...options })
}

/**
 * DELETE 请求快捷方法
 *
 * @param {string} url - 请求路径
 * @param {Object} [data] - 请求体数据
 * @param {Object} [options] - 额外配置
 * @returns {Promise<Object>}
 */
const del = (url, data, options = {}) => {
  return request({ url, method: 'DELETE', data, ...options })
}

export {
  request,
  get,
  post,
  put,
  del,
  BASE_URL
}
