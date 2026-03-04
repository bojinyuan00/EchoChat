<!--
  好友申请列表页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B / Muted #94A3B8
  ui-ux-pro-max 规范：防重复提交 / 骨架屏 / 过渡动画
-->
<template>
  <view class="page-wrapper">
    <!-- 骨架屏 -->
    <view v-if="loading" class="skeleton-list">
      <view v-for="i in 4" :key="i" class="skeleton-item">
        <view class="skeleton-avatar"></view>
        <view class="skeleton-info">
          <view class="skeleton-line skeleton-line--name"></view>
          <view class="skeleton-line skeleton-line--msg"></view>
        </view>
        <view class="skeleton-actions">
          <view class="skeleton-btn"></view>
          <view class="skeleton-btn"></view>
        </view>
      </view>
    </view>

    <!-- 申请列表 -->
    <view v-else-if="contactStore.pendingRequests.length > 0" class="request-list">
      <view
        v-for="req in contactStore.pendingRequests"
        :key="req.id"
        class="request-item"
      >
        <view class="avatar" :style="{ backgroundColor: getAvatarColor(req.nickname || req.username) }">
          <text class="avatar-text">{{ getInitial(req.nickname || req.username) }}</text>
        </view>
        <view class="request-info">
          <text class="request-name">{{ req.nickname || req.username }}</text>
          <text class="request-msg" v-if="req.message">{{ req.message }}</text>
          <text class="request-msg" v-else>请求添加你为好友</text>
          <text class="request-time">{{ formatTime(req.created_at) }}</text>
        </view>
        <view class="request-actions">
          <view
            class="btn btn--accept"
            :class="{ 'btn--disabled': processingMap[req.id] }"
            @tap="handleAccept(req.id)"
          >
            <text class="btn-text">接受</text>
          </view>
          <view
            class="btn btn--reject"
            :class="{ 'btn--disabled': processingMap[req.id] }"
            @tap="handleReject(req.id)"
          >
            <text class="btn-text btn-text--reject">拒绝</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 空状态 -->
    <view v-else class="empty-state">
      <text class="empty-title">暂无好友申请</text>
      <text class="empty-desc">当有人向你发送好友申请时，会在这里显示</text>
    </view>
  </view>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { useContactStore } from '@/store/contact'
import { getAvatarColor, getInitial } from '@/utils/avatar'

const contactStore = useContactStore()
const loading = ref(true)
const processingMap = reactive({})

const formatTime = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now - date
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days} 天前`
  return `${date.getMonth() + 1}/${date.getDate()}`
}

const handleAccept = async (requestId) => {
  if (processingMap[requestId]) return
  processingMap[requestId] = true
  try {
    await contactStore.acceptRequest(requestId)
    uni.showToast({ title: '已接受', icon: 'success' })
  } catch (e) {
    console.error('接受好友申请失败', e)
    uni.showToast({ title: e?.data?.message || '接受申请失败', icon: 'none' })
  } finally {
    processingMap[requestId] = false
  }
}

const handleReject = async (requestId) => {
  if (processingMap[requestId]) return
  processingMap[requestId] = true
  try {
    await contactStore.rejectRequest(requestId)
    uni.showToast({ title: '已拒绝', icon: 'none' })
  } catch (e) {
    console.error('拒绝好友申请失败', e)
    uni.showToast({ title: e?.data?.message || '拒绝申请失败', icon: 'none' })
  } finally {
    processingMap[requestId] = false
  }
}

onMounted(async () => {
  try {
    await contactStore.fetchPendingRequests()
  } catch (e) {
    console.error('获取好友申请失败', e)
  }
  loading.value = false
})
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
}

/* 骨架屏 */
.skeleton-list {
  background-color: #FFFFFF;
}

.skeleton-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  gap: 20rpx;
  border-bottom: 1rpx solid #F1F5F9;
}

.skeleton-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  flex-shrink: 0;
}

.skeleton-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10rpx;
}

.skeleton-line {
  border-radius: 8rpx;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

.skeleton-line--name {
  width: 40%;
  height: 28rpx;
}

.skeleton-line--msg {
  width: 60%;
  height: 24rpx;
}

.skeleton-actions {
  display: flex;
  gap: 12rpx;
}

.skeleton-btn {
  width: 100rpx;
  height: 56rpx;
  border-radius: 12rpx;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* 申请列表 */
.request-list {
  background-color: #FFFFFF;
}

.request-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  gap: 20rpx;
}

.request-item:last-child {
  border-bottom: none;
}

.avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.avatar-text {
  font-size: 32rpx;
  color: #FFFFFF;
  font-weight: 600;
}

.request-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
}

.request-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
}

.request-msg {
  font-size: 24rpx;
  color: #64748B;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.request-time {
  font-size: 22rpx;
  color: #94A3B8;
}

.request-actions {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
  flex-shrink: 0;
}

.btn {
  padding: 12rpx 28rpx;
  border-radius: 12rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: opacity 200ms ease;
}

.btn:active {
  opacity: 0.7;
}

.btn--accept {
  background-color: #2563EB;
}

.btn--reject {
  background-color: #F1F5F9;
}

.btn--disabled {
  opacity: 0.5;
  pointer-events: none;
}

.btn-text {
  font-size: 24rpx;
  font-weight: 500;
  color: #FFFFFF;
}

.btn-text--reject {
  color: #64748B;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: 300rpx;
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
  text-align: center;
  padding: 0 60rpx;
}
</style>
