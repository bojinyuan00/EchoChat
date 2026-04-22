<!--
  邀请弹窗（Task 11）

  职责：
  - 展示会议号（XXX-XXX-XXX 格式）、密码、加入链接
  - 提供三个一键复制入口：复制会议号、复制邀请文案、复制加入链接

  复制方案：H5 下使用 navigator.clipboard，失败时回落到 execCommand('copy')
-->
<template>
  <view v-if="visible" class="modal-root" @click.self="close">
    <view class="modal" :class="{ show: show }">
      <view class="head">
        <text class="title">邀请加入会议</text>
        <view class="btn-close" @click="close">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </view>
      </view>

      <view class="body">
        <view class="row">
          <text class="label">会议主题</text>
          <text class="value">{{ meetingTitle || '—' }}</text>
        </view>

        <view class="row code-row">
          <text class="label">会议号</text>
          <view class="code-block">
            <text class="code-text">{{ formattedCode }}</text>
            <button class="btn-copy" @click="copy(roomCode, '会议号')">复制</button>
          </view>
        </view>

        <view v-if="password" class="row">
          <text class="label">会议密码</text>
          <view class="code-block">
            <text class="code-text">{{ password }}</text>
            <button class="btn-copy" @click="copy(password, '密码')">复制</button>
          </view>
        </view>

        <view class="row">
          <text class="label">加入链接</text>
          <view class="link-block">
            <text class="link-text">{{ joinLink }}</text>
            <button class="btn-copy" @click="copy(joinLink, '链接')">复制</button>
          </view>
        </view>

        <view class="invite-area">
          <view class="invite-box">{{ inviteText }}</view>
          <button class="btn-primary" @click="copy(inviteText, '邀请信息')">一键复制邀请信息</button>
        </view>

        <view v-if="toast" class="toast">{{ toast }}</view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  roomCode: { type: String, default: '' },
  password: { type: String, default: '' },
  meetingTitle: { type: String, default: '' },
  joinLink: { type: String, default: '' }
})

const emit = defineEmits(['close'])

const show = ref(false)
const toast = ref('')

watch(() => props.visible, (v) => {
  if (v) {
    setTimeout(() => { show.value = true }, 10)
  } else {
    show.value = false
    toast.value = ''
  }
}, { immediate: true })

const formattedCode = computed(() => {
  const digits = String(props.roomCode || '').replace(/\D/g, '')
  if (digits.length !== 9) return digits
  return `${digits.slice(0, 3)}-${digits.slice(3, 6)}-${digits.slice(6)}`
})

const inviteText = computed(() => {
  const lines = [
    `邀请你参加 EchoChat 会议`,
    `主题：${props.meetingTitle || '会议'}`,
    `会议号：${formattedCode.value}`
  ]
  if (props.password) lines.push(`密码：${props.password}`)
  lines.push(`点击加入：${props.joinLink}`)
  return lines.join('\n')
})

const showToast = (msg) => {
  toast.value = msg
  setTimeout(() => {
    if (toast.value === msg) toast.value = ''
  }, 1600)
}

/** 统一复制工具：优先 Clipboard API，失败降级到 textarea + execCommand */
const copy = async (text, label) => {
  const data = String(text ?? '')
  if (!data) return
  // #ifdef H5
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(data)
      showToast(`${label}已复制`)
      return
    }
  } catch {}
  try {
    const ta = document.createElement('textarea')
    ta.value = data
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
    showToast(`${label}已复制`)
  } catch {
    showToast('复制失败，请手动选择')
  }
  // #endif
  // #ifndef H5
  uni.setClipboardData({
    data,
    success: () => showToast(`${label}已复制`),
    fail: () => showToast('复制失败')
  })
  // #endif
}

const close = () => emit('close')
</script>

<style scoped>
.modal-root {
  position: fixed;
  inset: 0;
  z-index: 210;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32rpx;
  box-sizing: border-box;
}

.modal {
  width: 100%;
  max-width: 640rpx;
  background: #FFFFFF;
  border-radius: 24rpx;
  overflow: hidden;
  transform: translateY(16rpx) scale(0.96);
  opacity: 0;
  transition: all 0.2s ease;
  box-shadow: 0 24rpx 48rpx rgba(0, 0, 0, 0.35);
}
.modal.show { transform: translateY(0) scale(1); opacity: 1; }

.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 28rpx 32rpx;
  border-bottom: 1rpx solid #F3F4F6;
}
.title {
  font-size: 32rpx;
  font-weight: 600;
  color: #111827;
}
.btn-close {
  width: 56rpx;
  height: 56rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: #6B7280;
  cursor: pointer;
}
.btn-close:hover { background: #F3F4F6; }

.body {
  padding: 24rpx 32rpx 32rpx;
  display: flex;
  flex-direction: column;
  gap: 22rpx;
  position: relative;
}

.row {
  display: flex;
  flex-direction: column;
  gap: 10rpx;
}
.label {
  font-size: 24rpx;
  color: #6B7280;
  letter-spacing: 0.5rpx;
}
.value {
  font-size: 28rpx;
  color: #111827;
  font-weight: 500;
}

.code-block, .link-block {
  display: flex;
  align-items: center;
  gap: 12rpx;
  background: #F9FAFB;
  border: 1rpx solid #E5E7EB;
  border-radius: 14rpx;
  padding: 14rpx 18rpx;
}

.code-text {
  flex: 1;
  font-size: 36rpx;
  font-weight: 600;
  color: #111827;
  letter-spacing: 2rpx;
  font-family: 'SF Mono', 'Menlo', monospace;
}

.link-text {
  flex: 1;
  font-size: 24rpx;
  color: #2563EB;
  word-break: break-all;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btn-copy {
  all: unset;
  cursor: pointer;
  flex-shrink: 0;
  padding: 8rpx 20rpx;
  background: #EFF6FF;
  color: #2563EB;
  font-size: 24rpx;
  font-weight: 600;
  border-radius: 999rpx;
  transition: background-color 0.12s ease;
}
.btn-copy:hover { background: #DBEAFE; }
.btn-copy:active { background: #BFDBFE; }

.invite-area {
  margin-top: 8rpx;
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}
.invite-box {
  background: #F3F4F6;
  color: #374151;
  padding: 20rpx;
  border-radius: 14rpx;
  font-size: 24rpx;
  line-height: 1.6;
  white-space: pre-wrap;
  max-height: 300rpx;
  overflow-y: auto;
}
.btn-primary {
  all: unset;
  cursor: pointer;
  text-align: center;
  background: #3B82F6;
  color: #FFFFFF;
  padding: 20rpx;
  border-radius: 14rpx;
  font-size: 28rpx;
  font-weight: 600;
  letter-spacing: 1rpx;
  transition: background-color 0.12s ease;
}
.btn-primary:hover { background: #2563EB; }
.btn-primary:active { background: #1D4ED8; }

.toast {
  position: absolute;
  bottom: 140rpx;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(17, 24, 39, 0.92);
  color: #FFFFFF;
  padding: 12rpx 28rpx;
  border-radius: 999rpx;
  font-size: 24rpx;
  pointer-events: none;
  animation: toast-in 0.2s ease;
}
@keyframes toast-in {
  from { opacity: 0; transform: translate(-50%, 10rpx); }
  to { opacity: 1; transform: translate(-50%, 0); }
}
</style>
