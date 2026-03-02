/**
 * WebSocket 连接状态 Store
 *
 * 管理 WebSocket 连接生命周期，提供：
 * - 连接/断开操作
 * - 连接状态跟踪（connected/disconnected）
 * - 与 user store 联动（登录时自动连接，登出时断开）
 *
 * 对应服务层：services/websocket.js
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import wsService from '@/services/websocket'

export const useWebSocketStore = defineStore('websocket', () => {
  /** 连接状态 */
  const connected = ref(false)

  /** 是否已连接 */
  const isConnected = computed(() => connected.value)

  /** 防止 _connected/_disconnected 监听器重复注册 */
  let _initialized = false

  /**
   * 初始化 WebSocket 连接
   * @param {string} [token] - JWT Token
   */
  const connect = (token) => {
    if (!_initialized) {
      wsService.on('_connected', () => {
        connected.value = true
      })
      wsService.on('_disconnected', () => {
        connected.value = false
      })
      _initialized = true
    }
    wsService.connect(token)
  }

  /** 断开 WebSocket 连接 */
  const disconnect = () => {
    wsService.disconnect()
    connected.value = false
  }

  /**
   * 注册事件监听（代理 wsService）
   * @param {string} event - 事件类型
   * @param {Function} callback - 回调函数
   */
  const on = (event, callback) => {
    wsService.on(event, callback)
  }

  /**
   * 移除事件监听（代理 wsService）
   * @param {string} event - 事件类型
   * @param {Function} callback - 回调函数
   */
  const off = (event, callback) => {
    wsService.off(event, callback)
  }

  return {
    connected,
    isConnected,
    connect,
    disconnect,
    on,
    off
  }
})
