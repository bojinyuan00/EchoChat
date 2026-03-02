<!--
  黑名单管理页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B
  ui-ux-pro-max 规范：防重复提交 / 确认弹窗 / 过渡动画
-->
<template>
  <view class="page-wrapper">
    <!-- 骨架屏 -->
    <view v-if="loading" class="skeleton-list">
      <view v-for="i in 3" :key="i" class="skeleton-item">
        <view class="skeleton-avatar"></view>
        <view class="skeleton-info">
          <view class="skeleton-line"></view>
        </view>
        <view class="skeleton-btn"></view>
      </view>
    </view>

    <!-- 黑名单列表 -->
    <view v-else-if="contactStore.blockList.length > 0" class="block-list">
      <view
        v-for="user in contactStore.blockList"
        :key="user.user_id"
        class="block-item"
      >
        <view class="avatar" :style="{ backgroundColor: getAvatarColor(user.nickname || user.username) }">
          <text class="avatar-text">{{ getInitial(user.nickname || user.username) }}</text>
        </view>
        <view class="user-info">
          <text class="user-name">{{ user.nickname || user.username }}</text>
          <text class="user-account">@{{ user.username }}</text>
        </view>
        <view
          class="unblock-btn"
          :class="{ 'unblock-btn--disabled': processingMap[user.user_id] }"
          @tap="handleUnblock(user)"
        >
          <text class="unblock-btn-text">取消拉黑</text>
        </view>
      </view>
    </view>

    <!-- 空状态 -->
    <view v-else class="empty-state">
      <text class="empty-title">黑名单为空</text>
      <text class="empty-desc">被拉黑的用户会显示在这里</text>
    </view>
  </view>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useContactStore } from '@/store/contact'
import { getAvatarColor, getInitial } from '@/utils/avatar'

const contactStore = useContactStore()
const loading = ref(true)
const processingMap = reactive({})

const handleUnblock = (user) => {
  if (processingMap[user.user_id]) return
  uni.showModal({
    title: '取消拉黑',
    content: `确定要取消拉黑 ${user.nickname || user.username} 吗？`,
    success: async (res) => {
      if (res.confirm) {
        processingMap[user.user_id] = true
        try {
          await contactStore.unblockUser(user.user_id)
          uni.showToast({ title: '已取消拉黑', icon: 'success' })
        } catch (e) {
          console.error('取消拉黑失败', e)
        } finally {
          processingMap[user.user_id] = false
        }
      }
    }
  })
}

onMounted(async () => {
  try {
    await contactStore.fetchBlockList()
  } catch (e) {
    console.error('获取黑名单失败', e)
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
}

.skeleton-line {
  width: 40%;
  height: 28rpx;
  border-radius: 8rpx;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

.skeleton-btn {
  width: 120rpx;
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

/* 黑名单列表 */
.block-list {
  background-color: #FFFFFF;
}

.block-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  gap: 20rpx;
}

.block-item:last-child {
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

.user-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}

.user-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
}

.user-account {
  font-size: 24rpx;
  color: #94A3B8;
}

.unblock-btn {
  padding: 12rpx 24rpx;
  border: 2rpx solid #E2E8F0;
  border-radius: 12rpx;
  flex-shrink: 0;
  cursor: pointer;
  transition: all 200ms ease;
}

.unblock-btn:active {
  background-color: #F1F5F9;
}

.unblock-btn--disabled {
  opacity: 0.5;
  pointer-events: none;
}

.unblock-btn-text {
  font-size: 24rpx;
  color: #64748B;
  font-weight: 500;
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
}
</style>
