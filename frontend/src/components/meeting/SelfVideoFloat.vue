<!--
  自视频浮窗（Task 15 原创特色 3：四角吸附拖拽 + 图钉切回网格）

  职责：
  - 桌面端恒为浮窗显示本地视频（默认右下角，可拖拽到任意位置，松手吸附到最近四角）
  - 提供图钉按钮切换 float ↔ grid 模式
  - 复用 VideoTile 的 track 挂载能力，只在外壳上做拖拽逻辑

  设计要点：
  - 仅 H5 生效；非 H5 直接渲染为空
  - mousemove/touchmove 注册在 window 上，避免 iframe / 外部元素失焦
  - onBeforeUnmount 必须移除监听，防止泄漏
  - 吸附计算：记录容器边界矩形，释放时分别算到 4 角中心距离，取最小
  - z-index: 120（低于 Toolbar 210、低于 MemberPanel 200，高于 VideoGrid）
-->
<template>
  <view
    v-if="!hidden"
    ref="floatEl"
    class="float-shell"
    :class="{ dragging: isDragging }"
    :style="positionStyle"
    @mousedown="onDragStart"
    @touchstart="onDragStart"
  >
    <VideoTile
      :user-id="userId"
      :name="name"
      :is-local="true"
      :is-host="isHost"
      :audio-enabled="audioEnabled"
      :video-enabled="videoEnabled"
      :audio-track="audioTrack"
      :video-track="videoTrack"
      :is-speaking="isSpeaking"
      :compact="true"
    />

    <view class="pin-btn" title="收回网格" @click.stop="onPinClick" @mousedown.stop @touchstart.stop>
      <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="12" y1="17" x2="12" y2="22"></line>
        <path d="M5 17h14l-1.405-1.405A2 2 0 0 1 17 14.172V11a5 5 0 0 0-10 0v3.172a2 2 0 0 1-.595 1.423L5 17z"></path>
      </svg>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import VideoTile from '@/components/meeting/VideoTile.vue'

const props = defineProps({
  userId: { type: [Number, String], required: true },
  name: { type: String, default: '你' },
  isHost: { type: Boolean, default: false },
  audioEnabled: { type: Boolean, default: true },
  videoEnabled: { type: Boolean, default: true },
  audioTrack: { type: Object, default: null },
  videoTrack: { type: Object, default: null },
  isSpeaking: { type: Boolean, default: false },
  /** 父容器引用，用于计算吸附边界；不传则以 window 为边界 */
  container: { type: Object, default: null },
  /** 是否隐藏浮窗（例如图钉切换后） */
  hidden: { type: Boolean, default: false }
})

const emit = defineEmits(['pin-click'])

const floatEl = ref(null)

// ============ 位置状态 ============

/** 浮窗当前位置（绝对像素 / 容器坐标系）；默认右下角 */
const position = ref({ x: 0, y: 0 })
const isDragging = ref(false)

/** 拖拽起始记录 */
const dragStart = ref({ pointerX: 0, pointerY: 0, startX: 0, startY: 0 })

/** 浮窗尺寸：桌面端（宽屏 >= 900px）280x180px，移动端 / 窄屏 160x100px */
const getDefaultSize = () => {
  // #ifdef H5
  const isDesktop = typeof window !== 'undefined' && window.innerWidth >= 900
  return isDesktop ? { width: 280, height: 180 } : { width: 160, height: 100 }
  // #endif
  // #ifndef H5
  return { width: 160, height: 100 }
  // #endif
}
const SIZE = getDefaultSize()
const EDGE_MARGIN = 24

const positionStyle = computed(() => ({
  transform: `translate(${position.value.x}px, ${position.value.y}px)`,
  width: `${SIZE.width}px`,
  height: `${SIZE.height}px`,
  transition: isDragging.value ? 'none' : 'transform 0.22s cubic-bezier(0.2, 0.8, 0.2, 1)'
}))

// ============ 计算父容器边界 ============

const getBounds = () => {
  // #ifdef H5
  const el = resolveDom(props.container) || document.body
  const rect = el.getBoundingClientRect()
  return {
    left: 0,
    top: 0,
    right: rect.width,
    bottom: rect.height
  }
  // #endif
  // #ifndef H5
  return { left: 0, top: 0, right: 320, bottom: 240 }
  // #endif
}

const resolveDom = (r) => {
  // #ifdef H5
  if (!r) return null
  if (r.$el) return r.$el
  if (r instanceof HTMLElement) return r
  // #endif
  return null
}

/** 初始化位置：右下角 */
const placeDefault = () => {
  const b = getBounds()
  position.value = {
    x: b.right - SIZE.width - EDGE_MARGIN,
    y: b.bottom - SIZE.height - EDGE_MARGIN
  }
}

/** 松手后吸附到最近四角 */
const snapToCorner = () => {
  const b = getBounds()
  const cx = position.value.x + SIZE.width / 2
  const cy = position.value.y + SIZE.height / 2
  const centers = [
    // 左上
    { x: EDGE_MARGIN, y: EDGE_MARGIN },
    // 右上
    { x: b.right - SIZE.width - EDGE_MARGIN, y: EDGE_MARGIN },
    // 左下
    { x: EDGE_MARGIN, y: b.bottom - SIZE.height - EDGE_MARGIN },
    // 右下
    { x: b.right - SIZE.width - EDGE_MARGIN, y: b.bottom - SIZE.height - EDGE_MARGIN }
  ]
  let best = centers[3]
  let min = Infinity
  for (const c of centers) {
    const ccx = c.x + SIZE.width / 2
    const ccy = c.y + SIZE.height / 2
    const d = (cx - ccx) ** 2 + (cy - ccy) ** 2
    if (d < min) {
      min = d
      best = c
    }
  }
  position.value = best
}

// ============ 拖拽交互 ============

const getPoint = (e) => {
  // #ifdef H5
  if (e.touches && e.touches.length > 0) {
    return { x: e.touches[0].clientX, y: e.touches[0].clientY }
  }
  return { x: e.clientX, y: e.clientY }
  // #endif
  // #ifndef H5
  return { x: 0, y: 0 }
  // #endif
}

const onDragStart = (e) => {
  // #ifdef H5
  if (e.target && e.target.closest && e.target.closest('.pin-btn')) return
  const pt = getPoint(e)
  dragStart.value = {
    pointerX: pt.x,
    pointerY: pt.y,
    startX: position.value.x,
    startY: position.value.y
  }
  isDragging.value = true
  window.addEventListener('mousemove', onDragMove, { passive: false })
  window.addEventListener('touchmove', onDragMove, { passive: false })
  window.addEventListener('mouseup', onDragEnd)
  window.addEventListener('touchend', onDragEnd)
  window.addEventListener('touchcancel', onDragEnd)
  // 禁用文字选中
  document.body.style.userSelect = 'none'
  e.preventDefault()
  // #endif
}

const onDragMove = (e) => {
  // #ifdef H5
  if (!isDragging.value) return
  const pt = getPoint(e)
  const dx = pt.x - dragStart.value.pointerX
  const dy = pt.y - dragStart.value.pointerY
  const b = getBounds()
  // Clamp 在 bounds 内，防止浮窗被拖出视频区域
  const nx = Math.max(0, Math.min(dragStart.value.startX + dx, b.right - SIZE.width))
  const ny = Math.max(0, Math.min(dragStart.value.startY + dy, b.bottom - SIZE.height))
  position.value = { x: nx, y: ny }
  if (e.cancelable) e.preventDefault()
  // #endif
}

const onDragEnd = () => {
  // #ifdef H5
  if (!isDragging.value) return
  isDragging.value = false
  window.removeEventListener('mousemove', onDragMove)
  window.removeEventListener('touchmove', onDragMove)
  window.removeEventListener('mouseup', onDragEnd)
  window.removeEventListener('touchend', onDragEnd)
  window.removeEventListener('touchcancel', onDragEnd)
  document.body.style.userSelect = ''
  snapToCorner()
  // #endif
}

// ============ 容器尺寸变化 ============

let _resizeObserver = null

const onContainerResize = () => {
  // 重新 clamp + 吸附到当前象限对应的四角
  const b = getBounds()
  if (position.value.x > b.right - SIZE.width - EDGE_MARGIN ||
      position.value.y > b.bottom - SIZE.height - EDGE_MARGIN) {
    snapToCorner()
  }
}

watch(() => props.container, () => {
  // container 引用变化时重新定位
  nextTick(placeDefault)
})

const onPinClick = () => {
  emit('pin-click')
}

onMounted(() => {
  nextTick(placeDefault)
  // #ifdef H5
  window.addEventListener('resize', onContainerResize)
  if (typeof ResizeObserver !== 'undefined' && props.container) {
    const el = resolveDom(props.container)
    if (el) {
      _resizeObserver = new ResizeObserver(onContainerResize)
      _resizeObserver.observe(el)
    }
  }
  // #endif
})

onBeforeUnmount(() => {
  // #ifdef H5
  window.removeEventListener('resize', onContainerResize)
  if (_resizeObserver) {
    _resizeObserver.disconnect()
    _resizeObserver = null
  }
  // 兜底：组件卸载时仍在拖拽中则强制清理 window 监听
  if (isDragging.value) {
    onDragEnd()
  }
  // #endif
})
</script>

<style scoped>
.float-shell {
  position: absolute;
  top: 0;
  left: 0;
  z-index: 120;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 12rpx 32rpx rgba(0, 0, 0, 0.45);
  border: 1rpx solid rgba(255, 255, 255, 0.1);
  cursor: grab;
  touch-action: none;
  user-select: none;
  /* transform 直接由 :style 驱动，不再设置 transition（由 isDragging 控制） */
}
.float-shell.dragging {
  cursor: grabbing;
  box-shadow: 0 16rpx 40rpx rgba(0, 0, 0, 0.55);
}

.pin-btn {
  position: absolute;
  top: 8rpx;
  right: 8rpx;
  width: 36rpx;
  height: 36rpx;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.45);
  color: rgba(255, 255, 255, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background-color 0.15s ease;
  z-index: 2;
}
.pin-btn:hover {
  background: rgba(0, 0, 0, 0.65);
  color: #FFFFFF;
}

@media (prefers-reduced-motion: reduce) {
  .float-shell { transition: none; }
}
</style>
