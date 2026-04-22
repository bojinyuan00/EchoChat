<!--
  会议聊天面板（Task 12）

  职责：
  - 右侧抽屉式消息流 + 底部输入框，抽屉风格与 MemberPanel 对齐
  - 自己与他人的消息分别右 / 左对齐；同一人连续消息聚合头像
  - 顶部"加载更多历史"按钮；首次打开时派发 @request-load 由父层触发一次懒加载

  约束：
  - 纯受控组件，不直接依赖 Pinia；由父组件把 messages/hasMore 等数据传入
  - 发送通过 @send 事件回传，成功/失败由父层决定是否 toast
-->
<template>
  <view v-if="visible" class="panel-root" @click.self="close">
    <view class="panel" :class="{ show }">
      <view class="header">
        <view class="title-area">
          <text class="title">会议聊天</text>
          <text class="count">({{ messages.length }})</text>
        </view>
        <view class="btn-close" @click="close">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </view>
      </view>

      <scroll-view
        class="msg-list"
        :scroll-y="true"
        :scroll-top="scrollTop"
        :scroll-with-animation="scrollAnimate"
        @scroll="onScroll"
      >
        <view v-if="hasMore" class="load-more" @click="$emit('load-more')">
          <text v-if="!loadingMore">加载更早的消息</text>
          <text v-else>加载中…</text>
        </view>
        <view v-else-if="messages.length" class="load-end">
          <text>没有更早的消息了</text>
        </view>

        <view
          v-for="(m, idx) in messages"
          :key="m.id || `tmp_${idx}`"
          class="msg-row"
          :class="{ self: m.user_id === currentUserId, 'merge-top': isMergedWithPrev(idx) }"
        >
          <view v-if="!isMergedWithPrev(idx)" class="avatar" :style="avatarStyle(m.user_id)">
            {{ initialOf(m) }}
          </view>
          <view v-else class="avatar-placeholder"></view>

          <view class="bubble-area">
            <view v-if="!isMergedWithPrev(idx)" class="meta">
              <text class="meta-name">{{ m.user_id === currentUserId ? '我' : displayName(m) }}</text>
              <text class="meta-time">{{ formatTime(m.created_at) }}</text>
            </view>
            <view class="bubble" :class="{ self: m.user_id === currentUserId }">
              <text>{{ m.content }}</text>
            </view>
          </view>
        </view>

        <view v-if="!messages.length" class="empty">
          <text>发送第一条消息，开始会议内对话</text>
        </view>

        <view :style="{ height: '4rpx' }" ref="bottomAnchor"></view>
      </scroll-view>

      <view class="input-area">
        <textarea
          class="input"
          :value="draft"
          placeholder="说点什么…"
          :maxlength="500"
          :adjust-position="false"
          :cursor-spacing="10"
          @input="onDraftInput"
          @confirm="onSend"
        />
        <button class="btn-send" :disabled="!canSend || sending" @click="onSend">
          <text v-if="!sending">发送</text>
          <text v-else>发送中</text>
        </button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  messages: { type: Array, default: () => [] },
  hasMore: { type: Boolean, default: true },
  loadingMore: { type: Boolean, default: false },
  currentUserId: { type: [Number, String], default: 0 },
  displayNameMap: { type: Object, default: () => ({}) }
})

const emit = defineEmits(['close', 'send', 'load-more', 'request-initial-load'])

const show = ref(false)
const draft = ref('')
const sending = ref(false)
const scrollTop = ref(0)
const scrollAnimate = ref(true)
const userScrolledUp = ref(false)
const lastScrollTop = ref(0)

watch(() => props.visible, async (v) => {
  if (v) {
    setTimeout(() => { show.value = true }, 10)
    emit('request-initial-load')
    await nextTick()
    scrollToBottom(false)
  } else {
    show.value = false
  }
}, { immediate: true })

// 新消息到达时，若用户未向上翻阅，自动滚到底
watch(() => props.messages.length, async () => {
  if (!props.visible) return
  await nextTick()
  if (!userScrolledUp.value) scrollToBottom(true)
})

const canSend = computed(() => draft.value.trim().length > 0)

const onDraftInput = (e) => {
  draft.value = e?.detail?.value ?? e?.target?.value ?? ''
}

const onSend = async () => {
  const content = draft.value.trim()
  if (!content || sending.value) return
  sending.value = true
  try {
    await emit('send', content)
    draft.value = ''
    await nextTick()
    scrollToBottom(true)
  } finally {
    sending.value = false
  }
}

const scrollToBottom = (animated) => {
  scrollAnimate.value = animated
  // 不可随便等于 last，不触发；每次加一个微量触发重置 scrollTop
  const next = scrollTop.value + 1
  scrollTop.value = 999999
  setTimeout(() => { if (scrollTop.value === 999999) scrollTop.value = next }, 30)
  userScrolledUp.value = false
}

const onScroll = (e) => {
  const top = e?.detail?.scrollTop ?? 0
  // 若用户明显向上拉动（大于 60），则记为"手动上滚"，暂停自动滚动
  if (top < lastScrollTop.value - 40) userScrolledUp.value = true
  // 拉到底时恢复自动滚动
  const max = (e?.detail?.scrollHeight || 0) - (e?.detail?.viewportHeight || e?.detail?.wheelDeltaY || 0)
  if (max > 0 && top >= max - 20) userScrolledUp.value = false
  lastScrollTop.value = top
}

/** 相邻消息同一个发送者且间隔 < 5 分钟 → 合并头像 */
const isMergedWithPrev = (idx) => {
  if (idx === 0) return false
  const cur = props.messages[idx]
  const prev = props.messages[idx - 1]
  if (!cur || !prev) return false
  if (cur.user_id !== prev.user_id) return false
  const diff = timestampOf(cur) - timestampOf(prev)
  return diff >= 0 && diff < 5 * 60 * 1000
}

const timestampOf = (m) => {
  if (!m?.created_at) return 0
  const t = Date.parse(String(m.created_at).replace(' ', 'T'))
  return Number.isFinite(t) ? t : 0
}

const displayName = (m) => {
  if (m.user_name) return m.user_name
  if (props.displayNameMap[m.user_id]) return props.displayNameMap[m.user_id]
  return `用户 ${m.user_id}`
}

const initialOf = (m) => {
  const n = (m.user_id === props.currentUserId ? '我' : displayName(m)).trim()
  return n ? n.charAt(0).toUpperCase() : '?'
}

const avatarStyle = (uid) => {
  const seed = String(uid || '0')
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) % 360
  return { background: `hsl(${h}, 60%, 48%)` }
}

const formatTime = (iso) => {
  if (!iso) return ''
  const t = Date.parse(String(iso).replace(' ', 'T'))
  if (!Number.isFinite(t)) return ''
  const d = new Date(t)
  const pad = (n) => n < 10 ? `0${n}` : String(n)
  const now = new Date()
  const sameDay = d.getFullYear() === now.getFullYear()
    && d.getMonth() === now.getMonth()
    && d.getDate() === now.getDate()
  if (sameDay) return `${pad(d.getHours())}:${pad(d.getMinutes())}`
  return `${d.getMonth() + 1}/${d.getDate()} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const close = () => emit('close')
</script>

<style scoped>
.panel-root {
  position: fixed;
  inset: 0;
  z-index: 205;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  justify-content: flex-end;
}

.panel {
  width: 640rpx;
  max-width: 92vw;
  height: 100%;
  background: #1F2937;
  color: #F3F4F6;
  display: flex;
  flex-direction: column;
  transform: translateX(100%);
  transition: transform 0.24s ease;
  box-shadow: -4rpx 0 24rpx rgba(0, 0, 0, 0.4);
}
.panel.show { transform: translateX(0); }

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 28rpx 32rpx;
  border-bottom: 1rpx solid rgba(255, 255, 255, 0.08);
  flex-shrink: 0;
}
.title-area { display: flex; align-items: baseline; gap: 10rpx; }
.title { font-size: 32rpx; font-weight: 600; }
.count { color: rgba(255, 255, 255, 0.5); font-size: 24rpx; }
.btn-close {
  width: 56rpx;
  height: 56rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
}
.btn-close:hover { background: rgba(255, 255, 255, 0.08); }

.msg-list {
  flex: 1;
  padding: 16rpx 24rpx 24rpx;
  box-sizing: border-box;
}

.load-more, .load-end {
  text-align: center;
  font-size: 22rpx;
  color: rgba(255, 255, 255, 0.5);
  padding: 14rpx 0;
  cursor: pointer;
}
.load-more:hover { color: rgba(255, 255, 255, 0.8); }

.empty {
  padding: 100rpx 0 80rpx;
  text-align: center;
  color: rgba(255, 255, 255, 0.45);
  font-size: 24rpx;
}

.msg-row {
  display: flex;
  gap: 16rpx;
  padding: 8rpx 0;
}
.msg-row.merge-top { margin-top: -12rpx; padding-top: 0; }
.msg-row.self { flex-direction: row-reverse; }

.avatar {
  flex-shrink: 0;
  width: 64rpx;
  height: 64rpx;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 26rpx;
  font-weight: 600;
  color: #FFFFFF;
}
.avatar-placeholder {
  flex-shrink: 0;
  width: 64rpx;
}

.bubble-area {
  display: flex;
  flex-direction: column;
  gap: 8rpx;
  max-width: 70%;
  min-width: 0;
}
.msg-row.self .bubble-area { align-items: flex-end; }

.meta {
  display: flex;
  align-items: baseline;
  gap: 10rpx;
  color: rgba(255, 255, 255, 0.55);
  font-size: 20rpx;
}
.msg-row.self .meta { flex-direction: row-reverse; }
.meta-name { font-weight: 500; color: rgba(255, 255, 255, 0.85); font-size: 22rpx; }
.meta-time { font-family: 'SF Mono', 'Menlo', monospace; }

.bubble {
  padding: 14rpx 20rpx;
  border-radius: 16rpx;
  background: rgba(55, 65, 81, 0.9);
  color: #F3F4F6;
  font-size: 26rpx;
  line-height: 1.55;
  word-break: break-word;
  white-space: pre-wrap;
  max-width: 100%;
}
.bubble.self {
  background: #3B82F6;
  color: #FFFFFF;
  border-bottom-right-radius: 4rpx;
}
.bubble:not(.self) { border-bottom-left-radius: 4rpx; }

.input-area {
  display: flex;
  align-items: flex-end;
  gap: 12rpx;
  padding: 16rpx 24rpx calc(16rpx + env(safe-area-inset-bottom));
  border-top: 1rpx solid rgba(255, 255, 255, 0.08);
  flex-shrink: 0;
}
.input {
  flex: 1;
  background: rgba(55, 65, 81, 0.6);
  border: 1rpx solid rgba(255, 255, 255, 0.1);
  border-radius: 14rpx;
  padding: 14rpx 18rpx;
  color: #F3F4F6;
  font-size: 26rpx;
  line-height: 1.5;
  min-height: 64rpx;
  max-height: 220rpx;
  box-sizing: border-box;
}
.input:focus {
  border-color: rgba(59, 130, 246, 0.7);
  outline: none;
}

.btn-send {
  all: unset;
  cursor: pointer;
  padding: 14rpx 28rpx;
  background: #3B82F6;
  color: #FFFFFF;
  font-size: 26rpx;
  font-weight: 600;
  border-radius: 14rpx;
  transition: background-color 0.15s ease;
  flex-shrink: 0;
}
.btn-send:hover { background: #2563EB; }
.btn-send[disabled] {
  background: rgba(59, 130, 246, 0.35);
  cursor: not-allowed;
}
</style>
