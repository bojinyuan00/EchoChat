<!--
  个人中心页面
  
  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B / Muted #94A3B8
  
  功能：
  - 显示用户头像、昵称、用户名
  - 编辑资料入口（跳转编辑页，后续实现）
  - 修改密码入口（跳转修改页，后续实现）
  - 退出登录按钮
  - 集成自定义 TabBar 组件
  
  对应 API：GET /api/v1/auth/profile、POST /api/v1/auth/logout
-->
<template>
  <view class="page-wrapper">
    <!-- 顶部栏（铃铛入口 + 徽标） -->
    <view class="top-bar">
      <text class="top-title">我的</text>
      <view class="bell-btn" @tap="goNotifyCenter">
        <text class="bell-icon">&#128276;</text>
        <view v-if="notifyStore.unreadTotal > 0" class="bell-badge">
          {{ notifyStore.unreadTotal > 99 ? '99+' : notifyStore.unreadTotal }}
        </view>
      </view>
    </view>

    <!-- 用户信息头部 -->
    <view class="profile-header">
      <view class="avatar-box">
        <text class="avatar-letter">{{ avatarLetter }}</text>
      </view>
      <text class="nickname">{{ userInfo.nickname || userInfo.username || '未登录' }}</text>
      <text class="username" v-if="userInfo.username">@{{ userInfo.username }}</text>
    </view>

    <!-- 功能菜单 -->
    <view class="menu-card">
      <view class="menu-item" @tap="goNotifyCenter">
        <text class="menu-label">通知中心</text>
        <view class="menu-right">
          <view v-if="notifyStore.unreadTotal > 0" class="menu-badge">
            {{ notifyStore.unreadTotal > 99 ? '99+' : notifyStore.unreadTotal }}
          </view>
          <text class="menu-arrow">›</text>
        </view>
      </view>
      <view class="menu-divider"></view>
      <view class="menu-item" @tap="goEditProfile">
        <text class="menu-label">编辑资料</text>
        <text class="menu-arrow">›</text>
      </view>
      <view class="menu-divider"></view>
      <view class="menu-item" @tap="goChangePassword">
        <text class="menu-label">修改密码</text>
        <text class="menu-arrow">›</text>
      </view>
    </view>

    <!-- 退出登录 -->
    <view class="logout-section">
      <button class="logout-btn" @tap="handleLogout">
        <text class="logout-text">退出登录</text>
      </button>
    </view>

    <CustomTabBar :current="3" />
  </view>
</template>

<script>
/**
 * 个人中心页面逻辑
 *
 * 从 useUserStore 获取用户信息
 * 退出登录时调用 store.logout()，清除 Redis Token + 本地缓存
 */
import { useUserStore } from '@/store/user'
import { useNotifyStore } from '@/store/notify'
import CustomTabBar from '@/components/CustomTabBar.vue'

export default {
  name: 'ProfileIndex',
  components: { CustomTabBar },
  computed: {
    /** 从 Store 获取用户信息 */
    userInfo() {
      const store = useUserStore()
      return store.userInfo || {}
    },
    /** 通知 Store（用于读取未读数并驱动铃铛徽标） */
    notifyStore() {
      return useNotifyStore()
    },
    /** 头像占位字母（取昵称或用户名首字符） */
    avatarLetter() {
      const name = this.userInfo.nickname || this.userInfo.username || '?'
      return name.charAt(0).toUpperCase()
    }
  },
  onShow() {
    // 每次进入「我的」页面时刷新未读数，保证徽标实时
    const store = useNotifyStore()
    store.initWsListeners()
    store.fetchUnreadCount().catch(() => {})
  },
  methods: {
    /** 跳转到通知中心页 */
    goNotifyCenter() {
      uni.navigateTo({ url: '/pages/notify/index' })
    },

    /** 编辑资料（后续实现，当前提示开发中） */
    goEditProfile() {
      uni.showToast({ title: '功能开发中', icon: 'none' })
    },

    /** 修改密码（后续实现） */
    goChangePassword() {
      uni.showToast({ title: '功能开发中', icon: 'none' })
    },

    /** 退出登录 */
    async handleLogout() {
      uni.showModal({
        title: '提示',
        content: '确定要退出登录吗？',
        success: async (res) => {
          if (!res.confirm) return
          const store = useUserStore()
          await store.logout()
          uni.reLaunch({ url: '/pages/auth/login' })
        }
      })
    }
  }
}
</script>

<style scoped>
/*
 * 设计系统：MASTER.md
 * 背景：#F8FAFC / 卡片：#FFFFFF / 文字：#1E293B / 辅助：#94A3B8
 * 圆角：24rpx（卡片）/ 按钮 16rpx
 * 间距：space-lg 48rpx / space-md 32rpx
 */
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
  padding-bottom: 120rpx;
}

/* ---- 顶部栏（铃铛入口） ---- */
.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20rpx 32rpx;
  background-color: #FFFFFF;
}

.top-title {
  font-size: 36rpx;
  font-weight: 700;
  color: #1E293B;
}

.bell-btn {
  position: relative;
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

.bell-btn:active {
  background-color: #E2E8F0;
}

.bell-icon {
  font-size: 36rpx;
  color: #1E293B;
  line-height: 1;
}

.bell-badge {
  position: absolute;
  top: 4rpx;
  right: 4rpx;
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
  border: 2rpx solid #FFFFFF;
}

.menu-right {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.menu-badge {
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

/* ---- 用户信息头部 ---- */
.profile-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80rpx 48rpx 48rpx;
  background-color: #FFFFFF;
  margin-bottom: 24rpx;
}

.avatar-box {
  width: 128rpx;
  height: 128rpx;
  border-radius: 50%;
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20rpx;
}

.avatar-letter {
  font-size: 52rpx;
  font-weight: 700;
  color: #FFFFFF;
}

.nickname {
  font-size: 36rpx;
  font-weight: 600;
  color: #1E293B;
  margin-bottom: 6rpx;
}

.username {
  font-size: 26rpx;
  color: #94A3B8;
}

/* ---- 功能菜单 ---- */
.menu-card {
  background-color: #FFFFFF;
  margin: 0 32rpx;
  border-radius: 24rpx;
  overflow: hidden;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.04);
}

.menu-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 36rpx 40rpx;
}

.menu-label {
  font-size: 30rpx;
  color: #1E293B;
}

.menu-arrow {
  font-size: 32rpx;
  color: #CBD5E1;
}

.menu-divider {
  height: 1rpx;
  background-color: #F1F5F9;
  margin: 0 40rpx;
}

/* ---- 退出登录 ---- */
.logout-section {
  padding: 48rpx 32rpx;
}

.logout-btn {
  width: 100%;
  height: 88rpx;
  background-color: #FFFFFF;
  border: 2rpx solid #EF4444;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logout-btn::after {
  border: none;
}

.logout-text {
  font-size: 30rpx;
  color: #EF4444;
  font-weight: 500;
}
</style>
