<!--
  语音消息组件
  显示播放按钮 + 波形条 + 时长
  宽度按时长线性缩放：最小 120rpx（1秒），最大 500rpx（60秒）
  同一时间只允许一个语音播放（全局单例）
-->
<template>
  <view v-if="msg.status === 2" class="recalled-text">消息已撤回</view>
  <view
    v-else
    class="msg-voice-wrap"
    :class="{ 'voice-self': isSelf, 'voice-playing': isPlaying }"
    :style="{ width: voiceWidth + 'rpx' }"
    @tap="onTogglePlay"
  >
    <view class="voice-icon" :class="{ 'voice-icon-self': isSelf }">
      <uni-icons :type="isPlaying ? 'sound-filled' : 'sound'" :size="18" :color="isSelf ? '#FFFFFF' : '#2563EB'" />
    </view>
    <view class="voice-bars">
      <view v-for="i in 5" :key="i" class="voice-bar" :class="{ 'bar-self': isSelf }" :style="{ height: barHeights[i-1] + 'rpx' }" />
    </view>
    <text class="voice-duration" :class="{ 'duration-self': isSelf }">{{ duration }}"</text>
  </view>
</template>

<script>
import { ref, computed, onBeforeUnmount } from 'vue'
import { parseExtra } from '@/utils/file'

let _globalAudio = null
let _globalPlayingId = null

export default {
  name: 'MsgVoice',
  props: {
    msg: { type: Object, required: true },
    isSelf: { type: Boolean, default: false }
  },
  setup(props) {
    const isPlaying = ref(false)

    const voiceData = computed(() => {
      const extra = parseExtra(props.msg.extra)
      return extra?.voice || {}
    })

    const duration = computed(() => voiceData.value.duration || 0)

    const voiceWidth = computed(() => {
      const d = duration.value
      const minW = 120, maxW = 500
      return Math.min(maxW, Math.max(minW, minW + (d / 60) * (maxW - minW)))
    })

    const barHeights = computed(() => {
      const base = [20, 30, 24, 32, 18]
      return base
    })

    const onTogglePlay = () => {
      const url = voiceData.value.url
      if (!url) return

      const msgId = props.msg.id || props.msg.client_msg_id

      if (_globalPlayingId === msgId && _globalAudio) {
        _globalAudio.stop()
        _globalAudio = null
        _globalPlayingId = null
        isPlaying.value = false
        return
      }

      if (_globalAudio) {
        _globalAudio.stop()
        _globalAudio = null
        _globalPlayingId = null
      }

      const audio = uni.createInnerAudioContext()
      audio.src = url
      audio.onPlay(() => {
        isPlaying.value = true
        _globalPlayingId = msgId
      })
      audio.onEnded(() => {
        isPlaying.value = false
        _globalAudio = null
        _globalPlayingId = null
      })
      audio.onError(() => {
        isPlaying.value = false
        _globalAudio = null
        _globalPlayingId = null
        uni.showToast({ title: '播放失败', icon: 'none' })
      })
      audio.play()
      _globalAudio = audio
    }

    onBeforeUnmount(() => {
      const msgId = props.msg.id || props.msg.client_msg_id
      if (_globalPlayingId === msgId && _globalAudio) {
        _globalAudio.stop()
        _globalAudio = null
        _globalPlayingId = null
      }
    })

    return { isPlaying, duration, voiceWidth, barHeights, onTogglePlay }
  }
}
</script>

<style scoped>
.msg-voice-wrap {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 4rpx 0;
  cursor: pointer;
}
.voice-icon {
  flex-shrink: 0;
}
.voice-bars {
  display: flex;
  align-items: center;
  gap: 6rpx;
  flex: 1;
}
.voice-bar {
  width: 6rpx;
  min-height: 12rpx;
  border-radius: 3rpx;
  background-color: #2563EB;
  transition: background-color 200ms;
}
.bar-self {
  background-color: rgba(255, 255, 255, 0.7);
}
.voice-playing .voice-bar {
  animation: voicePulse 0.8s ease-in-out infinite alternate;
}
.voice-duration {
  font-size: 24rpx;
  color: #64748B;
  flex-shrink: 0;
}
.duration-self {
  color: rgba(255, 255, 255, 0.9);
}
.recalled-text {
  font-size: 24rpx;
  color: #94A3B8;
  font-style: italic;
}

@keyframes voicePulse {
  0% { opacity: 0.4; }
  100% { opacity: 1; }
}
</style>
