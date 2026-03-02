<!--
  注册页面
  
  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / CTA #F97316 / Background #F8FAFC / Text #1E293B
  风格：Clean Minimalism，单列居中，低内容密度
  
  功能：
  - 用户名 + 邮箱 + 密码 + 确认密码 + 昵称（选填）
  - 实时表单验证（label + error feedback，不使用 placeholder-only）
  - 注册成功自动登录并跳转首页
  - "已有账号？去登录" 链接
  - Loading 状态防止重复提交
  
  对应 API：POST /api/v1/auth/register
  接口文档：docs/api/frontend/auth.md
-->
<template>
  <view class="page-container">
    <!-- 顶部品牌区域 -->
    <view class="brand-section">
      <view class="logo-box">
        <text class="logo-letter">E</text>
      </view>
      <text class="brand-name">创建账号</text>
      <text class="brand-slogan">加入 EchoChat，开启高效沟通</text>
    </view>

    <!-- 表单卡片 -->
    <view class="form-card">
      <!-- 用户名 -->
      <view class="field-group">
        <text class="field-label">用户名 <text class="required-mark">*</text></text>
        <view class="input-box" :class="{ 'input-focused': focus.username, 'input-error': errors.username }">
          <input
            class="input-control"
            type="text"
            placeholder="3-50 个字符，全局唯一"
            v-model="form.username"
            @focus="focus.username = true"
            @blur="onBlur('username')"
            maxlength="50"
          />
        </view>
        <text v-if="errors.username" class="field-error">{{ errors.username }}</text>
      </view>

      <!-- 邮箱 -->
      <view class="field-group">
        <text class="field-label">邮箱 <text class="required-mark">*</text></text>
        <view class="input-box" :class="{ 'input-focused': focus.email, 'input-error': errors.email }">
          <input
            class="input-control"
            type="text"
            placeholder="your@email.com"
            v-model="form.email"
            @focus="focus.email = true"
            @blur="onBlur('email')"
            maxlength="100"
          />
        </view>
        <text v-if="errors.email" class="field-error">{{ errors.email }}</text>
      </view>

      <!-- 密码 -->
      <view class="field-group">
        <text class="field-label">密码 <text class="required-mark">*</text></text>
        <view class="input-box" :class="{ 'input-focused': focus.password, 'input-error': errors.password }">
          <input
            class="input-control"
            :type="passwordVisible ? 'text' : 'password'"
            placeholder="至少 6 个字符"
            v-model="form.password"
            @focus="focus.password = true"
            @blur="onBlur('password')"
            maxlength="50"
          />
          <text class="pwd-toggle" @tap="passwordVisible = !passwordVisible">
            {{ passwordVisible ? '隐藏' : '显示' }}
          </text>
        </view>
        <text v-if="errors.password" class="field-error">{{ errors.password }}</text>
      </view>

      <!-- 确认密码 -->
      <view class="field-group">
        <text class="field-label">确认密码 <text class="required-mark">*</text></text>
        <view class="input-box" :class="{ 'input-focused': focus.confirmPwd, 'input-error': errors.confirmPwd }">
          <input
            class="input-control"
            :type="passwordVisible ? 'text' : 'password'"
            placeholder="再次输入密码"
            v-model="form.confirmPwd"
            @focus="focus.confirmPwd = true"
            @blur="onBlur('confirmPwd')"
            maxlength="50"
          />
        </view>
        <text v-if="errors.confirmPwd" class="field-error">{{ errors.confirmPwd }}</text>
      </view>

      <!-- 昵称 -->
      <view class="field-group">
        <text class="field-label">昵称 <text class="optional-mark">(选填)</text></text>
        <view class="input-box" :class="{ 'input-focused': focus.nickname }">
          <input
            class="input-control"
            type="text"
            placeholder="不填则默认使用用户名"
            v-model="form.nickname"
            @focus="focus.nickname = true"
            @blur="focus.nickname = false"
            maxlength="50"
            @confirm="submit"
          />
        </view>
      </view>

      <!-- 注册按钮 -->
      <button class="btn-primary" :class="{ 'btn-disabled': loading }" :disabled="loading" @tap="submit">
        <text class="btn-label">{{ loading ? '注册中…' : '注册' }}</text>
      </button>

      <!-- 底部链接 -->
      <view class="link-row">
        <text class="link-hint">已有账号？</text>
        <text class="link-action" @tap="goLogin">去登录</text>
      </view>
    </view>

    <!-- 底部留白，避免键盘弹起时遮挡 -->
    <view class="bottom-spacer"></view>
  </view>
</template>

<script>
/**
 * 注册页面逻辑
 *
 * 使用 useUserStore 处理注册（注册成功自动登录）
 * 注册成功后 reLaunch 到首页
 */
import { useUserStore } from '@/store/user'

/** 邮箱校验正则 */
const EMAIL_RE = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

export default {
  name: 'RegisterPage',
  data() {
    return {
      form: { username: '', email: '', password: '', confirmPwd: '', nickname: '' },
      passwordVisible: false,
      loading: false,
      focus: { username: false, email: false, password: false, confirmPwd: false, nickname: false },
      errors: { username: '', email: '', password: '', confirmPwd: '' }
    }
  },
  methods: {
    /** 失焦时触发字段校验 */
    onBlur(field) {
      this.focus[field] = false
      this.validate(field)
    },

    /**
     * 单字段校验
     * @param {string} field
     * @returns {boolean}
     */
    validate(field) {
      switch (field) {
        case 'username':
          if (!this.form.username.trim()) { this.errors.username = '请输入用户名'; return false }
          if (this.form.username.trim().length < 3) { this.errors.username = '用户名至少 3 个字符'; return false }
          this.errors.username = ''
          return true

        case 'email':
          if (!this.form.email.trim()) { this.errors.email = '请输入邮箱'; return false }
          if (!EMAIL_RE.test(this.form.email.trim())) { this.errors.email = '邮箱格式不正确'; return false }
          this.errors.email = ''
          return true

        case 'password':
          if (!this.form.password) { this.errors.password = '请输入密码'; return false }
          if (this.form.password.length < 6) { this.errors.password = '密码至少 6 位'; return false }
          this.errors.password = ''
          if (this.form.confirmPwd) this.validate('confirmPwd')
          return true

        case 'confirmPwd':
          if (!this.form.confirmPwd) { this.errors.confirmPwd = '请再次输入密码'; return false }
          if (this.form.confirmPwd !== this.form.password) { this.errors.confirmPwd = '两次密码不一致'; return false }
          this.errors.confirmPwd = ''
          return true

        default:
          return true
      }
    },

    /** 全字段校验 */
    validateAll() {
      const fields = ['username', 'email', 'password', 'confirmPwd']
      let ok = true
      fields.forEach(f => { if (!this.validate(f)) ok = false })
      return ok
    },

    /** 提交注册 */
    async submit() {
      if (this.loading) return
      if (!this.validateAll()) return

      this.loading = true
      try {
        const store = useUserStore()
        await store.register({
          username: this.form.username.trim(),
          email: this.form.email.trim(),
          password: this.form.password,
          nickname: this.form.nickname.trim() || undefined
        })
        uni.showToast({ title: '注册成功', icon: 'success' })
        setTimeout(() => uni.reLaunch({ url: '/pages/index/index' }), 800)
      } catch (e) {
        console.error('注册失败:', e)
      } finally {
        this.loading = false
      }
    },

    /** 返回登录页 */
    goLogin() {
      uni.navigateBack({
        fail: () => uni.redirectTo({ url: '/pages/auth/login' })
      })
    }
  }
}
</script>

<style scoped>
/*
 * 设计系统来源：design-system/echochat/MASTER.md
 * 色板：Primary #2563EB / CTA #F97316 / BG #F8FAFC / Text #1E293B
 * 输入框规范：border #E2E8F0 / radius 16rpx / focus border #2563EB + shadow
 * 按钮规范：radius 16rpx / transition 200ms
 * 间距规范：field-group mb 32rpx
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
  margin-top: 80rpx;
  margin-bottom: 48rpx;
}

.logo-box {
  width: 96rpx;
  height: 96rpx;
  border-radius: 20rpx;
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-letter {
  font-size: 44rpx;
  font-weight: 700;
  color: #FFFFFF;
}

.brand-name {
  font-size: 40rpx;
  font-weight: 700;
  color: #1E293B;
  margin-top: 20rpx;
}

.brand-slogan {
  font-size: 24rpx;
  color: #64748B;
  margin-top: 6rpx;
}

/* ---- 表单卡片 ---- */
.form-card {
  width: 100%;
  max-width: 800rpx;
  background-color: #FFFFFF;
  border-radius: 24rpx;
  padding: 48rpx 40rpx 40rpx;
  box-shadow: 0 4rpx 12rpx rgba(0, 0, 0, 0.06);
}

/* ---- 输入字段 ---- */
.field-group {
  margin-bottom: 32rpx;
}

.field-label {
  display: block;
  font-size: 26rpx;
  font-weight: 500;
  color: #475569;
  margin-bottom: 10rpx;
}

.required-mark {
  color: #EF4444;
}

.optional-mark {
  color: #94A3B8;
  font-weight: 400;
  font-size: 22rpx;
}

.input-box {
  display: flex;
  align-items: center;
  height: 92rpx;
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
  font-size: 28rpx;
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

/* ---- 按钮 ---- */
.btn-primary {
  width: 100%;
  height: 96rpx;
  background-color: #2563EB;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 40rpx;
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
  margin-top: 36rpx;
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

/* ---- 底部留白 ---- */
.bottom-spacer {
  height: 80rpx;
}
</style>
