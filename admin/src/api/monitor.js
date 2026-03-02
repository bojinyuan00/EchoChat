/**
 * 在线监控 API 模块
 *
 * 对应后端路由：/api/v1/admin/online/*
 * 所有接口需要 JWT + admin 角色权限
 */
import request from '@/utils/request'

/**
 * 获取在线用户列表
 * @returns {Promise<{data: Array<{user_id: number, username: string}>}>}
 */
export function getOnlineUsers() {
  return request({
    url: '/api/v1/admin/online/users',
    method: 'get'
  })
}

/**
 * 获取在线用户数量
 * @returns {Promise<{data: {count: number}}>}
 */
export function getOnlineCount() {
  return request({
    url: '/api/v1/admin/online/count',
    method: 'get'
  })
}
