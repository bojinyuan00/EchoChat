<!--
  会议视频块（Task 11）

  职责：
  - 挂载单个成员的 video/audio MediaStreamTrack（H5 下用原生 <video>/<audio>，因 uni <video> 不支持 srcObject）
  - 视频关闭时回落到头像占位
  - 右下角角标：昵称 + 麦克风静音图标 + 主持人徽章
  - 说话者流光边框（通过 isSpeaking prop 控制，实际探测留给上层）

  平台说明：主要支持 H5 / Web 浏览器，条件编译指令保留但本组件核心能力仅在 H5 生效
-->
<template>
  <view class="tile" :class="{ speaking: isSpeaking, local: isLocal, compact }">
    <view ref="mediaBox" class="media-slot"></view>

    <view v-if="!hasVideo" class="avatar-fallback">
      <view class="avatar-circle" :style="avatarStyle">{{ avatarInitial }}</view>
    </view>

    <view v-if="isLocal" class="tag tag-self">你</view>
    <view v-if="isHost" class="tag tag-host" title="主持人">主持人</view>

    <view class="footer">
      <text class="name">{{ name }}</text>
      <view class="icons">
        <view v-if="!audioEnabled" class="icon icon-mic-off" title="已静音">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="1" y1="1" x2="23" y2="23"></line>
            <path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"></path>
            <path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2a7 7 0 0 1-.11 1.23"></path>
            <line x1="12" y1="19" x2="12" y2="23"></line>
          </svg>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount, nextTick } from 'vue'

const props = defineProps({
  userId: { type: [Number, String], required: true },
  name: { type: String, default: '' },
  isLocal: { type: Boolean, default: false },
  isHost: { type: Boolean, default: false },
  audioEnabled: { type: Boolean, default: true },
  videoEnabled: { type: Boolean, default: true },
  audioTrack: { type: Object, default: null },
  videoTrack: { type: Object, default: null },
  isSpeaking: { type: Boolean, default: false },
  compact: { type: Boolean, default: false }
})

const mediaBox = ref(null)

const hasVideo = computed(() => !!props.videoTrack && props.videoEnabled)

const avatarInitial = computed(() => {
  const s = (props.name || '').trim()
  return s ? s.charAt(0).toUpperCase() : '?'
})

/** 基于 userId 稳定生成头像背景色（HSL 色相） */
const avatarStyle = computed(() => {
  const seed = String(props.userId || '0')
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) % 360
  return { background: `hsl(${h}, 60%, 48%)` }
})

/** 获取或创建容器下的原生 media 元素 */
const ensureMediaEl = (tagName) => {
  // #ifdef H5
  const parent = resolveDom(mediaBox.value)
  if (!parent) return null
  let el = parent.querySelector(`:scope > ${tagName}`)
  if (!el) {
    el = document.createElement(tagName)
    el.autoplay = true
    el.playsInline = true
    if (tagName === 'video') {
      el.style.width = '100%'
      el.style.height = '100%'
      el.style.objectFit = 'cover'
      el.style.background = '#0B1220'
    }
    parent.appendChild(el)
  }
  return el
  // #endif
  // #ifndef H5
  return null
  // #endif
}

const removeMediaEl = (tagName) => {
  // #ifdef H5
  const parent = resolveDom(mediaBox.value)
  if (!parent) return
  const el = parent.querySelector(`:scope > ${tagName}`)
  if (el) {
    try { el.srcObject = null } catch {}
    parent.removeChild(el)
  }
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

const applyVideo = () => {
  // #ifdef H5
  if (props.videoTrack && props.videoEnabled) {
    const el = ensureMediaEl('video')
    if (el) {
      el.muted = props.isLocal
      el.srcObject = new MediaStream([props.videoTrack])
    }
  } else {
    removeMediaEl('video')
  }
  // #endif
}

const applyAudio = () => {
  // #ifdef H5
  // 本地 tile 不挂载 audio（避免回声）
  if (!props.isLocal && props.audioTrack && props.audioEnabled) {
    const el = ensureMediaEl('audio')
    if (el) {
      el.srcObject = new MediaStream([props.audioTrack])
    }
  } else {
    removeMediaEl('audio')
  }
  // #endif
}

watch(() => [props.videoTrack, props.videoEnabled], () => nextTick(applyVideo), { immediate: true })
watch(() => [props.audioTrack, props.audioEnabled, props.isLocal], () => nextTick(applyAudio), { immediate: true })

onBeforeUnmount(() => {
  removeMediaEl('video')
  removeMediaEl('audio')
})
</script>

<style scoped>
.tile {
  position: relative;
  width: 100%;
  height: 100%;
  border-radius: 16rpx;
  overflow: hidden;
  background: #0B1220;
  box-shadow: 0 2rpx 10rpx rgba(0, 0, 0, 0.25);
  transition: box-shadow 0.18s ease, transform 0.18s ease;
  border: 2rpx solid transparent;
}
.tile.speaking {
  border-color: #3B82F6;
  box-shadow: 0 0 0 2rpx rgba(59, 130, 246, 0.18), 0 0 28rpx rgba(59, 130, 246, 0.35);
}
.tile.local::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  border-radius: 16rpx;
  box-shadow: inset 0 0 0 2rpx rgba(59, 130, 246, 0.45);
}

.media-slot {
  position: absolute;
  inset: 0;
}

.avatar-fallback {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.avatar-circle {
  width: 128rpx;
  height: 128rpx;
  border-radius: 50%;
  color: #FFFFFF;
  font-size: 52rpx;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  letter-spacing: 0;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.3);
}
.tile.compact .avatar-circle {
  width: 80rpx;
  height: 80rpx;
  font-size: 36rpx;
}

.tag {
  position: absolute;
  top: 16rpx;
  padding: 4rpx 12rpx;
  border-radius: 999rpx;
  font-size: 20rpx;
  line-height: 1;
  font-weight: 600;
  letter-spacing: 0.5rpx;
  color: #FFFFFF;
  backdrop-filter: blur(8px);
}
.tag-self {
  left: 16rpx;
  background: rgba(59, 130, 246, 0.7);
}
.tag-host {
  right: 16rpx;
  background: rgba(245, 158, 11, 0.85);
}

.footer {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 14rpx 18rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8rpx;
  background: linear-gradient(to top, rgba(0, 0, 0, 0.65), rgba(0, 0, 0, 0));
}
.name {
  color: #FFFFFF;
  font-size: 24rpx;
  font-weight: 500;
  text-shadow: 0 1rpx 2rpx rgba(0, 0, 0, 0.5);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}
.icons { display: flex; align-items: center; gap: 8rpx; flex-shrink: 0; }
.icon {
  width: 36rpx;
  height: 36rpx;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}
.icon-mic-off {
  background: rgba(239, 68, 68, 0.9);
  color: #FFFFFF;
}

.tile.compact .footer { padding: 8rpx 12rpx; }
.tile.compact .name { font-size: 20rpx; }
.tile.compact .tag { font-size: 18rpx; padding: 2rpx 8rpx; }
</style>
