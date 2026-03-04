<!--
  登录页面
  
  设计系统：design-system/echochat/MASTER.md + pages/login.md
  色板：Primary #2563EB / CTA #F97316 / Background #F8FAFC / Text #1E293B
  风格：Clean Minimalism，单列居中，低内容密度
  
  功能：
  - 账号（用户名或邮箱）+ 密码登录
  - 表单验证（label + error feedback，不使用 placeholder-only）
  - Loading 状态反馈，禁用按钮防止重复提交
  - 登录成功跳转首页
  - "没有账号？去注册" 链接
  
  对应 API：POST /api/v1/auth/login
  接口文档：docs/api/frontend/auth.md
-->
<template>
  <view class="page-container">
    <!-- 顶部品牌区域 -->
    <view class="brand-section">
      <view class="logo-box">
        <text class="logo-letter">E</text>
      </view>
      <text class="brand-name">EchoChat</text>
      <text class="brand-slogan">连接无限，沟通无界</text>
    </view>

    <!-- 表单卡片 -->
    <view class="form-card">
      <text class="form-heading">登录</text>

      <!-- 账号 -->
      <view class="field-group">
        <text class="field-label">账号</text>
        <view class="input-box" :class="{ 'input-focused': focusState.account, 'input-error': errors.account }">
          <input
            class="input-control"
            type="text"
            placeholder="请输入用户名或邮箱"
            v-model="form.account"
            @focus="focusState.account = true"
            @blur="onBlur('account')"
            maxlength="50"
          />
        </view>
        <text v-if="errors.account" class="field-error">{{ errors.account }}</text>
      </view>

      <!-- 密码 -->
      <view class="field-group">
        <text class="field-label">密码</text>
        <view class="input-box" :class="{ 'input-focused': focusState.password, 'input-error': errors.password }">
          <input
            class="input-control"
            :type="passwordVisible ? 'text' : 'password'"
            placeholder="请输入密码"
            v-model="form.password"
            @focus="focusState.password = true"
            @blur="onBlur('password')"
            maxlength="50"
            @confirm="submit"
          />
          <text class="pwd-toggle" @tap="passwordVisible = !passwordVisible">
            {{ passwordVisible ? '隐藏' : '显示' }}
          </text>
        </view>
        <text v-if="errors.password" class="field-error">{{ errors.password }}</text>
      </view>

      <!-- 登录按钮 -->
      <button class="btn-primary" :class="{ 'btn-disabled': loading }" :disabled="loading" @tap="submit">
        <text class="btn-label">{{ loading ? '登录中…' : '登录' }}</text>
      </button>

      <!-- 底部链接 -->
      <view class="link-row">
        <text class="link-hint">还没有账号？</text>
        <text class="link-action" @tap="goRegister">立即注册</text>
      </view>
    </view>
  </view>
</template>

<script>
/**
 * 登录页面逻辑
 *
 * 使用 useUserStore 处理登录
 * 登录成功后 reLaunch 到首页
 */
import { useUserStore } from '@/store/user'
import { useWebSocketStore } from '@/store/websocket'
import { useChatStore } from '@/store/chat'
import { useContactStore } from '@/store/contact'

export default {
  name: 'LoginPage',
  data() {
    return {
      form: { account: '', password: '' },
      passwordVisible: false,
      loading: false,
      focusState: { account: false, password: false },
      errors: { account: '', password: '' }
    }
  },
  methods: {
    /** 失焦时触发对应字段校验 */
    onBlur(field) {
      this.focusState[field] = false
      this.validate(field)
    },

    /**
     * 单字段校验
     * @param {string} field
     * @returns {boolean}
     */
    validate(field) {
      if (field === 'account') {
        if (!this.form.account.trim()) {
          this.errors.account = '请输入用户名或邮箱'
          return false
        }
        this.errors.account = ''
        return true
      }
      if (field === 'password') {
        if (!this.form.password) {
          this.errors.password = '请输入密码'
          return false
        }
        if (this.form.password.length < 6) {
          this.errors.password = '密码至少 6 位'
          return false
        }
        this.errors.password = ''
        return true
      }
      return true
    },

    /** 提交登录 */
    async submit() {
      if (this.loading) return
      const a = this.validate('account')
      const p = this.validate('password')
      if (!a || !p) return

      this.loading = true
      try {
        const store = useUserStore()
        await store.login({
          account: this.form.account.trim(),
          password: this.form.password
        })
        const wsStore = useWebSocketStore()
        wsStore.connect()
        useChatStore().initWsListeners()
        useContactStore().initWsListeners()
        uni.showToast({ title: '登录成功', icon: 'success' })
        setTimeout(() => uni.reLaunch({ url: '/pages/index/index' }), 800)
      } catch (e) {
        console.error('登录失败:', e)
      } finally {
        this.loading = false
      }
    },

    goRegister() {
      uni.navigateTo({ url: '/pages/auth/register' })
    }
  }
}
</script>

<style scoped>
/*
 * 设计系统来源：design-system/echochat/MASTER.md + pages/login.md
 * 色板：Primary #2563EB / CTA #F97316 / BG #F8FAFC / Text #1E293B
 * 输入框规范：border #E2E8F0 / radius 8px(16rpx) / focus border #2563EB + shadow
 * 按钮规范：radius 8px(16rpx) / transition 200ms / cursor-pointer
 * 间距规范：--space-md 16px(32rpx) / --space-lg 24px(48rpx)
 */

.page-container {
  min-height: 100vh;
  background-color: #F8FAFC;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0 48rpx;
}

/* ---- 品牌区域 ---- */
.brand-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 120rpx;
  margin-bottom: 64rpx;
}

.logo-box {
  width: 112rpx;
  height: 112rpx;
  border-radius: 24rpx;
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-letter {
  font-size: 52rpx;
  font-weight: 700;
  color: #FFFFFF;
}

.brand-name {
  font-size: 44rpx;
  font-weight: 700;
  color: #1E293B;
  margin-top: 24rpx;
  letter-spacing: 2rpx;
}

.brand-slogan {
  font-size: 26rpx;
  color: #64748B;
  margin-top: 8rpx;
}

/* ---- 表单卡片 ---- */
.form-card {
  width: 100%;
  max-width: 800rpx;
  background-color: #FFFFFF;
  border-radius: 24rpx;
  padding: 56rpx 40rpx 48rpx;
  box-shadow: 0 4rpx 12rpx rgba(0, 0, 0, 0.06);
}

.form-heading {
  font-size: 40rpx;
  font-weight: 600;
  color: #1E293B;
  margin-bottom: 48rpx;
}

/* ---- 输入字段 ---- */
.field-group {
  margin-bottom: 36rpx;
}

.field-label {
  display: block;
  font-size: 26rpx;
  font-weight: 500;
  color: #475569;
  margin-bottom: 12rpx;
}

.input-box {
  display: flex;
  align-items: center;
  height: 96rpx;
  background-color: #FFFFFF;
  border: 2rpx solid #E2E8F0;
  border-radius: 16rpx;
  padding: 0 24rpx;
  transition: border-color 200ms ease, box-shadow 200ms ease;
}

.input-box.input-focused {
  border-color: #2563EB;
  box-shadow: 0 0 0 6rpx rgba(37, 99, 235, 0.12);
}

.input-box.input-error {
  border-color: #EF4444;
}

.input-control {
  flex: 1;
  font-size: 30rpx;
  color: #1E293B;
}

.pwd-toggle {
  font-size: 24rpx;
  color: #2563EB;
  font-weight: 500;
  padding: 12rpx 0 12rpx 16rpx;
}

.field-error {
  display: block;
  font-size: 22rpx;
  color: #EF4444;
  margin-top: 8rpx;
  padding-left: 4rpx;
}

/* ---- 按钮（CTA 色 #F97316） ---- */
.btn-primary {
  width: 100%;
  height: 96rpx;
  background-color: #2563EB;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 48rpx;
  border: none;
  transition: opacity 200ms ease;
}

.btn-primary::after {
  border: none;
}

.btn-primary.btn-disabled {
  opacity: 0.6;
}

.btn-label {
  font-size: 32rpx;
  font-weight: 600;
  color: #FFFFFF;
  letter-spacing: 2rpx;
}

/* ---- 底部链接 ---- */
.link-row {
  display: flex;
  justify-content: center;
  align-items: center;
  margin-top: 40rpx;
}

.link-hint {
  font-size: 26rpx;
  color: #94A3B8;
}

.link-action {
  font-size: 26rpx;
  color: #2563EB;
  font-weight: 500;
  margin-left: 8rpx;
}
</style>
