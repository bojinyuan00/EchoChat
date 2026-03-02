<!--
  管理员登录页
  
  设计系统：design-system/echochat/MASTER.md + pages/admin-login.md
  色板：Primary #2563EB / Background #F0F2F5 / Text #1E293B
  
  功能：
  - 管理员账号 + 密码登录
  - 使用 Element Plus 表单组件 + 校验
  - Loading 状态防重复提交
  - 登录成功跳转仪表盘
  
  对应 API：POST /api/v1/admin/auth/login
-->
<template>
  <div class="login-page">
    <div class="login-card">
      <!-- 品牌区域 -->
      <div class="brand-area">
        <div class="logo-icon">E</div>
        <h1 class="brand-title">EchoChat 管理后台</h1>
        <p class="brand-desc">管理员专用，请使用管理员账号登录</p>
      </div>

      <!-- 登录表单 -->
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" size="large">
        <el-form-item label="账号" prop="account">
          <el-input
            v-model="form.account"
            placeholder="请输入管理员账号"
            :prefix-icon="User"
            clearable
          />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            :prefix-icon="Lock"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            :loading="loading"
            class="login-btn"
            @click="handleLogin"
          >
            {{ loading ? '登录中…' : '登 录' }}
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="login-footer">
      <p>EchoChat &copy; 2026 · 实时音视频通讯平台</p>
    </div>
  </div>
</template>

<script setup>
/**
 * 管理员登录逻辑
 *
 * 使用 Element Plus 表单校验 + useUserStore 登录
 * 登录成功后跳转到 redirect 参数指定的页面，默认仪表盘
 */
import { ref, shallowRef } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { useUserStore } from '@/store/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const formRef = ref(null)
const loading = ref(false)

const form = ref({
  account: '',
  password: ''
})

/** 表单校验规则 */
const rules = {
  account: [{ required: true, message: '请输入管理员账号', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' }
  ]
}

/** 提交登录 */
const handleLogin = async () => {
  if (loading.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  loading.value = true
  try {
    await userStore.login({
      account: form.value.account.trim(),
      password: form.value.password
    })
    ElMessage.success('登录成功')
    const redirect = route.query.redirect || '/dashboard'
    router.push(redirect)
  } catch (err) {
    console.error('管理员登录失败:', err)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/*
 * 管理员登录页样式
 * 居中卡片布局，背景灰色
 * 品牌色：#2563EB
 */
.login-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #EFF6FF 0%, #F0F2F5 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: #FFFFFF;
  border-radius: 12px;
  padding: 40px 36px 32px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
}

/* ---- 品牌区域 ---- */
.brand-area {
  text-align: center;
  margin-bottom: 32px;
}

.logo-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  background-color: #2563EB;
  color: #FFFFFF;
  font-size: 24px;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
}

.brand-title {
  font-size: 22px;
  font-weight: 600;
  color: #1E293B;
  margin: 0 0 8px;
}

.brand-desc {
  font-size: 14px;
  color: #94A3B8;
  margin: 0;
}

/* ---- 登录按钮 ---- */
.login-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
  letter-spacing: 2px;
}

/* ---- 底部 ---- */
.login-footer {
  margin-top: 32px;
  text-align: center;
}

.login-footer p {
  font-size: 13px;
  color: #94A3B8;
}
</style>
