<!--
  联系人列表页（TabBar 页面）

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B / Muted #94A3B8
  ui-ux-pro-max 规范：v-for :key / 触摸目标 >= 88rpx / 骨架屏 loading / cursor-pointer
-->
<template>
  <view class="page-wrapper">
    <!-- 顶部栏 -->
    <view class="header">
      <text class="header-title">联系人</text>
      <view class="header-actions">
        <view class="action-btn" @tap="goToGroups">
          <text class="action-icon">&#9776;</text>
        </view>
        <view class="action-btn" @tap="goToSearch">
          <text class="action-icon">+</text>
        </view>
      </view>
    </view>

    <!-- 搜索栏 -->
    <view class="search-bar">
      <view class="search-input-wrap">
        <text class="search-icon">&#128269;</text>
        <input
          class="search-input"
          v-model="searchKeyword"
          placeholder="搜索好友"
          placeholder-style="color: #94A3B8"
          confirm-type="search"
        />
      </view>
    </view>

    <!-- 功能入口 -->
    <view class="entry-section">
      <view class="entry-item" @tap="goToRequests">
        <view class="entry-left">
          <view class="entry-icon entry-icon--request">
            <text class="entry-icon-text">&#9993;</text>
          </view>
          <text class="entry-label">好友申请</text>
        </view>
        <view class="entry-right">
          <view v-if="contactStore.pendingCount > 0" class="badge">
            {{ contactStore.pendingCount > 99 ? '99+' : contactStore.pendingCount }}
          </view>
          <text class="arrow">&#8250;</text>
        </view>
      </view>
      <view class="entry-item" @tap="goToBlacklist">
        <view class="entry-left">
          <view class="entry-icon entry-icon--block">
            <text class="entry-icon-text">&#8856;</text>
          </view>
          <text class="entry-label">黑名单</text>
        </view>
        <view class="entry-right">
          <text class="arrow">&#8250;</text>
        </view>
      </view>
    </view>

    <!-- 好友列表 -->
    <view class="section-header" v-if="!loading">
      <text class="section-title">好友列表</text>
      <text class="section-count">{{ filteredFriends.length }} 人</text>
    </view>

    <!-- 骨架屏 -->
    <view v-if="loading" class="skeleton-list">
      <view v-for="i in 5" :key="i" class="skeleton-item">
        <view class="skeleton-avatar"></view>
        <view class="skeleton-info">
          <view class="skeleton-line skeleton-line--name"></view>
          <view class="skeleton-line skeleton-line--sub"></view>
        </view>
      </view>
    </view>

    <!-- 好友列表 -->
    <view v-else-if="filteredFriends.length > 0" class="friend-list">
      <view
        v-for="friend in filteredFriends"
        :key="friend.user_id"
        class="friend-item"
        @tap="goToDetail(friend)"
      >
        <view class="avatar-wrap">
          <view class="avatar" :style="{ backgroundColor: getAvatarColor(friend.nickname || friend.username) }">
            <text class="avatar-text">{{ getInitial(friend.remark || friend.nickname || friend.username) }}</text>
          </view>
          <view v-if="friend.is_online" class="online-dot"></view>
        </view>
        <view class="friend-info">
          <text class="friend-name">{{ friend.remark || friend.nickname || friend.username }}</text>
          <text class="friend-status">{{ friend.is_online ? '在线' : '离线' }}</text>
        </view>
      </view>
    </view>

    <!-- 空状态 -->
    <view v-else class="empty-state">
      <text class="empty-title">暂无好友</text>
      <text class="empty-desc">点击右上角 "+" 搜索并添加好友</text>
    </view>

    <CustomTabBar :current="1" />
  </view>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { useContactStore } from '@/store/contact'
import { useWebSocketStore } from '@/store/websocket'
import { useUserStore } from '@/store/user'
import { getAvatarColor, getInitial } from '@/utils/avatar'
import CustomTabBar from '@/components/CustomTabBar.vue'

const contactStore = useContactStore()
const wsStore = useWebSocketStore()
const userStore = useUserStore()

const searchKeyword = ref('')
const loading = ref(true)

const filteredFriends = computed(() => {
  if (!searchKeyword.value) return contactStore.friendList
  const kw = searchKeyword.value.toLowerCase()
  return contactStore.friendList.filter(f =>
    (f.remark || f.nickname || f.username || '').toLowerCase().includes(kw)
  )
})

const refreshData = async () => {
  if (!userStore.isLoggedIn) return
  try {
    await Promise.all([
      contactStore.fetchFriends(),
      contactStore.fetchPendingRequests()
    ])
  } catch (e) {
    console.error('获取联系人数据失败', e)
  }
}

onMounted(async () => {
  if (userStore.isLoggedIn) {
    wsStore.connect()
    contactStore.initWsListeners()
    await refreshData()
  }
  loading.value = false
})

onShow(() => {
  if (!loading.value) {
    refreshData()
  }
})

const goToSearch = () => uni.navigateTo({ url: '/pages/contact/search' })
const goToRequests = () => uni.navigateTo({ url: '/pages/contact/request' })
const goToDetail = (friend) => uni.navigateTo({ url: `/pages/contact/detail?userId=${friend.user_id}` })
const goToGroups = () => uni.navigateTo({ url: '/pages/contact/groups' })
const goToBlacklist = () => uni.navigateTo({ url: '/pages/contact/blacklist' })
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
  padding-bottom: 120rpx;
}

/* 顶部栏 */
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24rpx 32rpx;
  padding-top: calc(var(--status-bar-height, 44px) + 20rpx);
  background-color: #FFFFFF;
}

.header-title {
  font-size: 36rpx;
  font-weight: 700;
  color: #1E293B;
}

.header-actions {
  display: flex;
  gap: 16rpx;
}

.action-btn {
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background-color: #F1F5F9;
  cursor: pointer;
  transition: background-color 200ms ease;
}

.action-btn:active {
  background-color: #E2E8F0;
}

.action-icon {
  font-size: 36rpx;
  color: #1E293B;
  line-height: 1;
}

/* 搜索栏 */
.search-bar {
  padding: 16rpx 32rpx;
  background-color: #FFFFFF;
}

.search-input-wrap {
  display: flex;
  align-items: center;
  background-color: #F1F5F9;
  border-radius: 16rpx;
  padding: 0 24rpx;
  height: 72rpx;
}

.search-icon {
  font-size: 28rpx;
  color: #94A3B8;
  margin-right: 12rpx;
}

.search-input {
  flex: 1;
  font-size: 28rpx;
  color: #1E293B;
  height: 72rpx;
}

/* 功能入口 */
.entry-section {
  margin-top: 16rpx;
  background-color: #FFFFFF;
}

.entry-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  cursor: pointer;
  transition: background-color 200ms ease;
}

.entry-item:active {
  background-color: #F8FAFC;
}

.entry-item:last-child {
  border-bottom: none;
}

.entry-left {
  display: flex;
  align-items: center;
  gap: 20rpx;
}

.entry-icon {
  width: 72rpx;
  height: 72rpx;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.entry-icon--request {
  background-color: #DBEAFE;
}

.entry-icon--block {
  background-color: #FEE2E2;
}

.entry-icon-text {
  font-size: 32rpx;
}

.entry-label {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
}

.entry-right {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.badge {
  min-width: 36rpx;
  height: 36rpx;
  padding: 0 10rpx;
  border-radius: 18rpx;
  background-color: #EF4444;
  color: #FFFFFF;
  font-size: 22rpx;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}

.arrow {
  font-size: 28rpx;
  color: #CBD5E1;
}

/* section header */
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20rpx 32rpx;
}

.section-title {
  font-size: 26rpx;
  color: #94A3B8;
  font-weight: 500;
}

.section-count {
  font-size: 24rpx;
  color: #CBD5E1;
}

/* 骨架屏 */
.skeleton-list {
  background-color: #FFFFFF;
  margin-top: 4rpx;
}

.skeleton-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  gap: 20rpx;
}

.skeleton-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

.skeleton-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12rpx;
}

.skeleton-line {
  border-radius: 8rpx;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

.skeleton-line--name {
  width: 50%;
  height: 28rpx;
}

.skeleton-line--sub {
  width: 30%;
  height: 22rpx;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* 好友列表 */
.friend-list {
  background-color: #FFFFFF;
}

.friend-item {
  display: flex;
  align-items: center;
  padding: 20rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  cursor: pointer;
  transition: background-color 200ms ease;
}

.friend-item:active {
  background-color: #F8FAFC;
}

.friend-item:last-child {
  border-bottom: none;
}

.avatar-wrap {
  position: relative;
  margin-right: 20rpx;
  flex-shrink: 0;
}

.avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-text {
  font-size: 32rpx;
  color: #FFFFFF;
  font-weight: 600;
}

.online-dot {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 20rpx;
  height: 20rpx;
  border-radius: 50%;
  background-color: #10B981;
  border: 3rpx solid #FFFFFF;
}

.friend-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
}

.friend-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
}

.friend-status {
  font-size: 24rpx;
  color: #94A3B8;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: 200rpx;
}

.empty-title {
  font-size: 32rpx;
  color: #64748B;
  font-weight: 500;
  margin-bottom: 12rpx;
}

.empty-desc {
  font-size: 26rpx;
  color: #94A3B8;
}
</style>
