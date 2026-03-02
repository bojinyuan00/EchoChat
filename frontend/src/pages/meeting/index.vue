<!--
  会议页面（占位）
  
  设计系统：design-system/echochat/MASTER.md
  TabBar 页面，Phase 3（会议模块）中实现具体功能
-->
<template>
  <view class="page-wrapper">
    <view class="placeholder-content">
      <text class="placeholder-icon">🎥</text>
      <text class="placeholder-title">会议</text>
      <text class="placeholder-desc">视频会议功能开发中…</text>
    </view>
    <CustomTabBar :current="2" />
  </view>
</template>

<script>
import CustomTabBar from '@/components/CustomTabBar.vue'

export default {
  name: 'MeetingIndex',
  components: { CustomTabBar }
}
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
  padding-bottom: 120rpx;
}

.placeholder-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: 300rpx;
}

.placeholder-icon {
  font-size: 80rpx;
  margin-bottom: 24rpx;
}

.placeholder-title {
  font-size: 36rpx;
  font-weight: 600;
  color: #1E293B;
  margin-bottom: 12rpx;
}

.placeholder-desc {
  font-size: 28rpx;
  color: #94A3B8;
}
</style>
