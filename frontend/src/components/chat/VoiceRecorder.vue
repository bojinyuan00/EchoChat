<!--
  语音录制组件
  微信风格：长按录音 / 松手发送 / 上滑取消
  最长 60 秒，不足 1 秒提示太短
-->
<template>
  <view class="voice-recorder">
    <view
      class="record-btn"
      :class="{ 'recording': isRecording, 'cancel-zone': isCancelZone }"
      @touchstart.prevent="onTouchStart"
      @touchmove.prevent="onTouchMove"
      @touchend.prevent="onTouchEnd"
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

    const onTouchStart = (e) => {
      startY = e.touches[0].clientY
      isCancelZone.value = false
      recordingTime.value = 0

      recorder = uni.getRecorderManager()
      recorder.onStart(() => {
        isRecording.value = true
        timer = setInterval(() => {
          recordingTime.value++
          if (recordingTime.value >= 60) {
            onTouchEnd()
          }
        }, 1000)
      })
      recorder.onStop((res) => {
        clearInterval(timer)
        isRecording.value = false
        if (isCancelZone.value) return
        if (recordingTime.value < 1) {
          uni.showToast({ title: '录音时间太短', icon: 'none' })
          return
        }
        emit('recorded', {
          tempFilePath: res.tempFilePath,
          duration: recordingTime.value,
          fileSize: res.fileSize || 0
        })
      })
      recorder.onError(() => {
        clearInterval(timer)
        isRecording.value = false
        uni.showToast({ title: '录音失败', icon: 'none' })
      })

      recorder.start({
        format: 'mp3',
        duration: 60000,
        sampleRate: 44100,
        numberOfChannels: 1
      })
    }

    const onTouchMove = (e) => {
      if (!isRecording.value) return
      const deltaY = startY - e.touches[0].clientY
      isCancelZone.value = deltaY > 80
    }

    const onTouchEnd = () => {
      if (recorder && isRecording.value) {
        recorder.stop()
      }
    }

    onBeforeUnmount(() => {
      clearInterval(timer)
      if (recorder && isRecording.value) {
        isCancelZone.value = true
        recorder.stop()
      }
    })

    return { isRecording, isCancelZone, recordingTime, onTouchStart, onTouchMove, onTouchEnd }
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
