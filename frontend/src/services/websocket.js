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

// WebSocket 基础地址推导：
// - 若 BASE_URL 非空（生产 CDN 直连场景），将其协议从 http(s) 替换成 ws(s)
// - 若 BASE_URL 为空（开发 vite proxy / 生产同源 Nginx），用当前页面 origin 推导同源 ws(s)
// 这样 WebSocket 天然随页面协议走：HTTPS 页面用 wss，HTTP 页面用 ws，避免混合内容被浏览器阻断
const resolveWsBase = () => {
  if (BASE_URL) return BASE_URL.replace(/^http/, 'ws')
  // #ifdef H5
  if (typeof window !== 'undefined' && window.location) {
    const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${wsProto}//${window.location.host}`
  }
  // #endif
  return ''
}
const WS_BASE = resolveWsBase()
const HEARTBEAT_INTERVAL = 30000
const RECONNECT_BASE_DELAY = 1000
const RECONNECT_MAX_DELAY = 30000
const MAX_RECONNECT_ATTEMPTS = Infinity
const ACK_DEFAULT_TIMEOUT_MS = 10000
/** 异常重连检测：10s 窗口内 >=3 次断连即认为异常抖动（常见于同账号多 tab 互踢） */
const FLAPPING_WINDOW_MS = 10000
const FLAPPING_THRESHOLD = 3
/** 两次 flapping 警告之间最小间隔，避免持续抖动时每秒触发 */
const FLAPPING_NOTIFY_COOLDOWN_MS = 15000

class WebSocketService {
  constructor() {
    this.ws = null
    this.listeners = new Map()
    this.heartbeatTimer = null
    this.reconnectTimer = null
    this.reconnectAttempts = 0
    this.isManualClose = false
    this.seq = 0
    /**
     * 等待 ACK 的请求索引：Map<seq, { resolve, reject, timer, event }>
     * Task 9 引入，为 meeting.* 事件提供 Promise 化 ACK 等待
     */
    this.pendingAcks = new Map()
    /** 近期断连时间戳数组，用于 flapping 检测 */
    this.recentCloseAt = []
    /** 上次 flapping 警告时间，用于冷却 */
    this.lastFlappingNotifyAt = 0
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
   * 发送消息并等待服务端 ACK（Task 9 引入）
   *
   * 服务端对 C→S 事件以相同 event 名 + 对应 seq 返回 ACK 报文：
   *   { event, seq, data: { code, message, data? } }
   *
   * 约定：
   * - code === 0 表示成功，Promise resolve 为 data.data（业务 payload）
   * - code !== 0 表示业务失败，Promise reject 为 { code, message }
   * - 超时 reject 为 { code: -1, message: 'ACK 超时' }
   * - 连接未就绪 reject 为 { code: -1, message: '连接未就绪' }
   *
   * 典型用法：
   *   const info = await wsService.sendWithAck('meeting.transport.create', { room_code, direction: 'send' })
   *
   * @param {string} event - 事件名
   * @param {Object} [data] - 业务 payload
   * @param {number} [timeoutMS=10000] - 超时时间
   * @returns {Promise<any>} ACK 的 data 字段
   */
  sendWithAck(event, data = {}, timeoutMS = ACK_DEFAULT_TIMEOUT_MS) {
    return new Promise((resolve, reject) => {
      const seq = this.send(event, data)
      if (seq === -1) {
        reject({ code: -1, message: '连接未就绪' })
        return
      }
      const timer = setTimeout(() => {
        this.pendingAcks.delete(seq)
        reject({ code: -1, message: `ACK 超时 (${event})` })
      }, timeoutMS)
      this.pendingAcks.set(seq, { resolve, reject, timer, event })
    })
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
    this._rejectAllPendingAcks('连接已断开')
    this._emit('_disconnected', e)

    if (!this.isManualClose) {
      this._recordCloseForFlapping()
      this._scheduleReconnect()
    }
  }

  /**
   * 记录一次非主动断连，并在短窗口内多次断连时触发 _flapping 事件
   * 典型场景：同账号多端登录互踢导致 WS 每秒被后端关闭
   */
  _recordCloseForFlapping() {
    const now = Date.now()
    this.recentCloseAt.push(now)
    // 丢掉窗口外的历史记录
    this.recentCloseAt = this.recentCloseAt.filter((t) => now - t <= FLAPPING_WINDOW_MS)
    if (this.recentCloseAt.length < FLAPPING_THRESHOLD) return
    if (now - this.lastFlappingNotifyAt < FLAPPING_NOTIFY_COOLDOWN_MS) return
    this.lastFlappingNotifyAt = now
    const count = this.recentCloseAt.length
    console.warn(`[WS] 检测到异常重连抖动：${FLAPPING_WINDOW_MS / 1000}s 内断开 ${count} 次，可能同账号在其他 tab/设备登录`)
    this._emit('_flapping', { count, windowMs: FLAPPING_WINDOW_MS })
  }

  /**
   * 连接断开时统一拒绝所有未完成的 ACK 等待（Task 9）
   * 避免业务层 await sendWithAck() 永久挂起
   */
  _rejectAllPendingAcks(reason) {
    if (this.pendingAcks.size === 0) return
    this.pendingAcks.forEach((pending, seq) => {
      clearTimeout(pending.timer)
      pending.reject({ code: -1, message: reason, event: pending.event })
    })
    this.pendingAcks.clear()
  }

  _onError(e) {
    console.error('[WS] 连接错误', e)
  }

  _onMessage(data) {
    try {
      const msg = JSON.parse(data)
      // Task 9：ACK 报文（event 以 ".ack" 结尾）会先走 pendingAcks 处理
      // 后端 ws.NewResponse 固定在原 event 后追加 ".ack"
      //
      // 修复（2026-04-23）：历史实现在命中 _handleAck 后直接 return，
      // 导致通过 on('xxx.ack') 订阅 ACK 事件的 listener（chat store 的
      // _onSendACK / _onReadACK）永远收不到回调，表现为
      // "发送方消息一直卡在 _sending=true，气泡下方不显示 已读/未读"。
      // 两条通路互不冲突：sendWithAck() 按 seq 命中 pending；
      // on('xxx.ack') 按 event 命中 listener。此处改为并行分发。
      if (typeof msg.event === 'string' && msg.event.endsWith('.ack')) {
        this._handleAck(msg)
      }
      this._emit(msg.event, msg)
    } catch (e) {
      console.warn('[WS] 消息解析失败', data)
    }
  }

  /**
   * 处理 ACK 报文，触发 pendingAcks 中对应 seq 的 resolve/reject
   * @param {Object} msg - { event: 'xxx.ack', seq, code, message, data? }
   */
  _handleAck(msg) {
    const pending = this.pendingAcks.get(msg.seq)
    if (!pending) {
      return
    }
    this.pendingAcks.delete(msg.seq)
    clearTimeout(pending.timer)
    if (msg.code === 0) {
      pending.resolve(msg.data)
    } else {
      pending.reject({ code: msg.code, message: msg.message || 'WS 请求失败', event: pending.event })
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
