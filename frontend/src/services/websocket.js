/**
 * WebSocket 连接管理服务
 *
 * 单例模式，管理与后端的 WebSocket 长连接，提供：
 * - JWT Token 认证连接
 * - 心跳保活（30s 间隔）
 * - 断线自动重连（指数退避 1s→2s→4s→8s→30s max）
 * - 事件分发：on(event, callback) / off(event, callback)
 *
 * 对应后端：GET /ws?token=xxx
 */

import { BASE_URL } from '@/utils/request'
import { getToken } from '@/utils/storage'

const WS_BASE = BASE_URL.replace(/^http/, 'ws')
const HEARTBEAT_INTERVAL = 30000
const RECONNECT_BASE_DELAY = 1000
const RECONNECT_MAX_DELAY = 30000
const MAX_RECONNECT_ATTEMPTS = Infinity

class WebSocketService {
  constructor() {
    this.ws = null
    this.listeners = new Map()
    this.heartbeatTimer = null
    this.reconnectTimer = null
    this.reconnectAttempts = 0
    this.isManualClose = false
    this.seq = 0
  }

  /**
   * 建立 WebSocket 连接
   * @param {string} [token] - JWT Token，不传则从本地存储获取
   */
  connect(token) {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    const accessToken = token || getToken()
    if (!accessToken) {
      console.warn('[WS] 无 Token，无法连接')
      return
    }

    this.isManualClose = false
    const url = `${WS_BASE}/ws?token=${accessToken}`

    try {
      // #ifdef H5
      this.ws = new WebSocket(url)
      // #endif

      // #ifndef H5
      this.ws = uni.connectSocket({
        url,
        complete: () => {}
      })
      // #endif

      this._bindEvents()
    } catch (err) {
      console.error('[WS] 连接创建失败', err)
      this._scheduleReconnect()
    }
  }

  /** 主动断开连接 */
  disconnect() {
    this.isManualClose = true
    this._clearTimers()
    if (this.ws) {
      // #ifdef H5
      this.ws.close()
      // #endif
      // #ifndef H5
      uni.closeSocket()
      // #endif
      this.ws = null
    }
  }

  /**
   * 发送消息
   * @param {string} event - 事件类型
   * @param {Object} [data] - 业务数据
   * @returns {number} 消息序号
   */
  send(event, data = {}) {
    // #ifdef H5
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('[WS] 连接未就绪，无法发送')
      return -1
    }
    // #endif
    // #ifndef H5
    if (!this.ws) {
      console.warn('[WS] 连接未就绪，无法发送')
      return -1
    }
    // #endif

    this.seq++
    const msg = JSON.stringify({
      event,
      seq: this.seq,
      data,
      time: new Date().toISOString()
    })

    // #ifdef H5
    this.ws.send(msg)
    // #endif
    // #ifndef H5
    uni.sendSocketMessage({ data: msg })
    // #endif

    return this.seq
  }

  /**
   * 注册事件监听
   * @param {string} event - 事件类型
   * @param {Function} callback - 回调函数
   */
  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event).add(callback)
  }

  /**
   * 移除事件监听
   * @param {string} event - 事件类型
   * @param {Function} callback - 回调函数
   */
  off(event, callback) {
    const cbs = this.listeners.get(event)
    if (cbs) {
      cbs.delete(callback)
    }
  }

  /** 绑定 WebSocket 事件处理 */
  _bindEvents() {
    // #ifdef H5
    this.ws.onopen = () => this._onOpen()
    this.ws.onclose = (e) => this._onClose(e)
    this.ws.onerror = (e) => this._onError(e)
    this.ws.onmessage = (e) => this._onMessage(e.data)
    // #endif

    // #ifndef H5
    uni.onSocketOpen(() => this._onOpen())
    uni.onSocketClose((e) => this._onClose(e))
    uni.onSocketError((e) => this._onError(e))
    uni.onSocketMessage((e) => this._onMessage(e.data))
    // #endif
  }

  _onOpen() {
    console.log('[WS] 连接成功')
    this.reconnectAttempts = 0
    this._startHeartbeat()
    this._emit('_connected')
  }

  _onClose(e) {
    console.log('[WS] 连接关闭', e)
    this._clearTimers()
    this._emit('_disconnected')

    if (!this.isManualClose) {
      this._scheduleReconnect()
    }
  }

  _onError(e) {
    console.error('[WS] 连接错误', e)
  }

  _onMessage(data) {
    try {
      const msg = JSON.parse(data)
      this._emit(msg.event, msg)
    } catch (e) {
      console.warn('[WS] 消息解析失败', data)
    }
  }

  /** 分发事件给注册的监听器 */
  _emit(event, data) {
    const cbs = this.listeners.get(event)
    if (cbs) {
      cbs.forEach(cb => {
        try {
          cb(data)
        } catch (e) {
          console.error(`[WS] 事件处理器异常: ${event}`, e)
        }
      })
    }
  }

  /** 启动心跳 */
  _startHeartbeat() {
    this._clearTimers()
    this.heartbeatTimer = setInterval(() => {
      this.send('heartbeat')
    }, HEARTBEAT_INTERVAL)
  }

  /** 调度重连（指数退避） */
  _scheduleReconnect() {
    if (this.reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      console.error('[WS] 超过最大重连次数')
      return
    }

    const delay = Math.min(
      RECONNECT_BASE_DELAY * Math.pow(2, this.reconnectAttempts),
      RECONNECT_MAX_DELAY
    )

    console.log(`[WS] ${delay}ms 后重连 (第 ${this.reconnectAttempts + 1} 次)`)
    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempts++
      this.connect()
    }, delay)
  }

  _clearTimers() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }
}

/** 全局单例 */
const wsService = new WebSocketService()
export default wsService
