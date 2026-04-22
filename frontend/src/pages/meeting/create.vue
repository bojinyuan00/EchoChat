<!--
  创建会议表单页（Task 10）

  数据流：
    用户在本页填写 title/password/开关 → 点击"下一步"
    → 保存表单到 meetingStore.draftCreatePayload
    → uni.navigateTo('/pages/meeting/preview?mode=create')
    → 在预览页选定设备后真正调用 createAndEnter

  设计依据：
  - Phase 2e-2 设计 §8.1 路由表 + D04 决策（入会前强制经过设备预览页）
  - 本页不调用任何会议 API，无副作用；取消即返回
-->
<template>
  <view class="page">
    <view class="container">
      <text class="title">创建会议</text>
      <text class="subtitle">填写会议基本信息，下一步设置摄像头与麦克风</text>

      <view class="card">
        <view class="field">
          <text class="label">会议标题 <text class="required">*</text></text>
          <input
            v-model="form.title"
            class="input"
            placeholder="例如：周会同步（限 40 字）"
            maxlength="40"
          />
          <text class="hint" v-if="titleError">{{ titleError }}</text>
        </view>

        <view class="field">
          <text class="label">会议密码（可选）</text>
          <input
            v-model="form.password"
            class="input"
            placeholder="留空表示无密码；4-16 位数字/字母"
            maxlength="16"
          />
          <text class="hint" v-if="passwordError">{{ passwordError }}</text>
        </view>

        <view class="field switch-row">
          <view class="switch-label">
            <text class="label">入会默认静音</text>
            <text class="desc">新成员加入时麦克风默认关闭</text>
          </view>
          <switch :checked="form.enter_muted" @change="onToggleMuted" color="#3B82F6" />
        </view>

        <view class="field switch-row">
          <view class="switch-label">
            <text class="label">允许会议内聊天</text>
            <text class="desc">关闭后成员不能发送聊天消息</text>
          </view>
          <switch :checked="form.allow_chat" @change="onToggleChat" color="#3B82F6" />
        </view>
      </view>

      <view class="actions">
        <button class="btn btn-secondary" @click="onCancel">取消</button>
        <button class="btn btn-primary" :disabled="!isFormValid" @click="onNext">下一步：设备预览</button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { reactive, computed } from 'vue'
import { useMeetingStore } from '@/store/meeting'
import { useUserStore } from '@/store/user'

const meetingStore = useMeetingStore()
const userStore = useUserStore()

const defaultTitle = () => {
  const name = userStore.userInfo?.nickname || userStore.userInfo?.username || '我'
  return `${name}的会议`
}

const form = reactive({
  title: defaultTitle(),
  password: '',
  enter_muted: false,
  allow_chat: true
})

const titleError = computed(() => {
  const t = form.title.trim()
  if (!t) return '会议标题不能为空'
  if (t.length > 40) return '会议标题不能超过 40 字'
  return ''
})

const passwordError = computed(() => {
  const p = form.password
  if (!p) return ''
  if (p.length < 4 || p.length > 16) return '密码长度需为 4-16 位'
  if (!/^[A-Za-z0-9]+$/.test(p)) return '密码仅支持数字和英文字母'
  return ''
})

const isFormValid = computed(() => !titleError.value && !passwordError.value)

const onToggleMuted = (e) => { form.enter_muted = e.detail.value }
const onToggleChat = (e) => { form.allow_chat = e.detail.value }

const onNext = () => {
  if (!isFormValid.value) return
  meetingStore.draftCreatePayload = {
    title: form.title.trim(),
    password: form.password || undefined,
    enter_muted: form.enter_muted,
    allow_chat: form.allow_chat
  }
  uni.navigateTo({ url: '/pages/meeting/preview?mode=create' })
}

const onCancel = () => {
  uni.navigateBack()
}
</script>

<style scoped>
.page { min-height: 100vh; background: #F8FAFC; padding: 24rpx; }
.container { max-width: 680rpx; margin: 0 auto; }
.title { font-size: 40rpx; font-weight: 600; color: #0F172A; display: block; margin: 24rpx 0 8rpx; }
.subtitle { font-size: 26rpx; color: #64748B; display: block; margin-bottom: 32rpx; }

.card {
  background: #FFFFFF;
  border-radius: 16rpx;
  padding: 32rpx;
  box-shadow: 0 2rpx 8rpx rgba(15, 23, 42, 0.04);
}

.field { margin-bottom: 28rpx; }
.field:last-child { margin-bottom: 0; }
.label { font-size: 28rpx; color: #1E293B; font-weight: 500; display: block; margin-bottom: 12rpx; }
.required { color: #EF4444; font-weight: 600; }
.desc { font-size: 24rpx; color: #94A3B8; display: block; margin-top: 4rpx; }
.hint { font-size: 22rpx; color: #EF4444; display: block; margin-top: 8rpx; }

.input {
  width: 100%;
  padding: 20rpx 24rpx;
  background: #F1F5F9;
  border-radius: 12rpx;
  font-size: 28rpx;
  color: #0F172A;
  border: 1px solid transparent;
}
.input:focus { border-color: #3B82F6; background: #FFFFFF; }

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.switch-label { flex: 1; }

.actions {
  display: flex;
  gap: 24rpx;
  margin-top: 40rpx;
}
.btn {
  flex: 1;
  padding: 24rpx 0;
  border-radius: 12rpx;
  font-size: 30rpx;
  font-weight: 500;
  border: none;
}
.btn-secondary {
  background: #F1F5F9;
  color: #475569;
}
.btn-primary {
  background: #3B82F6;
  color: #FFFFFF;
}
.btn-primary[disabled] {
  background: #BFDBFE;
  color: #FFFFFF;
}
</style>
