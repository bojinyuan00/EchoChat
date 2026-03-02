<!--
  启动页 / 路由分发页
  
  功能：
  - 判断用户登录状态（检查本地 Token）
  - 已登录 → 跳转消息列表页（TabBar 首页）
  - 未登录 → 跳转登录页
  
  此页面不展示 UI，仅作为路由分发入口
-->
<template>
  <view class="launch-page">
    <text class="loading-text">加载中…</text>
  </view>
</template>

<script>
/**
 * 启动页逻辑
 *
 * 在 onLoad 生命周期中检查登录状态并重定向
 * 使用 reLaunch 确保清除导航栈
 */
import { useUserStore } from '@/store/user'

export default {
  name: 'LaunchPage',
  onLoad() {
    const store = useUserStore()
    if (store.isLoggedIn) {
      uni.switchTab({ url: '/pages/chat/index' })
    } else {
      uni.reLaunch({ url: '/pages/auth/login' })
    }
  }
}
</script>

<style scoped>
.launch-page {
  min-height: 100vh;
  background-color: #F8FAFC;
  display: flex;
  align-items: center;
  justify-content: center;
}

.loading-text {
  font-size: 28rpx;
  color: #94A3B8;
}
</style>
