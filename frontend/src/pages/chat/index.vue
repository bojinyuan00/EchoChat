<!--
  消息列表页面（占位）
  
  设计系统：design-system/echochat/MASTER.md
  TabBar 页面，Phase 2（IM 模块）中实现具体功能
  
  当前功能：
  - 显示模块名称和开发中提示
  - 集成自定义 TabBar 组件
-->
<template>
  <view class="page-wrapper">
    <view class="placeholder-content">
      <text class="placeholder-icon">💬</text>
      <text class="placeholder-title">消息</text>
      <text class="placeholder-desc">即时通讯功能开发中…</text>
    </view>
    <CustomTabBar :current="0" />
  </view>
</template>

<script>
import CustomTabBar from '@/components/CustomTabBar.vue'

export default {
  name: 'ChatIndex',
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
