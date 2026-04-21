/**
 * 通知中心 Store
 *
 * 统一管理：
 * - 通知列表（按分类 Tab 分页缓存，游标分页）
 * - 未读数统计（总数 + 分类统计）
 * - WebSocket 实时事件：notify.new（新通知）/ notify.unread.total（断线补偿）
 *
 * 对应后端 API：/api/v1/notifications/*
 * 对应 WS 事件：notify.new、notify.unread.total
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import notifyApi from '@/api/notify'
import wsService from '@/services/websocket'

/** 前端 5 个 Tab 分类常量 */
export const NOTIFY_CATEGORY_ALL = 'all'
export const NOTIFY_CATEGORY_FRIEND = 'friend'
export const NOTIFY_CATEGORY_GROUP = 'group'
export const NOTIFY_CATEGORY_MEETING = 'meeting'
export const NOTIFY_CATEGORY_SYSTEM = 'system'

const CATEGORY_LIST = [
  NOTIFY_CATEGORY_ALL,
  NOTIFY_CATEGORY_FRIEND,
  NOTIFY_CATEGORY_GROUP,
  NOTIFY_CATEGORY_MEETING,
  NOTIFY_CATEGORY_SYSTEM
]

const DEFAULT_PAGE_SIZE = 20

/**
 * 创建空的分类分页状态
 */
const createCategoryState = () => ({
  list: [],
  hasMore: true,
  loading: false
})

export const useNotifyStore = defineStore('notify', () => {
  /** 按分类存储通知列表（Map 结构 category -> { list, hasMore, loading }） */
  const byCategory = ref({
    [NOTIFY_CATEGORY_ALL]: createCategoryState(),
    [NOTIFY_CATEGORY_FRIEND]: createCategoryState(),
    [NOTIFY_CATEGORY_GROUP]: createCategoryState(),
    [NOTIFY_CATEGORY_MEETING]: createCategoryState(),
    [NOTIFY_CATEGORY_SYSTEM]: createCategoryState()
  })

  /** 未读总数 */
  const unreadTotal = ref(0)

  /** 未读分类统计 { friend, group, meeting, system } */
  const unreadByCategory = ref({
    [NOTIFY_CATEGORY_FRIEND]: 0,
    [NOTIFY_CATEGORY_GROUP]: 0,
    [NOTIFY_CATEGORY_MEETING]: 0,
    [NOTIFY_CATEGORY_SYSTEM]: 0
  })

  /** 当前激活的 Tab（页面用于驱动切换） */
  const activeCategory = ref(NOTIFY_CATEGORY_ALL)

  /** Tab 角标：未读总数 */
  const badgeTotal = computed(() => unreadTotal.value)

  /** 获取指定分类的状态（安全读取） */
  const getCategoryState = (category) => {
    const key = CATEGORY_LIST.includes(category) ? category : NOTIFY_CATEGORY_ALL
    if (!byCategory.value[key]) {
      byCategory.value[key] = createCategoryState()
    }
    return byCategory.value[key]
  }

  // ==================== Actions ====================

  /**
   * 拉取未读数统计
   * 场景：页面初始化、WS 断线补偿 notify.unread.total 事件、明确需要刷新时
   */
  const fetchUnreadCount = async () => {
    try {
      const res = await notifyApi.getUnreadCount()
      const data = res.data || {}
      unreadTotal.value = data.total || 0
      const byCat = data.by_category || {}
      unreadByCategory.value = {
        [NOTIFY_CATEGORY_FRIEND]: byCat[NOTIFY_CATEGORY_FRIEND] || 0,
        [NOTIFY_CATEGORY_GROUP]: byCat[NOTIFY_CATEGORY_GROUP] || 0,
        [NOTIFY_CATEGORY_MEETING]: byCat[NOTIFY_CATEGORY_MEETING] || 0,
        [NOTIFY_CATEGORY_SYSTEM]: byCat[NOTIFY_CATEGORY_SYSTEM] || 0
      }
    } catch (e) {
      console.warn('[notify] 获取未读数失败', e)
    }
  }

  /**
   * 加载指定分类的通知列表（首屏刷新）
   * @param {string} category - 分类
   * @param {Object} [opts]
   * @param {boolean} [opts.onlyUnread] - 是否仅看未读
   */
  const fetchList = async (category = NOTIFY_CATEGORY_ALL, opts = {}) => {
    const state = getCategoryState(category)
    state.loading = true
    try {
      const params = {
        category,
        limit: DEFAULT_PAGE_SIZE
      }
      if (opts.onlyUnread === true) params.is_read = false
      const res = await notifyApi.getNotifications(params)
      const data = res.data || {}
      state.list = data.list || []
      state.hasMore = data.has_more === true
    } finally {
      state.loading = false
    }
    return state.list
  }

  /**
   * 加载下一页（游标分页，基于当前列表最后一条的 ID）
   * @param {string} category
   * @param {Object} [opts]
   * @param {boolean} [opts.onlyUnread]
   */
  const loadMore = async (category = NOTIFY_CATEGORY_ALL, opts = {}) => {
    const state = getCategoryState(category)
    if (!state.hasMore || state.loading) return []
    const last = state.list[state.list.length - 1]
    if (!last) return fetchList(category, opts)

    state.loading = true
    try {
      const params = {
        category,
        limit: DEFAULT_PAGE_SIZE,
        before_id: last.id
      }
      if (opts.onlyUnread === true) params.is_read = false
      const res = await notifyApi.getNotifications(params)
      const data = res.data || {}
      const newList = data.list || []
      state.list = state.list.concat(newList)
      state.hasMore = data.has_more === true
      return newList
    } finally {
      state.loading = false
    }
  }

  /**
   * 标记单条已读
   * 后端成功后本地更新列表状态 + 扣减对应分类未读数
   * @param {number} id - 通知 ID
   */
  const markRead = async (id) => {
    const targetCategory = _findCategoryById(id)
    const target = _findNotifyById(id)
    // 先快照原始未读状态，避免 _patchAll 改写 target.is_read 后再判断失效
    const wasUnread = !!target && !target.is_read
    if (target && target.is_read) return

    await notifyApi.markRead(id)

    // 更新所有缓存列表中对应通知的状态
    _patchAll(id, { is_read: true, read_at: _nowString() })

    if (wasUnread) {
      if (targetCategory && unreadByCategory.value[targetCategory] > 0) {
        unreadByCategory.value[targetCategory]--
      }
      if (unreadTotal.value > 0) unreadTotal.value--
    }
  }

  /**
   * 标记全部（或某分类）已读
   * @param {string} [category] - 分类：friend/group/meeting/system，不传则全部
   */
  const markAllRead = async (category) => {
    await notifyApi.markAllRead(category)

    // 本地更新所有缓存列表
    Object.keys(byCategory.value).forEach(cat => {
      if (!category || cat === category || cat === NOTIFY_CATEGORY_ALL) {
        byCategory.value[cat].list.forEach(n => {
          if (!category || n.category === category) {
            if (!n.is_read) {
              n.is_read = true
              n.read_at = _nowString()
            }
          }
        })
      }
    })

    // 重新拉取未读数，保证后端权威
    await fetchUnreadCount()
  }

  /**
   * 切换激活 Tab
   * @param {string} category
   */
  const setActiveCategory = (category) => {
    if (CATEGORY_LIST.includes(category)) {
      activeCategory.value = category
    }
  }

  /**
   * 重置 store（登出调用）
   */
  const reset = () => {
    Object.keys(byCategory.value).forEach(cat => {
      byCategory.value[cat] = createCategoryState()
    })
    unreadTotal.value = 0
    unreadByCategory.value = {
      [NOTIFY_CATEGORY_FRIEND]: 0,
      [NOTIFY_CATEGORY_GROUP]: 0,
      [NOTIFY_CATEGORY_MEETING]: 0,
      [NOTIFY_CATEGORY_SYSTEM]: 0
    }
    activeCategory.value = NOTIFY_CATEGORY_ALL
  }

  // ==================== WebSocket 事件 ====================

  /** 防止 WebSocket 事件监听重复注册 */
  let _wsInitialized = false

  /**
   * 处理 notify.new 事件：新通知到达
   * 行为：
   * 1) 插入到对应分类列表和 all 分类列表顶部
   * 2) 未读总数 +1、对应分类 +1
   */
  const _onNotifyNew = (msg) => {
    if (!msg || !msg.data) return
    const notify = msg.data

    const targetCat = notify.category
    if (targetCat && byCategory.value[targetCat]) {
      byCategory.value[targetCat].list.unshift(notify)
    }
    byCategory.value[NOTIFY_CATEGORY_ALL].list.unshift(notify)

    if (!notify.is_read) {
      unreadTotal.value++
      if (targetCat && targetCat in unreadByCategory.value) {
        unreadByCategory.value[targetCat]++
      }
    }
  }

  /**
   * 处理 notify.unread.total 事件：断线补偿
   * 行为：以权威推送值覆盖本地缓存的未读数
   */
  const _onUnreadTotal = (msg) => {
    if (!msg || !msg.data) return
    const data = msg.data
    if (typeof data.total === 'number') unreadTotal.value = data.total
    if (data.by_category) {
      unreadByCategory.value = {
        [NOTIFY_CATEGORY_FRIEND]: data.by_category[NOTIFY_CATEGORY_FRIEND] || 0,
        [NOTIFY_CATEGORY_GROUP]: data.by_category[NOTIFY_CATEGORY_GROUP] || 0,
        [NOTIFY_CATEGORY_MEETING]: data.by_category[NOTIFY_CATEGORY_MEETING] || 0,
        [NOTIFY_CATEGORY_SYSTEM]: data.by_category[NOTIFY_CATEGORY_SYSTEM] || 0
      }
    }
  }

  /** 初始化 WebSocket 事件监听（幂等，多次调用只注册一次） */
  const initWsListeners = () => {
    if (_wsInitialized) return
    _wsInitialized = true
    wsService.on('notify.new', _onNotifyNew)
    wsService.on('notify.unread.total', _onUnreadTotal)
  }

  // ==================== 内部工具 ====================

  /** 在所有缓存列表中查找指定 ID 的通知 */
  const _findNotifyById = (id) => {
    for (const cat of CATEGORY_LIST) {
      const state = byCategory.value[cat]
      if (!state) continue
      const found = state.list.find(n => n.id === id)
      if (found) return found
    }
    return null
  }

  /** 推断某条通知所属的分类（根据缓存查找） */
  const _findCategoryById = (id) => {
    for (const cat of CATEGORY_LIST) {
      if (cat === NOTIFY_CATEGORY_ALL) continue
      const state = byCategory.value[cat]
      if (!state) continue
      if (state.list.some(n => n.id === id)) return cat
    }
    // 兜底：从 all 列表查找 category 字段
    const inAll = byCategory.value[NOTIFY_CATEGORY_ALL].list.find(n => n.id === id)
    return inAll ? inAll.category : null
  }

  /** 在所有分类列表中 patch 对应 ID 的通知字段 */
  const _patchAll = (id, patch) => {
    CATEGORY_LIST.forEach(cat => {
      const state = byCategory.value[cat]
      if (!state) return
      const target = state.list.find(n => n.id === id)
      if (target) Object.assign(target, patch)
    })
  }

  const _nowString = () => {
    const d = new Date()
    const pad = (n) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }

  return {
    // state
    byCategory,
    unreadTotal,
    unreadByCategory,
    activeCategory,
    // computed
    badgeTotal,
    // actions
    getCategoryState,
    fetchUnreadCount,
    fetchList,
    loadMore,
    markRead,
    markAllRead,
    setActiveCategory,
    reset,
    initWsListeners
  }
})
