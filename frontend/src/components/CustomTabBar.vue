<!--
  自定义 TabBar 组件
  
  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / Text #1E293B / Muted #94A3B8
  
  功能：
  - 底部导航栏，4 个 Tab：消息 / 联系人 / 会议 / 我的
  - 选中状态高亮（Primary 色）
  - 使用 Unicode 图标占位（后续可替换为 SVG 图标组件）
  - 支持 switchTab 跳转
  
  使用方式：在每个 TabBar 页面中引入此组件并放在页面底部
-->
<template>
  <view class="tab-bar">
    <view
      v-for="(item, index) in tabs"
      :key="item.path"
      class="tab-item"
      :class="{ 'tab-active': currentIndex === index }"
      @tap="switchTo(index)"
    >
      <view class="tab-icon-wrap">
        <text class="tab-icon">{{ item.icon }}</text>
        <text v-if="getBadge(index) > 0" class="tab-badge">{{ getBadge(index) > 99 ? '99+' : getBadge(index) }}</text>
      </view>
      <text class="tab-label">{{ item.label }}</text>
    </view>
  </view>
</template>

<script>
/**
 * 自定义 TabBar 组件
 *
 * Props:
 * - current: 当前选中的 Tab 索引
 *
 * 功能：
 * - 跳转使用 uni.switchTab，确保与 pages.json tabBar 配置的页面一致
 * - 消息 Tab（index=0）显示全局未读消息 badge
 */
import { useChatStore } from '@/store/chat'

export default {
  name: 'CustomTabBar',
  props: {
    /** 当前选中 Tab 的索引（0-3） */
    current: {
      type: Number,
      default: 0
    }
  },
  data() {
    return {
      /** Tab 配置列表 */
      tabs: [
        { path: '/pages/chat/index', label: '消息', icon: '💬' },
        { path: '/pages/contact/index', label: '联系人', icon: '👥' },
        { path: '/pages/meeting/index', label: '会议', icon: '🎥' },
        { path: '/pages/profile/index', label: '我的', icon: '👤' }
      ]
    }
  },
  computed: {
    currentIndex() {
      return this.current
    }
  },
  methods: {
    /**
     * 切换 Tab
     * @param {number} index - 目标 Tab 索引
     */
    switchTo(index) {
      if (index === this.currentIndex) return
      uni.switchTab({ url: this.tabs[index].path })
    },
    /**
     * 获取指定 Tab 的 badge 数
     * @param {number} index - Tab 索引
     * @returns {number} badge 数量（0 表示不显示）
     */
    getBadge(index) {
      if (index === 0) {
        const chatStore = useChatStore()
        return chatStore.totalUnread
      }
      return 0
    }
  }
}
</script>

<style scoped>
/*
 * 设计系统：MASTER.md
 * 选中色：#2563EB（Primary）
 * 未选中色：#94A3B8（Muted）
 * 背景：#FFFFFF + 顶部分割线
 * 高度：适配安全区域（底部 safe area）
 */
.tab-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 110rpx;
  background-color: #FFFFFF;
  display: flex;
  align-items: center;
  justify-content: space-around;
  border-top: 1rpx solid #E2E8F0;
  padding-bottom: env(safe-area-inset-bottom, 0);
  z-index: 999;
}

.tab-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  height: 100%;
  transition: color 200ms ease;
}

.tab-icon-wrap {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tab-icon {
  font-size: 40rpx;
  margin-bottom: 4rpx;
}

.tab-badge {
  position: absolute;
  top: -8rpx;
  right: -20rpx;
  min-width: 32rpx;
  height: 32rpx;
  padding: 0 8rpx;
  font-size: 20rpx;
  line-height: 32rpx;
  color: #FFFFFF;
  background-color: #EF4444;
  border-radius: 16rpx;
  text-align: center;
}

.tab-label {
  font-size: 22rpx;
  color: #94A3B8;
  font-weight: 400;
}

/* 选中状态 */
.tab-active .tab-label {
  color: #2563EB;
  font-weight: 500;
}
</style>
