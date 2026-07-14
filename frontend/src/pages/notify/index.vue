<!--
  通知中心主页
  
  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B / Muted #94A3B8 / Danger #EF4444
  
  功能：
  - 顶部导航：返回 + 标题 + 全部已读按钮
  - 5 个分类 Tab：全部 / 好友 / 群聊 / 会议 / 系统（未读数角标）
  - 下拉/上拉刷新（游标分页）
  - 点击通知 → 标记已读 + deep-link 跳转
  - 群邀请/入群申请通知：支持内联“接受/拒绝”
  - 空状态、骨架屏、实时 WS 事件接入
-->
<template>
  <view class="page-wrapper">
    <!-- 顶部栏 -->
    <view class="header">
      <view class="back-btn" @tap="goBack">
        <text class="back-icon">&lsaquo;</text>
      </view>
      <text class="header-title">通知中心</text>
      <view class="mark-all-btn" @tap="handleMarkAll">
        <text class="mark-all-text">全部已读</text>
      </view>
    </view>

    <!-- 分类 Tab -->
    <scroll-view
      scroll-x
      class="tab-scroll"
      :show-scrollbar="false"
    >
      <view class="tab-list">
        <view
          v-for="tab in tabs"
          :key="tab.key"
          class="tab-item"
          :class="{ 'tab-item--active': activeCategory === tab.key }"
          @tap="switchTab(tab.key)"
        >
          <text class="tab-label">{{ tab.label }}</text>
          <view v-if="tabBadge(tab.key) > 0" class="tab-badge">
            {{ tabBadge(tab.key) > 99 ? '99+' : tabBadge(tab.key) }}
          </view>
        </view>
      </view>
    </scroll-view>

    <!-- 骨架屏 -->
    <view v-if="isInitLoading" class="skeleton-list">
      <view v-for="i in 4" :key="i" class="skeleton-item">
        <view class="skeleton-icon"></view>
        <view class="skeleton-body">
          <view class="skeleton-line skeleton-line--title"></view>
          <view class="skeleton-line skeleton-line--content"></view>
        </view>
      </view>
    </view>

    <!-- 列表 -->
    <scroll-view
      v-else
      scroll-y
      class="notify-list"
      :refresher-enabled="true"
      :refresher-triggered="refreshing"
      @refresherrefresh="onRefresh"
      @scrolltolower="onLoadMore"
    >
      <view v-if="currentList.length === 0" class="empty-state">
        <text class="empty-icon">&#128276;</text>
        <text class="empty-title">暂无通知</text>
        <text class="empty-desc">你暂时没有新的消息通知</text>
      </view>

      <NotifyItem
        v-for="item in currentList"
        :key="item.id"
        :notify="item"
        @item-tap="handleNotifyTap"
        @item-accept="handleAccept"
        @item-reject="handleReject"
      />

      <view v-if="currentList.length > 0" class="footer-hint">
        <text v-if="currentState.loading">加载中...</text>
        <text v-else-if="!currentState.hasMore">已经到底啦</text>
      </view>
    </scroll-view>
  </view>
</template>

<script setup>
/**
 * 通知中心页面
 * 状态集中在 useNotifyStore 中；本组件只负责 UI 编排与事件绑定
 */
import { ref, computed, onMounted } from 'vue'
import { onShow, onUnload } from '@dcloudio/uni-app'
import { useNotifyStore } from '@/store/notify'
import {
  NOTIFY_CATEGORY_TABS,
  NOTIFY_CATEGORY_ALL,
  NOTIFY_CATEGORY_FRIEND,
  NOTIFY_CATEGORY_GROUP,
  NOTIFY_CATEGORY_MEETING,
  NOTIFY_CATEGORY_SYSTEM,
  NOTIFY_TYPE_FRIEND_REQUEST,
  NOTIFY_TYPE_GROUP_INVITE,
  NOTIFY_TYPE_GROUP_JOIN_REQUEST,
  NOTIFY_TYPE_GROUP_JOIN_APPROVED,
  NOTIFY_TYPE_MEETING_INVITE
} from '@/constants/notify'
import NotifyItem from '@/components/notify/NotifyItem.vue'
import groupApi from '@/api/group'

const notifyStore = useNotifyStore()
const tabs = NOTIFY_CATEGORY_TABS

const refreshing = ref(false)
const isInitLoading = ref(true)

const activeCategory = computed(() => notifyStore.activeCategory)
const currentState = computed(() => notifyStore.getCategoryState(activeCategory.value))
const currentList = computed(() => currentState.value.list)

/** 计算 Tab 徽标（友 / 群 / 会议 / 系统 基于 unreadByCategory，全部 基于 unreadTotal） */
const tabBadge = (key) => {
  if (key === NOTIFY_CATEGORY_ALL) return notifyStore.unreadTotal
  return notifyStore.unreadByCategory[key] || 0
}

/** 初始化：注册 WS + 拉未读数 + 首屏列表 */
onMounted(async () => {
  notifyStore.initWsListeners()
  try {
    await Promise.all([
      notifyStore.fetchUnreadCount(),
      notifyStore.fetchList(activeCategory.value)
    ])
  } catch (e) {
    console.error('[notify] 初始化失败', e)
  } finally {
    isInitLoading.value = false
  }
})

/** 每次页面重新显示：刷新当前 Tab + 未读数（保证从其他页面返回时数据最新） */
onShow(() => {
  if (isInitLoading.value) return
  notifyStore.fetchUnreadCount().catch(() => {})
  notifyStore.fetchList(activeCategory.value).catch(() => {})
})

onUnload(() => {
  // 不重置 store，保留缓存以便下次进入直接看到
})

/** 切换 Tab，若缓存为空则拉取 */
const switchTab = async (key) => {
  if (activeCategory.value === key) return
  notifyStore.setActiveCategory(key)
  const state = notifyStore.getCategoryState(key)
  if (state.list.length === 0) {
    try {
      await notifyStore.fetchList(key)
    } catch (e) {
      console.error('[notify] 切 Tab 加载失败', e)
    }
  }
}

/** 下拉刷新当前 Tab */
const onRefresh = async () => {
  refreshing.value = true
  try {
    await Promise.all([
      notifyStore.fetchUnreadCount(),
      notifyStore.fetchList(activeCategory.value)
    ])
  } catch (e) {
    console.error('[notify] 刷新失败', e)
  } finally {
    refreshing.value = false
  }
}

/** 触底加载更多 */
const onLoadMore = async () => {
  try {
    await notifyStore.loadMore(activeCategory.value)
  } catch (e) {
    console.error('[notify] 加载更多失败', e)
  }
}

/** 标记全部已读（当前 Tab 为分类则只清该类） */
const handleMarkAll = async () => {
  const category = activeCategory.value === NOTIFY_CATEGORY_ALL ? undefined : activeCategory.value
  try {
    await notifyStore.markAllRead(category)
    uni.showToast({ title: '已全部标记为已读', icon: 'none' })
  } catch (e) {
    uni.showToast({ title: '操作失败，请重试', icon: 'none' })
  }
}

/**
 * 点击通知：标记已读 + deep-link 跳转
 * 根据 target_type + target_id 或 type 语义决定跳转路径
 */
const handleNotifyTap = async (notify) => {
  if (!notify || !notify.id) {
    console.warn('[notify] handleNotifyTap 收到无效 payload', notify)
    return
  }
  if (!notify.is_read) {
    try {
      await notifyStore.markRead(notify.id)
    } catch (e) {
      console.warn('[notify] 标记已读失败', e)
    }
  }
  _navigateByNotify(notify)
}

/**
 * 内联"接受"按钮
 * - group_invite / group_join_request：沿用原逻辑
 * - meeting_invite：立即加入，跳预览页（mode=join&code=...）
 */
const handleAccept = async (notify) => {
  try {
    if (notify.type === NOTIFY_TYPE_GROUP_JOIN_REQUEST) {
      const extra = _parseExtra(notify.extra)
      if (extra && extra.group_id && extra.request_id) {
        await groupApi.reviewJoinRequest(extra.group_id, extra.request_id, 'approve')
        uni.showToast({ title: '已批准入群申请', icon: 'success' })
      }
    } else if (notify.type === NOTIFY_TYPE_GROUP_INVITE) {
      // 群邀请采用 pending 流程：调用 accept 接口后才真正入群
      const extra = _parseExtra(notify.extra)
      if (!extra || !extra.request_id) {
        uni.showToast({ title: '邀请信息缺失', icon: 'none' })
        return
      }
      await groupApi.acceptInvitation(extra.request_id)
      uni.showToast({ title: '已加入群聊', icon: 'success' })
      _navigateToGroup(extra)
    } else if (notify.type === NOTIFY_TYPE_MEETING_INVITE) {
      const extra = _parseExtra(notify.extra)
      if (!_navigateToMeetingInvite(extra)) {
        uni.showToast({ title: '邀请信息已失效', icon: 'none' })
      }
    }
    await notifyStore.markRead(notify.id).catch(() => {})
  } catch (e) {
    console.error('[notify] 接受失败', e)
    uni.showToast({ title: e && e.message ? e.message : '操作失败', icon: 'none' })
  }
}

/**
 * 内联"拒绝/稍后"按钮
 * - group_invite / group_join_request：沿用原逻辑
 * - meeting_invite：仅标记已读，不弹 toast（"稍后"是轻量操作）
 */
const handleReject = async (notify) => {
  try {
    if (notify.type === NOTIFY_TYPE_GROUP_JOIN_REQUEST) {
      const extra = _parseExtra(notify.extra)
      if (extra && extra.group_id && extra.request_id) {
        await groupApi.reviewJoinRequest(extra.group_id, extra.request_id, 'reject')
        uni.showToast({ title: '已拒绝入群申请', icon: 'success' })
      }
    } else if (notify.type === NOTIFY_TYPE_GROUP_INVITE) {
      // 群邀请采用 pending 流程：调用 reject 接口后该邀请被置为 rejected，不产生群成员变动
      const extra = _parseExtra(notify.extra)
      if (!extra || !extra.request_id) {
        uni.showToast({ title: '邀请信息缺失', icon: 'none' })
        return
      }
      await groupApi.rejectInvitation(extra.request_id)
      uni.showToast({ title: '已拒绝邀请', icon: 'success' })
    }
    await notifyStore.markRead(notify.id).catch(() => {})
  } catch (e) {
    console.error('[notify] 拒绝失败', e)
    uni.showToast({ title: e && e.message ? e.message : '操作失败', icon: 'none' })
  }
}

/** 根据通知的业务对象进行跳转 */
const _navigateByNotify = (notify) => {
  const extra = _parseExtra(notify.extra)
  switch (notify.type) {
    case NOTIFY_TYPE_FRIEND_REQUEST:
      uni.navigateTo({ url: '/pages/contact/request' })
      return
    case NOTIFY_TYPE_GROUP_JOIN_REQUEST:
      if (extra && extra.group_id) {
        uni.navigateTo({ url: `/pages/group/join-requests?groupId=${extra.group_id}` })
        return
      }
      break
    case NOTIFY_TYPE_GROUP_INVITE:
    case NOTIFY_TYPE_GROUP_JOIN_APPROVED:
      _navigateToGroup(extra)
      return
    case NOTIFY_TYPE_MEETING_INVITE:
      if (_navigateToMeetingInvite(extra)) return
      // 过期或 extra 缺失时给用户提示
      uni.showToast({ title: '邀请已过期或信息缺失', icon: 'none' })
      return
    default:
      break
  }
  // 回退：基于 target_type
  if (notify.target_type === 'group' && notify.target_id) {
    _navigateToGroup({ group_id: notify.target_id, conversation_id: extra && extra.conversation_id })
  } else if (notify.target_type === 'user' && notify.target_id) {
    uni.navigateTo({ url: `/pages/contact/detail?userId=${notify.target_id}` })
  }
}

/** 跳转到群聊会话（优先使用 conversation_id，否则群设置页） */
const _navigateToGroup = (extra) => {
  if (!extra) return
  if (extra.conversation_id) {
    uni.navigateTo({ url: `/pages/group/conversation?conversationId=${extra.conversation_id}` })
  } else if (extra.group_id) {
    uni.navigateTo({ url: `/pages/group/settings?groupId=${extra.group_id}` })
  }
}

/**
 * 跳转到会议预览页（meeting_invite 卡片专用）
 *
 * 路由协议：/pages/meeting/preview?mode=join&code=xxx
 *   - 预览页负责让用户确认设备、输入密码（若 has_password），再调 joinAndEnter
 *   - invite_token 当前不在 URL 透传：Redis 写入 600s，Token 保留冗余兑换
 *     后端 RedeemInviteToken 目前在独立接口；MVP 阶段仅凭 room_code 加入即可
 *
 * @returns {boolean} 是否成功发起跳转（用于调用方判断是否需要 fallback 提示）
 */
const _navigateToMeetingInvite = (extra) => {
  if (!extra || !extra.room_code) return false
  if (extra.expired_at && Number(extra.expired_at) * 1000 < Date.now()) {
    return false
  }
  uni.navigateTo({ url: `/pages/meeting/preview?mode=join&code=${encodeURIComponent(extra.room_code)}` })
  return true
}

/** 解析 extra（后端以 JSON 字符串存储） */
const _parseExtra = (extra) => {
  if (!extra) return null
  if (typeof extra === 'object') return extra
  try {
    return JSON.parse(extra)
  } catch (e) {
    return null
  }
}

/**
 * 返回上一页
 * - 正常路径：从 profile/其他页 → 通知中心 → navigateBack 到上一页
 * - 兜底路径：直接 URL 访问通知中心导致页面栈为空时，降级到 tabBar 页
 *   注意：profile 与 chat 都是 tabBar 页，必须使用 switchTab（不能用 navigateTo/reLaunch）。
 *   reLaunch 在 H5 下跳 tabBar 页面可能导致 tabBar UI 不渲染，故改用 switchTab。
 */
const goBack = () => {
  uni.navigateBack({
    delta: 1,
    fail: () => {
      uni.switchTab({
        url: '/pages/profile/index',
        fail: () => {
          uni.switchTab({ url: '/pages/chat/index' })
        }
      })
    }
  })
}
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
}

/* ---- 顶部栏 ---- */
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20rpx 24rpx;
  padding-top: calc(var(--status-bar-height, 44px) + 20rpx);
  background-color: #FFFFFF;
  box-shadow: 0 1rpx 0 rgba(0, 0, 0, 0.04);
}

.back-btn {
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  cursor: pointer;
}

.back-btn:active {
  background-color: #F1F5F9;
}

.back-icon {
  font-size: 48rpx;
  color: #1E293B;
  line-height: 1;
}

.header-title {
  font-size: 34rpx;
  font-weight: 700;
  color: #1E293B;
}

.mark-all-btn {
  padding: 14rpx 20rpx;
  border-radius: 16rpx;
  cursor: pointer;
}

.mark-all-btn:active {
  background-color: #F1F5F9;
}

.mark-all-text {
  font-size: 26rpx;
  color: #2563EB;
  font-weight: 500;
}

/* ---- Tab ---- */
.tab-scroll {
  background-color: #FFFFFF;
  white-space: nowrap;
  border-bottom: 1rpx solid #F1F5F9;
}

.tab-list {
  display: inline-flex;
  padding: 8rpx 24rpx;
  gap: 8rpx;
}

.tab-item {
  position: relative;
  display: inline-flex;
  align-items: center;
  padding: 18rpx 28rpx;
  gap: 8rpx;
  border-radius: 100rpx;
  cursor: pointer;
  transition: background-color 200ms ease;
}

.tab-item--active {
  background-color: #EFF6FF;
}

.tab-label {
  font-size: 28rpx;
  color: #64748B;
}

.tab-item--active .tab-label {
  color: #2563EB;
  font-weight: 600;
}

.tab-badge {
  min-width: 32rpx;
  height: 32rpx;
  padding: 0 8rpx;
  border-radius: 16rpx;
  background-color: #EF4444;
  color: #FFFFFF;
  font-size: 20rpx;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ---- 列表 ---- */
.notify-list {
  height: calc(100vh - 200rpx);
}

/* ---- 骨架屏 ---- */
.skeleton-list {
  padding: 20rpx 0;
}

.skeleton-item {
  display: flex;
  gap: 20rpx;
  padding: 28rpx 32rpx;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #F1F5F9;
}

.skeleton-icon {
  width: 80rpx;
  height: 80rpx;
  border-radius: 16rpx;
  background-color: #E2E8F0;
  animation: pulse 1.4s ease-in-out infinite;
}

.skeleton-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12rpx;
  justify-content: center;
}

.skeleton-line {
  height: 24rpx;
  border-radius: 12rpx;
  background-color: #E2E8F0;
  animation: pulse 1.4s ease-in-out infinite;
}

.skeleton-line--title {
  width: 40%;
}

.skeleton-line--content {
  width: 80%;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* ---- 空状态 ---- */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 160rpx 40rpx;
  gap: 16rpx;
}

.empty-icon {
  font-size: 96rpx;
  opacity: 0.6;
}

.empty-title {
  font-size: 30rpx;
  color: #475569;
  font-weight: 600;
}

.empty-desc {
  font-size: 26rpx;
  color: #94A3B8;
  text-align: center;
}

/* ---- 底部提示 ---- */
.footer-hint {
  padding: 32rpx 0;
  text-align: center;
}

.footer-hint text {
  font-size: 24rpx;
  color: #94A3B8;
}
</style>
