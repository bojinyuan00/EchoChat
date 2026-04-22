<!--
  会议首页（Meeting Hub）

  职责：
  - 作为 tabBar "会议" 的统一入口，提供创建 / 加入 / 最近会议跳转
  - 不直接承载 mediasoup 连接，只做导航与轻量查询

  数据流：
  - 点击「创建即时会议」→ /pages/meeting/create
  - 点击「加入会议」→ /pages/meeting/join
-->
<template>
  <view class="page">
    <view class="hero">
      <view class="hero-icon">🎥</view>
      <text class="hero-title">视频会议</text>
      <text class="hero-desc">创建或加入会议，进行高清音视频协作</text>
    </view>

    <view class="actions">
      <view class="action-card primary" @click="goCreate">
        <view class="action-icon">
          <svg viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polygon points="23 7 16 12 23 17 23 7"></polygon>
            <rect x="1" y="5" width="15" height="14" rx="2" ry="2"></rect>
          </svg>
        </view>
        <view class="action-text">
          <text class="action-title">创建即时会议</text>
          <text class="action-desc">立即发起一场新会议并邀请成员</text>
        </view>
        <view class="action-arrow">›</view>
      </view>

      <view class="action-card" @click="goJoin">
        <view class="action-icon alt">
          <svg viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 9l6 6-6 6"></path>
            <path d="M4 4v7a4 4 0 0 0 4 4h12"></path>
          </svg>
        </view>
        <view class="action-text">
          <text class="action-title">加入会议</text>
          <text class="action-desc">输入 9 位会议号加入已有会议</text>
        </view>
        <view class="action-arrow">›</view>
      </view>
    </view>

    <view class="tips">
      <text class="tips-title">提示</text>
      <text class="tips-line">• 入会前会先进入设备预览页选择摄像头 / 麦克风</text>
      <text class="tips-line">• 主持人可在会议中邀请成员、管理权限、结束会议</text>
      <text class="tips-line">• 支持静音、关闭摄像头、实时聊天等常用操作</text>
    </view>

    <CustomTabBar :current="2" />
  </view>
</template>

<script>
import CustomTabBar from '@/components/CustomTabBar.vue'

export default {
  name: 'MeetingIndex',
  components: { CustomTabBar },
  methods: {
    goCreate() {
      uni.navigateTo({ url: '/pages/meeting/create' })
    },
    goJoin() {
      uni.navigateTo({ url: '/pages/meeting/join' })
    }
  }
}
</script>

<style scoped>
.page {
  min-height: 100vh;
  background: linear-gradient(180deg, #EEF4FF 0%, #F8FAFC 40%);
  padding: 48rpx 32rpx 180rpx;
  box-sizing: border-box;
}

.hero {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  padding: 20rpx 8rpx 40rpx;
}
.hero-icon {
  font-size: 72rpx;
  line-height: 1;
  margin-bottom: 16rpx;
}
.hero-title {
  font-size: 44rpx;
  font-weight: 700;
  color: #0F172A;
  margin-bottom: 8rpx;
}
.hero-desc {
  font-size: 26rpx;
  color: #64748B;
  line-height: 1.6;
}

.actions {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-top: 8rpx;
}

.action-card {
  display: flex;
  align-items: center;
  gap: 24rpx;
  padding: 28rpx 28rpx;
  background: #FFFFFF;
  border-radius: 20rpx;
  box-shadow: 0 4rpx 16rpx rgba(15, 23, 42, 0.05);
  transition: transform 0.12s ease, box-shadow 0.12s ease;
  cursor: pointer;
}
.action-card:active {
  transform: translateY(1rpx);
  box-shadow: 0 2rpx 8rpx rgba(15, 23, 42, 0.08);
}
.action-card.primary {
  background: linear-gradient(135deg, #3B82F6 0%, #2563EB 100%);
  box-shadow: 0 6rpx 20rpx rgba(59, 130, 246, 0.28);
}
.action-card.primary .action-title,
.action-card.primary .action-desc,
.action-card.primary .action-arrow {
  color: #FFFFFF;
}
.action-card.primary .action-desc { color: rgba(255, 255, 255, 0.82); }

.action-icon {
  width: 88rpx;
  height: 88rpx;
  border-radius: 20rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.22);
  color: #FFFFFF;
  flex-shrink: 0;
}
.action-icon.alt {
  background: rgba(59, 130, 246, 0.12);
  color: #2563EB;
}

.action-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
}
.action-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #0F172A;
}
.action-desc {
  font-size: 24rpx;
  color: #64748B;
  line-height: 1.5;
}
.action-arrow {
  font-size: 40rpx;
  color: #CBD5E1;
  flex-shrink: 0;
}

.tips {
  margin-top: 48rpx;
  padding: 24rpx 28rpx;
  background: rgba(255, 255, 255, 0.7);
  border-radius: 16rpx;
  border: 1rpx solid rgba(59, 130, 246, 0.08);
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}
.tips-title {
  font-size: 26rpx;
  font-weight: 600;
  color: #0F172A;
  margin-bottom: 4rpx;
}
.tips-line {
  font-size: 24rpx;
  color: #475569;
  line-height: 1.7;
}
</style>
