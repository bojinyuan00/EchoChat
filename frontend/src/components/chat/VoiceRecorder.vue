<!--
  语音录制组件
  - 微信风格：长按录音 / 松手发送 / 上滑取消，最长 60 秒
  - 事件兼容：同时支持 touch（移动端真机）与 mouse（PC H5 调试）
  - 平台适配：
    • App/小程序：uni.getRecorderManager()（原生实现）
    • H5 端：MediaRecorder + getUserMedia（uni-app H5 不支持 getRecorderManager，自行实现）
-->
<template>
  <view class="voice-recorder">
    <view
      class="record-btn"
      :class="{ 'recording': isRecording, 'cancel-zone': isCancelZone }"
      @touchstart.prevent="onPressStart"
      @touchmove.prevent="onPressMove"
      @touchend.prevent="onPressEnd"
      @touchcancel.prevent="onPressEnd"
      @mousedown.prevent="onPressStart"
      @mousemove.prevent="onPressMove"
      @mouseup.prevent="onPressEnd"
      @mouseleave="onMouseLeave"
    >
      <text class="record-text">
        {{ isRecording ? (isCancelZone ? '松开取消' : '松开发送') : '按住 说话' }}
      </text>
    </view>
    <view v-if="isRecording" class="recording-overlay">
      <view class="recording-indicator">
        <view class="recording-icon" :class="{ 'cancel-icon': isCancelZone }">
          <uni-icons :type="isCancelZone ? 'close' : 'mic'" :size="40" color="#FFFFFF" />
        </view>
        <text class="recording-time">{{ recordingTime }}"</text>
        <text class="recording-tip">{{ isCancelZone ? '松开手指，取消发送' : '上滑取消发送' }}</text>
      </view>
    </view>
  </view>
</template>

<script>
import { ref, onBeforeUnmount } from 'vue'

/**
 * H5 端录音器实现（基于 MediaRecorder + getUserMedia）
 * 与 uni.getRecorderManager() 对齐接口：onStart / onStop / onError / start / stop
 */
class H5Recorder {
  constructor() {
    this._startCb = null
    this._stopCb = null
    this._errCb = null
    this._mediaRecorder = null
    this._stream = null
    this._chunks = []
    this._mimeType = ''
  }
  onStart(cb) { this._startCb = cb }
  onStop(cb) { this._stopCb = cb }
  onError(cb) { this._errCb = cb }
  async start() {
    try {
      if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
        this._errCb && this._errCb({ errMsg: '浏览器不支持录音（需 HTTPS 或 localhost）' })
        return
      }
      this._stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const candidates = [
        'audio/webm;codecs=opus',
        'audio/webm',
        'audio/mp4',
        'audio/ogg;codecs=opus'
      ]
      this._mimeType = candidates.find(m => {
        try { return typeof MediaRecorder !== 'undefined' && MediaRecorder.isTypeSupported(m) } catch { return false }
      }) || ''
      const opts = this._mimeType ? { mimeType: this._mimeType } : undefined
      this._mediaRecorder = new MediaRecorder(this._stream, opts)
      this._chunks = []
      this._mediaRecorder.ondataavailable = (e) => {
        if (e.data && e.data.size > 0) this._chunks.push(e.data)
      }
      this._mediaRecorder.onstart = () => {
        this._startCb && this._startCb()
      }
      this._mediaRecorder.onstop = () => {
        const mime = this._mediaRecorder?.mimeType || this._mimeType || 'audio/webm'
        const blob = new Blob(this._chunks, { type: mime })
        const tempFilePath = URL.createObjectURL(blob)
        if (this._stream) {
          this._stream.getTracks().forEach(t => t.stop())
          this._stream = null
        }
        this._stopCb && this._stopCb({
          tempFilePath,
          fileSize: blob.size,
          duration: 0,
          blob,
          mimeType: mime
        })
      }
      this._mediaRecorder.onerror = (e) => {
        this._errCb && this._errCb({ errMsg: e?.error?.message || '录音异常' })
      }
      this._mediaRecorder.start()
    } catch (err) {
      const msg = err && err.name === 'NotAllowedError'
        ? '麦克风权限被拒绝，请在浏览器地址栏允许访问麦克风'
        : (err?.message || '无法启动录音')
      this._errCb && this._errCb({ errMsg: msg })
    }
  }
  stop() {
    try {
      if (this._mediaRecorder && this._mediaRecorder.state !== 'inactive') {
        this._mediaRecorder.stop()
      }
    } catch (_) { /* ignore */ }
  }
}

/**
 * 跨端创建录音器：优先使用浏览器 MediaRecorder；否则回退到 uni.getRecorderManager
 */
function createRecorder() {
  // H5 优先走 MediaRecorder（uni-app H5 的 getRecorderManager 是 stub）
  if (typeof window !== 'undefined'
      && typeof window.MediaRecorder !== 'undefined'
      && navigator && navigator.mediaDevices) {
    return new H5Recorder()
  }
  // App / 小程序 等 uni 原生端
  if (typeof uni !== 'undefined' && typeof uni.getRecorderManager === 'function') {
    try {
      return uni.getRecorderManager()
    } catch (_) { /* fallthrough */ }
  }
  return null
}

export default {
  name: 'VoiceRecorder',
  emits: ['recorded'],
  setup(props, { emit }) {
    const isRecording = ref(false)
    const isCancelZone = ref(false)
    const recordingTime = ref(0)
    let recorder = null
    let timer = null
    let startY = 0
    // 防止 touchstart 与 mousedown 重复触发，以及 mouseleave 误停
    let pressing = false

    const getClientY = (e) => {
      if (e.touches && e.touches.length > 0) return e.touches[0].clientY
      if (e.changedTouches && e.changedTouches.length > 0) return e.changedTouches[0].clientY
      return typeof e.clientY === 'number' ? e.clientY : 0
    }

    const onPressStart = (e) => {
      if (pressing) return
      pressing = true
      startY = getClientY(e)
      isCancelZone.value = false
      recordingTime.value = 0

      recorder = createRecorder()
      if (!recorder) {
        pressing = false
        uni.showToast({ title: '当前环境不支持录音', icon: 'none' })
        return
      }
      recorder.onStart(() => {
        isRecording.value = true
        timer = setInterval(() => {
          recordingTime.value++
          if (recordingTime.value >= 60) {
            onPressEnd()
          }
        }, 1000)
      })
      recorder.onStop((res) => {
        clearInterval(timer)
        const wasRecording = isRecording.value
        isRecording.value = false
        pressing = false
        if (!wasRecording) return
        if (isCancelZone.value) return
        if (recordingTime.value < 1) {
          uni.showToast({ title: '录音时间太短', icon: 'none' })
          return
        }
        emit('recorded', {
          tempFilePath: res.tempFilePath,
          duration: recordingTime.value,
          fileSize: res.fileSize || 0,
          blob: res.blob || null,
          mimeType: res.mimeType || ''
        })
      })
      recorder.onError((err) => {
        clearInterval(timer)
        isRecording.value = false
        pressing = false
        uni.showToast({ title: err?.errMsg || '录音失败', icon: 'none' })
      })

      // uni 原生 recorder.start 接收参数对象；H5Recorder.start 忽略参数
      try {
        recorder.start({
          format: 'mp3',
          duration: 60000,
          sampleRate: 44100,
          numberOfChannels: 1
        })
      } catch (e) {
        pressing = false
        isRecording.value = false
        uni.showToast({ title: e?.message || '无法启动录音', icon: 'none' })
      }
    }

    const onPressMove = (e) => {
      if (!isRecording.value) return
      const y = getClientY(e)
      const deltaY = startY - y
      isCancelZone.value = deltaY > 80
    }

    const onPressEnd = () => {
      if (recorder && isRecording.value) {
        recorder.stop()
      } else {
        // 未 onStart 就抬起：重置 pressing 避免死锁
        pressing = false
      }
    }

    // 鼠标离开按钮（PC H5 兜底）：视为取消录音
    const onMouseLeave = () => {
      if (!isRecording.value) return
      isCancelZone.value = true
      onPressEnd()
    }

    onBeforeUnmount(() => {
      clearInterval(timer)
      if (recorder && isRecording.value) {
        isCancelZone.value = true
        recorder.stop()
      }
    })

    return {
      isRecording,
      isCancelZone,
      recordingTime,
      onPressStart,
      onPressMove,
      onPressEnd,
      onMouseLeave
    }
  }
}
</script>

<style scoped>
.voice-recorder {
  flex: 1;
}
.record-btn {
  height: 72rpx;
  border-radius: 36rpx;
  background-color: #F1F5F9;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 200ms;
  cursor: pointer;
  user-select: none;
}
.record-btn.recording {
  background-color: #DBEAFE;
}
.record-btn.cancel-zone {
  background-color: #FEE2E2;
}
.record-text {
  font-size: 28rpx;
  color: #64748B;
  user-select: none;
}
.recording-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  pointer-events: none;
}
.recording-indicator {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16rpx;
  padding: 40rpx 60rpx;
  background-color: rgba(0, 0, 0, 0.7);
  border-radius: 24rpx;
}
.recording-icon {
  width: 120rpx;
  height: 120rpx;
  border-radius: 60rpx;
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
}
.cancel-icon {
  background-color: #DC2626;
}
.recording-time {
  font-size: 40rpx;
  font-weight: 700;
  color: #FFFFFF;
}
.recording-tip {
  font-size: 24rpx;
  color: rgba(255, 255, 255, 0.7);
}
</style>
