<!--
  加入会议表单页（Task 10；Task 16 P0-4 修复密码明文走 URL 的问题）

  数据流：
    用户输入会议号（可粘贴带 / 不带连字符的 9 位数字）+ 可选密码 → 点击"下一步"
    → 把 { code, password } 写入 meetingStore.draftJoinPayload
    → uni.navigateTo('/pages/meeting/preview?mode=join&code=XXX-XXX-XXX')  // URL 不再带密码
    → 预览页从 draftJoinPayload 读密码并立即清空，再调用 joinAndEnter

  邀请链接支持（设计 §2.2.1）：
  - URL 参数 code=XXX-XXX-XXX（必传时直接回填）
  - 严禁 URL 参数 password=xxx（旧实现已下线；邀请链接走 token 机制，password 只在当前页面表单内）
-->
<template>
  <view class="page">
    <view class="container">
      <text class="title">加入会议</text>
      <text class="subtitle">输入主持人分享的 9 位会议号，下一步选择设备</text>

      <view class="card">
        <view class="field">
          <text class="label">会议号 <text class="required">*</text></text>
          <input
            v-model="rawCodeInput"
            class="input code-input"
            placeholder="XXX-XXX-XXX"
            maxlength="11"
            @input="onCodeInput"
          />
          <text class="hint" v-if="codeError">{{ codeError }}</text>
          <text class="hint neutral" v-else-if="formattedCode">已识别：{{ formattedCode }}</text>
        </view>

        <view class="field">
          <text class="label">会议密码（如有）</text>
          <input
            v-model="form.password"
            class="input"
            placeholder="若会议设置了密码请填写"
            maxlength="16"
          />
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
import { ref, reactive, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { useMeetingStore } from '@/store/meeting'

const meetingStore = useMeetingStore()
const rawCodeInput = ref('')
const form = reactive({ password: '' })

/** 去掉所有非数字字符，截至 9 位 */
const digitsOf = (s) => (s || '').replace(/\D/g, '').slice(0, 9)

/** 把 9 位数字格式化为 XXX-XXX-XXX（不足 9 位时按实际位数插连字符） */
const formatCode = (digits) => {
  if (digits.length <= 3) return digits
  if (digits.length <= 6) return `${digits.slice(0, 3)}-${digits.slice(3)}`
  return `${digits.slice(0, 3)}-${digits.slice(3, 6)}-${digits.slice(6)}`
}

const formattedCode = computed(() => {
  const d = digitsOf(rawCodeInput.value)
  return d.length === 9 ? formatCode(d) : ''
})

const codeError = computed(() => {
  const d = digitsOf(rawCodeInput.value)
  if (d.length === 0) return ''
  if (d.length < 9) return `还差 ${9 - d.length} 位`
  return ''
})

const isFormValid = computed(() => digitsOf(rawCodeInput.value).length === 9)

const onCodeInput = (e) => {
  // 同时兼容 uni-app 事件（e.detail.value）与原生 input 事件（e.target.value），自动化测试友好
  const raw = e?.detail?.value ?? e?.target?.value ?? rawCodeInput.value
  const d = digitsOf(raw)
  rawCodeInput.value = formatCode(d)
}

onLoad((query) => {
  if (query?.code) {
    const d = digitsOf(query.code)
    if (d.length === 9) rawCodeInput.value = formatCode(d)
  }
  // P0-4：邀请链接只允许带 code / token，不再接受 password query；历史 URL 里若仍有 password 一律忽略
})

const onNext = () => {
  if (!isFormValid.value) return
  const code = formattedCode.value
  meetingStore.draftJoinPayload = { code, password: form.password || '' }
  uni.navigateTo({
    url: `/pages/meeting/preview?mode=join&code=${code}`
  })
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
.hint { font-size: 22rpx; color: #EF4444; display: block; margin-top: 8rpx; }
.hint.neutral { color: #10B981; }

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
.code-input {
  font-size: 36rpx;
  letter-spacing: 4rpx;
  text-align: center;
  font-variant-numeric: tabular-nums;
}

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
.btn-secondary { background: #F1F5F9; color: #475569; }
.btn-primary { background: #3B82F6; color: #FFFFFF; }
.btn-primary[disabled] { background: #BFDBFE; color: #FFFFFF; }
</style>
