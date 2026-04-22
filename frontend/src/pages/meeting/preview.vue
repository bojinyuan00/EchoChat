<!--
  设备预览页（Task 10）

  来自：
  - /pages/meeting/create（mode=create）→ 读 meetingStore.draftCreatePayload 调 createAndEnter
  - /pages/meeting/join（mode=join&code=...&password=...）→ 调 joinAndEnter

  核心能力：
  - navigator.mediaDevices.enumerateDevices() 列出摄像头/麦克风/扬声器
  - getUserMedia({ audio, video }) 本地画面 + 音频轨预览
  - AudioContext + AnalyserNode 实时音量条（20Hz 采样）
  - 「入会默认开麦/摄像头」两个开关写入 mediaPrefs
  - 兜底：权限被拒 / 浏览器不支持 WebRTC 时给明确错误提示

  入会成功后：uni.redirectTo('/pages/meeting/room?code=xxx')
-->
<template>
  <view class="page">
    <view class="container">
      <text class="title">设备预览</text>
      <text class="subtitle">{{ modeLabel }}，确认设备和显示名称后加入会议</text>

      <view class="content">
        <view class="preview-box">
          <view ref="videoBox" class="video-slot"></view>
          <view class="preview-footer">
            <view class="volume-wrap">
              <text class="volume-label">麦克风音量</text>
              <view class="volume-bar"><view class="volume-fill" :style="{ width: volumePercent + '%' }"></view></view>
            </view>
            <text v-if="permissionError" class="error">{{ permissionError }}</text>
          </view>
        </view>

        <view class="controls card">
          <view class="field">
            <text class="label">摄像头</text>
            <picker
              :value="videoPickerIndex"
              :range="videoDeviceLabels"
              :disabled="!hasPermission || videoDevices.length === 0"
              @change="onVideoChange"
            >
              <view class="picker">{{ videoDeviceLabels[videoPickerIndex] || '未检测到摄像头' }}</view>
            </picker>
          </view>

          <view class="field">
            <text class="label">麦克风</text>
            <picker
              :value="audioPickerIndex"
              :range="audioDeviceLabels"
              :disabled="!hasPermission || audioDevices.length === 0"
              @change="onAudioChange"
            >
              <view class="picker">{{ audioDeviceLabels[audioPickerIndex] || '未检测到麦克风' }}</view>
            </picker>
          </view>

          <view class="field" v-if="speakerDevices.length > 0">
            <text class="label">扬声器</text>
            <picker
              :value="speakerPickerIndex"
              :range="speakerDeviceLabels"
              @change="onSpeakerChange"
            >
              <view class="picker">{{ speakerDeviceLabels[speakerPickerIndex] || '系统默认' }}</view>
            </picker>
          </view>

          <view class="field">
            <text class="label">显示名称</text>
            <input v-model="displayName" class="input" placeholder="会议中显示的名字" maxlength="20" />
          </view>

          <view class="field switch-row">
            <view class="switch-label">
              <text class="label">入会默认开启摄像头</text>
            </view>
            <switch :checked="mediaPrefs.startVideo" @change="onToggleVideoDefault" color="#3B82F6" />
          </view>

          <view class="field switch-row">
            <view class="switch-label">
              <text class="label">入会默认开启麦克风</text>
            </view>
            <switch :checked="mediaPrefs.startAudio" @change="onToggleAudioDefault" color="#3B82F6" />
          </view>
        </view>
      </view>

      <view class="actions">
        <button class="btn btn-secondary" :disabled="joining" @click="onCancel">上一步</button>
        <button class="btn btn-primary" :disabled="joining || !canJoin" @click="onJoin">
          {{ joining ? '加入中…' : '加入会议' }}
        </button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, reactive, computed, onBeforeUnmount, nextTick } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { useMeetingStore } from '@/store/meeting'
import { useUserStore } from '@/store/user'

const meetingStore = useMeetingStore()
const userStore = useUserStore()

const mode = ref('create')
const joinCode = ref('')
const joinPassword = ref('')

const modeLabel = computed(() => mode.value === 'create' ? '即将创建新会议' : `即将加入会议 ${joinCode.value}`)

const videoBox = ref(null)
let previewStream = null
let audioContext = null
let analyser = null
let volumeRafId = null

const hasPermission = ref(false)
const permissionError = ref('')
const joining = ref(false)

const videoDevices = ref([])
const audioDevices = ref([])
const speakerDevices = ref([])

const selectedVideoId = ref('')
const selectedAudioId = ref('')
const selectedSpeakerId = ref('')

const videoDeviceLabels = computed(() => videoDevices.value.map((d, i) => d.label || `摄像头 ${i + 1}`))
const audioDeviceLabels = computed(() => audioDevices.value.map((d, i) => d.label || `麦克风 ${i + 1}`))
const speakerDeviceLabels = computed(() => speakerDevices.value.map((d, i) => d.label || `扬声器 ${i + 1}`))

const videoPickerIndex = computed(() => {
  const i = videoDevices.value.findIndex(d => d.deviceId === selectedVideoId.value)
  return i < 0 ? 0 : i
})
const audioPickerIndex = computed(() => {
  const i = audioDevices.value.findIndex(d => d.deviceId === selectedAudioId.value)
  return i < 0 ? 0 : i
})
const speakerPickerIndex = computed(() => {
  const i = speakerDevices.value.findIndex(d => d.deviceId === selectedSpeakerId.value)
  return i < 0 ? 0 : i
})

const displayName = ref(userStore.userInfo?.nickname || userStore.userInfo?.username || '')
const mediaPrefs = reactive({
  startAudio: true,
  startVideo: true
})

const volumePercent = ref(0)

const canJoin = computed(() => {
  if (mode.value === 'join' && !joinCode.value) return false
  return true
})

// ==================== 设备 / 预览流 ====================

const stopPreviewStream = () => {
  if (previewStream) {
    previewStream.getTracks().forEach(t => { try { t.stop() } catch {} })
    previewStream = null
  }
}

const stopAudioMeter = () => {
  if (volumeRafId) {
    cancelAnimationFrame(volumeRafId)
    volumeRafId = null
  }
  if (audioContext) {
    try { audioContext.close() } catch {}
    audioContext = null
    analyser = null
  }
  volumePercent.value = 0
}

/** 基于当前已选 deviceId 启动预览（无 ID 则首次调用由浏览器分配默认设备） */
const startPreview = async () => {
  // #ifdef H5
  if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
    permissionError.value = '当前浏览器不支持音视频预览（需 HTTPS + WebRTC）'
    return
  }
  stopPreviewStream()
  stopAudioMeter()
  try {
    const constraints = {
      audio: selectedAudioId.value ? { deviceId: { exact: selectedAudioId.value } } : true,
      video: selectedVideoId.value
        ? { deviceId: { exact: selectedVideoId.value }, width: 640, height: 480 }
        : { width: 640, height: 480 }
    }
    previewStream = await navigator.mediaDevices.getUserMedia(constraints)
    hasPermission.value = true
    permissionError.value = ''
    await nextTick()
    mountPreviewVideo()
    startAudioMeter(previewStream)
  } catch (err) {
    hasPermission.value = false
    permissionError.value = err?.name === 'NotAllowedError'
      ? '已拒绝摄像头/麦克风权限，请在浏览器地址栏左侧恢复权限后重试'
      : `无法访问音视频设备：${err?.message || err}`
    console.warn('[Preview] getUserMedia 失败', err)
  }
  // #endif
}

const mountPreviewVideo = () => {
  // #ifdef H5
  const parent = resolveDom(videoBox.value)
  if (!parent || !previewStream) return
  let videoEl = parent.querySelector(':scope > video')
  if (!videoEl) {
    videoEl = document.createElement('video')
    videoEl.autoplay = true
    videoEl.playsInline = true
    videoEl.muted = true // 预览时本地回声静音
    videoEl.style.width = '100%'
    videoEl.style.height = '100%'
    videoEl.style.objectFit = 'cover'
    videoEl.style.background = '#000'
    videoEl.style.borderRadius = '12rpx'
    parent.appendChild(videoEl)
  }
  videoEl.srcObject = previewStream
  // #endif
}

const resolveDom = (refValue) => {
  if (!refValue) return null
  // #ifdef H5
  if (refValue instanceof HTMLElement) return refValue
  if (refValue.$el) return refValue.$el
  if (refValue.exposed && refValue.exposed.$el) return refValue.exposed.$el
  // #endif
  return null
}

/** 用 AudioContext.AnalyserNode 采样 RMS，换算成 0-100 的音量条 */
const startAudioMeter = (stream) => {
  // #ifdef H5
  const audioTrack = stream.getAudioTracks()[0]
  if (!audioTrack) return
  try {
    const AC = window.AudioContext || window.webkitAudioContext
    if (!AC) return
    audioContext = new AC()
    const source = audioContext.createMediaStreamSource(new MediaStream([audioTrack]))
    analyser = audioContext.createAnalyser()
    analyser.fftSize = 1024
    source.connect(analyser)
    const data = new Uint8Array(analyser.fftSize)
    const loop = () => {
      if (!analyser) return
      analyser.getByteTimeDomainData(data)
      // RMS：((sample-128)/128) 的平方平均
      let sum = 0
      for (let i = 0; i < data.length; i++) {
        const v = (data[i] - 128) / 128
        sum += v * v
      }
      const rms = Math.sqrt(sum / data.length)
      // 经验映射：放大 1.6 倍，截顶 100
      volumePercent.value = Math.min(100, Math.round(rms * 160))
      volumeRafId = requestAnimationFrame(loop)
    }
    loop()
  } catch (err) {
    console.warn('[Preview] AudioMeter 失败', err)
  }
  // #endif
}

const loadDevices = async () => {
  // #ifdef H5
  if (!navigator.mediaDevices?.enumerateDevices) return
  const list = await navigator.mediaDevices.enumerateDevices()
  videoDevices.value = list.filter(d => d.kind === 'videoinput')
  audioDevices.value = list.filter(d => d.kind === 'audioinput')
  speakerDevices.value = list.filter(d => d.kind === 'audiooutput')
  // 有权限后 label 才会返回，没默认选中时自动选第一个
  if (!selectedVideoId.value && videoDevices.value[0]) selectedVideoId.value = videoDevices.value[0].deviceId
  if (!selectedAudioId.value && audioDevices.value[0]) selectedAudioId.value = audioDevices.value[0].deviceId
  if (!selectedSpeakerId.value && speakerDevices.value[0]) selectedSpeakerId.value = speakerDevices.value[0].deviceId
  // #endif
}

// ==================== 事件 ====================

const onVideoChange = (e) => {
  const idx = Number(e.detail.value)
  const d = videoDevices.value[idx]
  if (!d) return
  selectedVideoId.value = d.deviceId
  startPreview()
}
const onAudioChange = (e) => {
  const idx = Number(e.detail.value)
  const d = audioDevices.value[idx]
  if (!d) return
  selectedAudioId.value = d.deviceId
  startPreview()
}
const onSpeakerChange = (e) => {
  const idx = Number(e.detail.value)
  const d = speakerDevices.value[idx]
  if (!d) return
  selectedSpeakerId.value = d.deviceId
}

const onToggleVideoDefault = (e) => { mediaPrefs.startVideo = e.detail.value }
const onToggleAudioDefault = (e) => { mediaPrefs.startAudio = e.detail.value }

const onCancel = () => {
  uni.navigateBack()
}

const onJoin = async () => {
  if (joining.value || !canJoin.value) return
  joining.value = true

  meetingStore.devicePreview = {
    audioDeviceId: selectedAudioId.value,
    videoDeviceId: selectedVideoId.value,
    speakerDeviceId: selectedSpeakerId.value,
    displayName: displayName.value,
    startAudio: mediaPrefs.startAudio,
    startVideo: mediaPrefs.startVideo
  }

  // 释放预览流，避免与 mediasoup produce 争用摄像头（浏览器单轨互斥）
  stopPreviewStream()
  stopAudioMeter()

  const prefs = {
    startAudio: mediaPrefs.startAudio,
    startVideo: mediaPrefs.startVideo,
    audioDeviceId: selectedAudioId.value,
    videoDeviceId: selectedVideoId.value
  }

  try {
    if (mode.value === 'create') {
      const payload = meetingStore.draftCreatePayload
      if (!payload) {
        throw new Error('未找到创建会议的表单草稿，请返回创建页重新填写')
      }
      await meetingStore.createAndEnter(payload, prefs)
      meetingStore.draftCreatePayload = null
    } else {
      await meetingStore.joinAndEnter(joinCode.value, joinPassword.value, prefs)
    }
    const roomCode = meetingStore.currentRoom?.room_code
    uni.redirectTo({ url: `/pages/meeting/room${roomCode ? `?code=${roomCode}` : ''}` })
  } catch (err) {
    const msg = err?.message || JSON.stringify(err)
    uni.showToast({ title: `加入失败：${msg}`, icon: 'none', duration: 3500 })
    console.error('[Preview] 加入会议失败', err)
  } finally {
    joining.value = false
  }
}

// ==================== 生命周期 ====================

onLoad(async (query) => {
  mode.value = query?.mode === 'join' ? 'join' : 'create'
  joinCode.value = query?.code || ''
  joinPassword.value = query?.password ? decodeURIComponent(query.password) : ''

  // 先请求权限并预览，默认拿到第一个设备
  await startPreview()
  // 权限到手后枚举设备拿到 label
  await loadDevices()
  // 若此时已选中某个设备，重新启动预览确保走明确 deviceId（避免浏览器默认设备切换）
  if (hasPermission.value && (selectedVideoId.value || selectedAudioId.value)) {
    await startPreview()
  }
})

onBeforeUnmount(() => {
  stopPreviewStream()
  stopAudioMeter()
})
</script>

<style scoped>
.page { min-height: 100vh; background: #F8FAFC; padding: 24rpx; }
.container { max-width: 960rpx; margin: 0 auto; }
.title { font-size: 40rpx; font-weight: 600; color: #0F172A; display: block; margin: 24rpx 0 8rpx; }
.subtitle { font-size: 26rpx; color: #64748B; display: block; margin-bottom: 24rpx; }

.content {
  display: flex;
  gap: 24rpx;
  flex-wrap: wrap;
}
.preview-box {
  flex: 1.4 1 400rpx;
  min-width: 360rpx;
  background: #FFFFFF;
  border-radius: 16rpx;
  padding: 20rpx;
  box-shadow: 0 2rpx 8rpx rgba(15, 23, 42, 0.04);
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}
.video-slot {
  width: 100%;
  aspect-ratio: 4 / 3;
  min-height: 360rpx;
  background: #0B1220;
  border-radius: 12rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
.preview-footer { display: flex; flex-direction: column; gap: 8rpx; }
.volume-wrap { display: flex; flex-direction: column; gap: 6rpx; }
.volume-label { font-size: 22rpx; color: #64748B; }
.volume-bar {
  width: 100%;
  height: 10rpx;
  background: #E2E8F0;
  border-radius: 6rpx;
  overflow: hidden;
}
.volume-fill {
  height: 100%;
  background: linear-gradient(90deg, #10B981 0%, #3B82F6 60%, #F59E0B 90%);
  transition: width 0.08s linear;
}
.error { color: #EF4444; font-size: 22rpx; }

.controls {
  flex: 1 1 300rpx;
  min-width: 280rpx;
  background: #FFFFFF;
  border-radius: 16rpx;
  padding: 32rpx;
  box-shadow: 0 2rpx 8rpx rgba(15, 23, 42, 0.04);
}
.field { margin-bottom: 24rpx; }
.field:last-child { margin-bottom: 0; }
.label { font-size: 28rpx; color: #1E293B; font-weight: 500; display: block; margin-bottom: 10rpx; }
.picker, .input {
  width: 100%;
  padding: 18rpx 24rpx;
  background: #F1F5F9;
  border-radius: 12rpx;
  font-size: 28rpx;
  color: #0F172A;
  border: 1px solid transparent;
  box-sizing: border-box;
}
.picker {
  min-height: 68rpx;
  line-height: 32rpx;
  display: flex;
  align-items: center;
}
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.switch-label { flex: 1; }

.actions { display: flex; gap: 24rpx; margin-top: 32rpx; }
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
.btn-primary[disabled], .btn-secondary[disabled] { opacity: 0.6; }
</style>
