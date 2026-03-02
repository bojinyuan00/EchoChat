/**
 * 用户搜索 API
 *
 * 对应后端路由：/api/v1/users/*
 */

import { get } from '@/utils/request'

/** 搜索用户（按用户名/昵称模糊匹配） */
const searchUsers = (keyword, page = 1, pageSize = 20) => {
  return get('/api/v1/users/search', { keyword, page, page_size: pageSize })
}

export default {
  searchUsers
}
