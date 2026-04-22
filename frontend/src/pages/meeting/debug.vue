<!--
  会议调试页（Task 9 临时产物）

  目的：
  - 在没有正式 UI 的情况下，手工验证 frontend/src/store/meeting.js 各 action 通链路
  - 仅 H5 端可用（mediasoup-client 平台限制，见 Task 9 决策 Q1）

  注意：
  - uni-app H5 的 <video>/<audio> 标签会被编译成 uni 自定义组件（不支持 srcObject）
  - 因此本页用 <view> 作为容器，在 H5 环境用原生 document.createElement('video'/'audio') 挂 WebRTC track

  Task 10 开始后此页可删除或重构为正式 UI，路由信息仅在 pages.json 的 debug 块中保留
-->
<template>
  <view class="page">
    <view class="header">
      <text class="title">会议调试（Task 9）</text>
      <view class="ws-status">
        <view class="ws-dot" :class="wsStatusClass"></view>
        <text class="ws-text">{{ wsStatusLabel }}</text>
        <text class="ws-flap" v-if="wsFlapping">⚠ 检测到异常重连（{{ wsFlapCount }} 次/{{ Math.round(wsFlapWindowMs / 1000) }}s），可能同账号在其他端登录</text>
      </view>
      <text class="state">状态：{{ localStateLabel }}</text>
      <text class="state">房号：{{ currentRoom?.room_code || '-' }} ({{ currentRoom?.title || '-' }})</text>
      <text class="state">是否主持人：{{ isHost ? '是' : '否' }}</text>
      <text class="state" v-if="lastEndedReason">结束原因：{{ lastEndedReason }}</text>
      <text class="state" v-if="lastAutoHostReason">自动转让：{{ lastAutoHostReason }}</text>
    </view>

    <view class="section">
      <text class="subtitle">1. 会议生命周期</text>
      <view class="row">
        <input v-model="form.title" class="input" placeholder="会议标题" />
        <button class="btn primary" :disabled="isInMeeting" @click="onCreate">创建并入会</button>
      </view>
      <view class="row">
        <input v-model="form.roomCode" class="input" placeholder="会议号（XXX-XXX-XXX 或 9 位）" />
        <input v-model="form.password" class="input password" placeholder="密码（可选）" />
        <button class="btn primary" :disabled="isInMeeting" @click="onJoin">加入会议</button>
      </view>
      <view class="row">
        <button class="btn" :disabled="!isInMeeting" @click="onLeave">离开会议</button>
        <button class="btn danger" :disabled="!isHost || !isInMeeting" @click="onEnd">结束会议（host）</button>
        <button class="btn" @click="onCleanupStale">清理遗留会议</button>
      </view>
      <text class="hint" v-if="lastHint">{{ lastHint }}</text>
    </view>

    <view class="section">
      <text class="subtitle">2. 本地推流</text>
      <view class="row">
        <button class="btn" :disabled="!isConnected" @click="onToggleAudio">
          {{ localAudioEnabled ? '关闭麦克风' : '开启麦克风' }}
        </button>
        <button class="btn" :disabled="!isConnected" @click="onToggleVideo">
          {{ localVideoEnabled ? '关闭摄像头' : '开启摄像头' }}
        </button>
      </view>
      <view class="video-wrap">
        <text class="caption">本地预览</text>
        <view ref="localVideoBox" class="video-slot"></view>
      </view>
    </view>

    <view class="section">
      <text class="subtitle">3. 远端参与者（{{ activeParticipants.length }} 人活跃）</text>
      <view v-for="p in activeParticipants" :key="p.user_id" class="participant">
        <text class="pname">userId={{ p.user_id }} role={{ p.role }} joined={{ p.joined_at }}</text>
        <view class="video-wrap">
          <view :ref="el => setRemoteBox(p.user_id, el)" class="video-slot small"></view>
        </view>
      </view>
    </view>

    <view class="section">
      <text class="subtitle">4. 会议内聊天</text>
      <view class="chat-box">
        <view v-for="(m, i) in chatMessages" :key="i" class="chat-msg">
          <text>{{ m.user_name || m.user_id }}：{{ m.content }}</text>
        </view>
      </view>
      <view class="row">
        <input v-model="chatInput" class="input" placeholder="输入聊天内容" />
        <button class="btn" :disabled="!isInMeeting" @click="onSendChat">发送</button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, reactive, computed, watch, onBeforeUnmount, onMounted, nextTick } from 'vue'
import { useMeetingStore } from '@/store/meeting'
import { MEETING_LOCAL_STATE_LABEL } from '@/constants/meeting'
import wsService from '@/services/websocket'
import { storeToRefs } from 'pinia'

const meetingStore = useMeetingStore()
const {
  localState,
  currentRoom,
  chatMessages,
  lastEndedReason,
  lastAutoHostReason,
  localAudioEnabled,
  localVideoEnabled,
  isInMeeting,
  isConnected,
  isHost,
  activeParticipants,
  remoteConsumers
} = storeToRefs(meetingStore)

const form = reactive({ title: '调试会议', roomCode: '', password: '' })
const chatInput = ref('')
const lastHint = ref('')

// 容器 ref（保存 Vue 组件实例或真实 DOM）
const localVideoBox = ref(null)
const remoteBoxes = reactive({})

const setRemoteBox = (userId, el) => {
  if (el) remoteBoxes[userId] = el
  else delete remoteBoxes[userId]
}

const localStateLabel = computed(() => {
  const label = MEETING_LOCAL_STATE_LABEL[localState.value] || localState.value
  return `${localState.value} (${label})`
})

// ==================== WS 状态指示灯 ====================

/** 'connected' | 'reconnecting' | 'disconnected' */
const wsStatus = ref('disconnected')
const wsFlapping = ref(false)
const wsFlapCount = ref(0)
const wsFlapWindowMs = ref(10000)
let _wsFlapResetTimer = null

const wsStatusClass = computed(() => `ws-dot-${wsStatus.value}`)
const wsStatusLabel = computed(() => {
  switch (wsStatus.value) {
    case 'connected': return 'WS 已连接'
    case 'reconnecting': return 'WS 重连中…'
    default: return 'WS 已断开'
  }
})

const _onWsConnected = () => { wsStatus.value = 'connected' }
const _onWsDisconnected = () => {
  wsStatus.value = wsService.isManualClose ? 'disconnected' : 'reconnecting'
}
const _onWsFlapping = (info) => {
  wsFlapping.value = true
  wsFlapCount.value = info?.count || 0
  wsFlapWindowMs.value = info?.windowMs || 10000
  if (_wsFlapResetTimer) clearTimeout(_wsFlapResetTimer)
  // 连续 20s 无新 flapping 信号时自动消隐
  _wsFlapResetTimer = setTimeout(() => { wsFlapping.value = false }, 20000)
}

onMounted(() => {
  // 页面加载时根据当前 WS 实例判断初始状态
  // #ifdef H5
  if (wsService.ws && wsService.ws.readyState === WebSocket.OPEN) {
    wsStatus.value = 'connected'
  } else if (wsService.ws && wsService.ws.readyState === WebSocket.CONNECTING) {
    wsStatus.value = 'reconnecting'
  } else {
    wsStatus.value = 'disconnected'
  }
  // #endif
  wsService.on('_connected', _onWsConnected)
  wsService.on('_disconnected', _onWsDisconnected)
  wsService.on('_flapping', _onWsFlapping)
})

// ==================== 原生 DOM 辅助（仅 H5）====================

/** 把 uni <view> 的 ref 统一转成真实 DOM 元素 */
const resolveDom = (refValue) => {
  if (!refValue) return null
  // #ifdef H5
  if (refValue instanceof HTMLElement) return refValue
  if (refValue.$el) return refValue.$el
  if (refValue.exposed && refValue.exposed.$el) return refValue.exposed.$el
  // #endif
  return null
}

/** 在容器内确保存在一个 <video> / <audio> 原生元素，并返回它 */
const ensureMediaEl = (container, tagName, className = '') => {
  // #ifdef H5
  const parent = resolveDom(container)
  if (!parent) return null
  let el = parent.querySelector(`:scope > ${tagName}`)
  if (!el) {
    el = document.createElement(tagName)
    el.autoplay = true
    el.playsInline = true
    if (tagName === 'video') {
      el.style.width = '100%'
      el.style.maxWidth = '240px'
      el.style.background = '#000'
      el.style.borderRadius = '4px'
    }
    if (className) el.className = className
    parent.appendChild(el)
  }
  return el
  // #endif
  // #ifndef H5
  return null
  // #endif
}

const removeMediaEl = (container, tagName) => {
  // #ifdef H5
  const parent = resolveDom(container)
  if (!parent) return
  const el = parent.querySelector(`:scope > ${tagName}`)
  if (el) {
    try { el.srcObject = null } catch {}
    parent.removeChild(el)
  }
  // #endif
}

// ==================== 本地/远端 track 挂载 ====================

const attachLocalTrack = () => {
  // #ifdef H5
  const track = meetingStore.getLocalTrack('video')
  if (track) {
    const el = ensureMediaEl(localVideoBox.value, 'video')
    if (el) {
      el.muted = true
      el.srcObject = new MediaStream([track])
    }
  } else {
    removeMediaEl(localVideoBox.value, 'video')
  }
  // #endif
}

const attachRemoteTracks = () => {
  // #ifdef H5
  Object.keys(remoteConsumers.value).forEach((uid) => {
    const box = remoteBoxes[uid]
    if (!box) return
    const videoTrack = meetingStore.getRemoteTrack(Number(uid), 'video')
    const audioTrack = meetingStore.getRemoteTrack(Number(uid), 'audio')

    if (videoTrack) {
      const videoEl = ensureMediaEl(box, 'video', 'remote-video')
      if (videoEl) videoEl.srcObject = new MediaStream([videoTrack])
    } else {
      removeMediaEl(box, 'video')
    }

    if (audioTrack) {
      const audioEl = ensureMediaEl(box, 'audio', 'remote-audio')
      if (audioEl) audioEl.srcObject = new MediaStream([audioTrack])
    } else {
      removeMediaEl(box, 'audio')
    }
  })
  // #endif
}

watch(localVideoEnabled, () => nextTick(attachLocalTrack))
watch(remoteConsumers, () => nextTick(attachRemoteTracks), { deep: true })
watch(activeParticipants, () => nextTick(attachRemoteTracks))

// ==================== Actions ====================

const handleStaleHint = (err) => {
  if (err && err.staleMeetingHint) {
    lastHint.value = '检测到僵尸参会残留：请点击"清理遗留会议"按钮后重试'
  }
}

const onCreate = async () => {
  lastHint.value = ''
  try {
    await meetingStore.createAndEnter({ title: form.title })
    uni.showToast({ title: '已创建并入会', icon: 'success' })
  } catch (err) {
    handleStaleHint(err)
    uni.showToast({ title: `创建失败：${err.message || JSON.stringify(err)}`, icon: 'none', duration: 3000 })
  }
}

const onJoin = async () => {
  lastHint.value = ''
  try {
    await meetingStore.joinAndEnter(form.roomCode, form.password)
    uni.showToast({ title: '已加入会议', icon: 'success' })
  } catch (err) {
    handleStaleHint(err)
    uni.showToast({ title: `加入失败：${err.message || JSON.stringify(err)}`, icon: 'none', duration: 3000 })
  }
}

const onCleanupStale = async () => {
  lastHint.value = ''
  try {
    const { cleanedCodes } = await meetingStore.cleanupStaleMeetings()
    if (cleanedCodes.length === 0) {
      lastHint.value = '无遗留活跃会议'
      uni.showToast({ title: '无遗留活跃会议', icon: 'none' })
    } else {
      lastHint.value = `已清理 ${cleanedCodes.length} 条：${cleanedCodes.join('、')}`
      uni.showToast({ title: `已清理 ${cleanedCodes.length} 条`, icon: 'success' })
    }
  } catch (err) {
    uni.showToast({ title: `清理失败：${err.message || JSON.stringify(err)}`, icon: 'none', duration: 3000 })
  }
}

const onLeave = async () => {
  await meetingStore.leave()
  uni.showToast({ title: '已离会', icon: 'success' })
}

const onEnd = async () => {
  try {
    await meetingStore.endMeeting()
  } catch (err) {
    uni.showToast({ title: `结束失败：${err.message || JSON.stringify(err)}`, icon: 'none', duration: 3000 })
  }
}

const onToggleAudio = async () => {
  try {
    if (localAudioEnabled.value) await meetingStore.stopLocalAudio()
    else await meetingStore.startLocalAudio()
  } catch (err) {
    uni.showToast({ title: `音频切换失败：${err.message}`, icon: 'none' })
  }
}

const onToggleVideo = async () => {
  try {
    if (localVideoEnabled.value) await meetingStore.stopLocalVideo()
    else await meetingStore.startLocalVideo()
  } catch (err) {
    uni.showToast({ title: `视频切换失败：${err.message}`, icon: 'none' })
  }
}

const onSendChat = async () => {
  const content = chatInput.value.trim()
  if (!content) return
  try {
    await meetingStore.sendChat(content)
    chatInput.value = ''
  } catch (err) {
    uni.showToast({ title: `发送失败：${err.message || JSON.stringify(err)}`, icon: 'none' })
  }
}

onBeforeUnmount(() => {
  wsService.off('_connected', _onWsConnected)
  wsService.off('_disconnected', _onWsDisconnected)
  wsService.off('_flapping', _onWsFlapping)
  if (_wsFlapResetTimer) clearTimeout(_wsFlapResetTimer)
  if (isInMeeting.value) {
    meetingStore.leave().catch(() => {})
  }
})
</script>

<style scoped>
.page { padding: 24rpx; background: #F8FAFC; min-height: 100vh; }
.header { background: #fff; padding: 24rpx; border-radius: 12rpx; margin-bottom: 24rpx; }
.title { font-size: 36rpx; font-weight: 600; color: #1E293B; display: block; margin-bottom: 12rpx; }
.state { font-size: 26rpx; color: #475569; display: block; margin-top: 4rpx; }
.section { background: #fff; padding: 24rpx; border-radius: 12rpx; margin-bottom: 24rpx; }
.subtitle { font-size: 30rpx; font-weight: 600; color: #1E293B; display: block; margin-bottom: 16rpx; }
.row { display: flex; align-items: center; gap: 16rpx; margin-bottom: 16rpx; flex-wrap: wrap; }
.input { flex: 1; min-width: 200rpx; border: 1px solid #CBD5E1; padding: 12rpx 16rpx; border-radius: 8rpx; font-size: 28rpx; }
.input.password { max-width: 200rpx; }
.btn { padding: 12rpx 24rpx; border-radius: 8rpx; background: #E2E8F0; color: #1E293B; font-size: 26rpx; }
.btn.primary { background: #3B82F6; color: #fff; }
.btn.danger { background: #EF4444; color: #fff; }
.btn[disabled] { opacity: 0.5; }
.video-wrap { display: flex; flex-direction: column; gap: 8rpx; margin-top: 16rpx; }
.caption { font-size: 22rpx; color: #64748B; }
.video-slot { width: 100%; min-height: 180rpx; background: #0B1220; border-radius: 8rpx; display: flex; align-items: center; justify-content: center; }
.video-slot.small { min-height: 140rpx; }
.participant { padding: 16rpx 0; border-bottom: 1px dashed #E2E8F0; }
.pname { font-size: 24rpx; color: #475569; }
.chat-box { max-height: 300rpx; overflow-y: auto; background: #F1F5F9; padding: 12rpx; border-radius: 8rpx; margin-bottom: 12rpx; }
.chat-msg { font-size: 26rpx; color: #1E293B; padding: 4rpx 0; }
.hint { font-size: 24rpx; color: #F59E0B; margin-top: 12rpx; display: block; }
.ws-status { display: flex; align-items: center; gap: 10rpx; margin: 8rpx 0 12rpx; flex-wrap: wrap; }
.ws-dot { width: 16rpx; height: 16rpx; border-radius: 50%; background: #94A3B8; box-shadow: 0 0 0 2rpx rgba(148,163,184,0.18); }
.ws-dot.ws-dot-connected { background: #10B981; box-shadow: 0 0 0 2rpx rgba(16,185,129,0.22); }
.ws-dot.ws-dot-reconnecting { background: #F59E0B; box-shadow: 0 0 0 2rpx rgba(245,158,11,0.22); animation: ws-pulse 1.2s ease-in-out infinite; }
.ws-dot.ws-dot-disconnected { background: #EF4444; box-shadow: 0 0 0 2rpx rgba(239,68,68,0.22); }
.ws-text { font-size: 24rpx; color: #334155; }
.ws-flap { font-size: 22rpx; color: #B45309; background: #FEF3C7; padding: 4rpx 10rpx; border-radius: 6rpx; }
@keyframes ws-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.35; } }
</style>
