<!--
  通知卡片组件
  
  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B / Muted #94A3B8
  ui-ux-pro-max 规范：触摸目标 ≥ 88rpx / 可点击 cursor-pointer / 新旧状态差异化
  
  功能：
  - 按 type 渲染图标和文案
  - 未读加红点标识
  - group_invite / group_join_request 支持内联“接受/拒绝”按钮
  - 点击卡片按 target_type + target_id 进行 deep-link 跳转
-->
<template>
  <view
    class="notify-item"
    :class="{ 'notify-item--unread': !notify.is_read }"
    @tap="handleTap"
  >
    <!-- 图标 -->
    <view
      class="notify-icon"
      :style="{ backgroundColor: iconBg }"
    >
      <text class="notify-icon-text">{{ icon }}</text>
      <view v-if="!notify.is_read" class="notify-dot"></view>
    </view>

    <!-- 内容 -->
    <view class="notify-body">
      <view class="notify-header">
        <text class="notify-title">{{ displayTitle }}</text>
        <text class="notify-time">{{ formatTime(notify.created_at) }}</text>
      </view>
      <text class="notify-content" v-if="notify.content">{{ notify.content }}</text>

      <!-- 内联操作按钮：仅限群聊邀请 / 入群申请 -->
      <view
        v-if="showActions"
        class="notify-actions"
      >
        <button
          class="notify-btn notify-btn--primary"
          :disabled="processing"
          @tap.stop="handleAccept"
        >
          接受
        </button>
        <button
          class="notify-btn notify-btn--ghost"
          :disabled="processing"
          @tap.stop="handleReject"
        >
          拒绝
        </button>
      </view>
    </view>
  </view>
</template>

<script setup>
/**
 * 通知卡片
 * 仅负责渲染和派发事件；具体业务动作由父页面实现
 */
import { computed, ref } from 'vue'
import {
  NOTIFY_TYPE_ICON,
  NOTIFY_DEFAULT_ICON,
  NOTIFY_TYPE_LABEL,
  NOTIFY_CATEGORY_COLOR,
  supportsInlineAction
} from '@/constants/notify'

const props = defineProps({
  /** 通知对象，包含 id / type / category / title / content / is_read / created_at / extra 等 */
  notify: {
    type: Object,
    required: true
  }
})

// 注意：uni-app 的 `tap` 是原生 DOM 事件名，若将自定义 emit 事件名也取作 `tap`，
// 则原生 tap 冒泡会与自定义 emit 在父组件处形成双重触发，且原生事件会把 Event 对象
// 作为参数，导致父组件收到的 notify 变成 Event 对象，随后读取 notify.id 为 undefined。
// 为彻底规避冲突，这里统一使用非 DOM 原生事件名。
const emit = defineEmits(['item-tap', 'item-accept', 'item-reject'])

/** 按钮处理中状态（防止重复点击） */
const processing = ref(false)

/** 显示标题：优先 notify.title，其次按 type 取默认文案 */
const displayTitle = computed(() => {
  return props.notify.title || NOTIFY_TYPE_LABEL[props.notify.type] || '通知'
})

/** 图标 */
const icon = computed(() => {
  return NOTIFY_TYPE_ICON[props.notify.type] || NOTIFY_DEFAULT_ICON
})

/** 图标背景色（按分类） */
const iconBg = computed(() => {
  return NOTIFY_CATEGORY_COLOR[props.notify.category] || '#F1F5F9'
})

/** 是否展示内联操作按钮（仅未读的 invite/join_request 通知） */
const showActions = computed(() => {
  if (props.notify.is_read) return false
  return supportsInlineAction(props.notify.type)
})

const handleTap = () => {
  emit('item-tap', props.notify)
}

const handleAccept = async () => {
  if (processing.value) return
  processing.value = true
  try {
    await emit('item-accept', props.notify)
  } finally {
    processing.value = false
  }
}

const handleReject = async () => {
  if (processing.value) return
  processing.value = true
  try {
    await emit('item-reject', props.notify)
  } finally {
    processing.value = false
  }
}

/**
 * 将 ISO 时间字符串格式化为相对时间
 * - 1 分钟内：刚刚
 * - 1 小时内：X 分钟前
 * - 24 小时内：X 小时前
 * - 否则：MM-DD HH:mm
 */
const formatTime = (str) => {
  if (!str) return ''
  const t = new Date(str.replace(' ', 'T'))
  if (isNaN(t.getTime())) return str
  const now = Date.now()
  const diff = Math.max(0, now - t.getTime())
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min} 分钟前`
  const hour = Math.floor(min / 60)
  if (hour < 24) return `${hour} 小时前`
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(t.getMonth() + 1)}-${pad(t.getDate())} ${pad(t.getHours())}:${pad(t.getMinutes())}`
}
</script>

<style scoped>
.notify-item {
  position: relative;
  display: flex;
  gap: 20rpx;
  padding: 28rpx 32rpx;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #F1F5F9;
  cursor: pointer;
  transition: background-color 200ms ease;
}

.notify-item:active {
  background-color: #F8FAFC;
}

.notify-item--unread {
  background-color: #F8FAFC;
}

.notify-icon {
  position: relative;
  width: 80rpx;
  height: 80rpx;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.notify-icon-text {
  font-size: 36rpx;
  line-height: 1;
}

.notify-dot {
  position: absolute;
  top: -2rpx;
  right: -2rpx;
  width: 16rpx;
  height: 16rpx;
  border-radius: 50%;
  background-color: #EF4444;
  border: 2rpx solid #FFFFFF;
}

.notify-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}

.notify-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 16rpx;
}

.notify-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #1E293B;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notify-time {
  font-size: 22rpx;
  color: #94A3B8;
  flex-shrink: 0;
}

.notify-content {
  font-size: 26rpx;
  color: #64748B;
  line-height: 1.5;
  /* 限制两行显示 */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 内联操作按钮 */
.notify-actions {
  display: flex;
  gap: 16rpx;
  margin-top: 12rpx;
}

.notify-btn {
  flex: 1;
  height: 64rpx;
  line-height: 64rpx;
  border-radius: 12rpx;
  font-size: 26rpx;
  font-weight: 500;
  padding: 0;
  margin: 0;
  border: none;
  cursor: pointer;
  transition: opacity 200ms ease;
}

.notify-btn::after {
  border: none;
}

.notify-btn[disabled] {
  opacity: 0.5;
}

.notify-btn--primary {
  background-color: #2563EB;
  color: #FFFFFF;
}

.notify-btn--ghost {
  background-color: #F1F5F9;
  color: #1E293B;
}
</style>
