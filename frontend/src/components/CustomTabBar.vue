<!--
  自定义 TabBar 组件

  设计系统：design-system/echochat/MASTER.md
  图标方案：@dcloudio/uni-ui uni-icons（跨平台兼容）
  色板：Primary #2563EB / Text #1E293B / Muted #94A3B8

  功能：
  - 底部导航栏，4 个 Tab：消息 / 联系人 / 会议 / 我的
  - 选中态使用 filled 图标 + Primary 色，未选中态使用轮廓图标 + Muted 色
  - 消息 Tab / 联系人 Tab：数字 badge（getBadge）
  - 我的 Tab：聚合小红点（hasDot），当前来源 notifyStore.unreadTotal，
    未来可扩展为「资料待完善」「安全提醒」「新版本可用」等聚合指示器
  - 使用 switchTab 跳转
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
        <uni-icons
          :type="currentIndex === index ? item.iconActive : item.icon"
          size="24"
          :color="currentIndex === index ? '#2563EB' : '#94A3B8'"
        />
        <!-- 优先显示数字 badge（消息/联系人）；否则若有未读红点则显示小红点（我的） -->
        <text v-if="getBadge(index) > 0" class="tab-badge">{{ getBadge(index) > 99 ? '99+' : getBadge(index) }}</text>
        <text v-else-if="hasDot(index)" class="tab-dot" />
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
 * - 联系人 Tab（index=1）显示待处理好友申请 badge
 * - 选中态使用 filled 图标变体以增强视觉区分
 */
import { useChatStore } from '@/store/chat'
import { useContactStore } from '@/store/contact'
import { useNotifyStore } from '@/store/notify'

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
      /** Tab 配置列表：icon 为未选中态，iconActive 为选中态 */
      tabs: [
        { path: '/pages/chat/index', label: '消息', icon: 'chatbubble', iconActive: 'chatbubble-filled' },
        { path: '/pages/contact/index', label: '联系人', icon: 'contact', iconActive: 'contact-filled' },
        { path: '/pages/meeting/index', label: '会议', icon: 'videocam', iconActive: 'videocam-filled' },
        { path: '/pages/profile/index', label: '我的', icon: 'person', iconActive: 'person-filled' }
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
     * 获取指定 Tab 的数字 badge 数（消息/联系人 Tab 使用）
     * @param {number} index - Tab 索引
     * @returns {number} badge 数量（0 表示不显示）
     */
    getBadge(index) {
      if (index === 0) {
        const chatStore = useChatStore()
        return chatStore.totalUnread
      }
      if (index === 1) {
        const contactStore = useContactStore()
        return contactStore.pendingCount
      }
      return 0
    },
    /**
     * 判断指定 Tab 是否需要显示小红点（聚合指示器）
     *
     * 当前实现：
     *   - index=3（我的）：聚合 notifyStore.unreadTotal > 0
     *
     * 未来扩展示例（仅示意，不是 TODO）：
     *   - 资料待完善：profileStore.hasProfileReminder
     *   - 安全提醒：securityStore.hasSecurityAlert
     *   - 新版本提示：appStore.hasNewVersion
     *   追加新来源时，仅需在 index=3 分支里 `|| 新来源`，外层调用方无需感知
     *
     * 与 getBadge 语义区分：
     *   - getBadge 返回数字（0 表示不显示）
     *   - hasDot 返回布尔（true 表示仅显示纯红点）
     *   模板先判断 getBadge，数字优先；无数字时再判断 hasDot
     *
     * @param {number} index - Tab 索引
     * @returns {boolean} 是否显示红点
     */
    hasDot(index) {
      if (index === 3) {
        const notifyStore = useNotifyStore()
        return notifyStore.unreadTotal > 0
      }
      return false
    }
  }
}
</script>

<style scoped>
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
  min-height: 88rpx;
  transition: opacity 150ms ease;
}

.tab-item:active {
  opacity: 0.7;
}

.tab-icon-wrap {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
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

/* 聚合未读小红点（用于"我的" Tab，纯红色圆点，不显示数字） */
.tab-dot {
  position: absolute;
  top: -4rpx;
  right: -8rpx;
  width: 16rpx;
  height: 16rpx;
  background-color: #EF4444;
  border-radius: 50%;
  border: 2rpx solid #FFFFFF;
}

.tab-label {
  font-size: 22rpx;
  color: #94A3B8;
  font-weight: 400;
}

.tab-active .tab-label {
  color: #2563EB;
  font-weight: 500;
}
</style>
