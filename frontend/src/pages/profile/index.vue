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
    /** 头像占位字母（取昵称或用户名首字符） */
    avatarLetter() {
      const name = this.userInfo.nickname || this.userInfo.username || '?'
      return name.charAt(0).toUpperCase()
    }
  },
  methods: {
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
