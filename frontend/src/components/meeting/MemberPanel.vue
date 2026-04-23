<!--
  成员抽屉（Task 11）

  职责：
  - 展示当前会议参与者列表（排序：主持人置顶 → 自己 → 其他加入时间倒序）
  - 主持人可通过行尾菜单 转让主持 / 踢出成员；事件派发由父组件承接 API 调用
  - 受控组件：通过 visible + @close 控制显隐，遮罩点击关闭

  说明：参与者基础字段来自 meeting.member.joined 广播，name/nickname 字段
  由后端后续补全；当前 fallback 到 "用户 {user_id}"
-->
<template>
  <view v-if="visible" class="panel-root" @click.self="onMaskClick">
    <view class="panel" :class="{ show: show }">
      <view class="header">
        <view class="title-area">
          <text class="title">会议成员</text>
          <text class="count">({{ participants.length }})</text>
        </view>
        <view class="btn-close" @click="close">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </view>
      </view>

      <scroll-view scroll-y class="list">
        <view
          v-for="p in sortedList"
          :key="p.user_id"
          class="row"
        >
          <view class="avatar" :style="avatarStyle(p.user_id)">
            {{ nameInitial(p) }}
          </view>
          <view class="info">
            <view class="info-name">
              <text class="name-text">{{ displayName(p) }}</text>
              <view v-if="p.user_id === currentUserId" class="tag tag-self">你</view>
              <view v-if="p.user_id === hostId" class="tag tag-host">主持人</view>
            </view>
            <view class="info-state">
              <view class="state-icon" :class="{ off: !p.audio_enabled }">
                <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"></path>
                  <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
                  <line x1="12" y1="19" x2="12" y2="23"></line>
                </svg>
              </view>
              <view class="state-icon" :class="{ off: !p.video_enabled }">
                <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polygon points="23 7 16 12 23 17 23 7"></polygon>
                  <rect x="1" y="5" width="15" height="14" rx="2" ry="2"></rect>
                </svg>
              </view>
            </view>
          </view>

          <view
            v-if="canOperate(p)"
            class="actions"
            @click.stop="toggleMenu(p.user_id)"
          >
            <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="1"></circle>
              <circle cx="12" cy="5" r="1"></circle>
              <circle cx="12" cy="19" r="1"></circle>
            </svg>
            <view v-if="openMenuUid === p.user_id" class="menu" @click.stop>
              <view
                v-if="p.audio_enabled !== false"
                class="menu-item"
                @click="onMute(p, true)"
              >
                <text>请他静音</text>
              </view>
              <view
                v-else
                class="menu-item"
                @click="onMute(p, false)"
              >
                <text>请他开麦</text>
              </view>
              <view class="menu-item" @click="onTransfer(p)">
                <text>转让主持人</text>
              </view>
              <view class="menu-item menu-danger" @click="onKick(p)">
                <text>踢出会议</text>
              </view>
            </view>
          </view>
        </view>

        <view v-if="!sortedList.length" class="empty-state">
          <text>当前会议室还没有其他成员</text>
        </view>
      </scroll-view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  participants: { type: Array, default: () => [] },
  currentUserId: { type: [Number, String], default: 0 },
  hostId: { type: [Number, String], default: 0 },
  isHost: { type: Boolean, default: false },
  displayNameMap: { type: Object, default: () => ({}) }
})

const emit = defineEmits(['close', 'kick', 'transfer-host', 'mute-member'])

const openMenuUid = ref(0)
const show = ref(false)

/** 动画：先挂载再 nextTick 触发 transform */
watch(() => props.visible, (v) => {
  if (v) {
    setTimeout(() => { show.value = true }, 10)
  } else {
    show.value = false
    openMenuUid.value = 0
  }
}, { immediate: true })

const sortedList = computed(() => {
  const list = props.participants.filter(p => p.is_active !== false)
  return [...list].sort((a, b) => {
    if (a.user_id === props.hostId) return -1
    if (b.user_id === props.hostId) return 1
    if (a.user_id === props.currentUserId) return -1
    if (b.user_id === props.currentUserId) return 1
    const ta = a.joined_at ? Date.parse(a.joined_at) : 0
    const tb = b.joined_at ? Date.parse(b.joined_at) : 0
    return ta - tb
  })
})

const displayName = (p) => {
  if (p.nickname) return p.nickname
  if (p.username) return p.username
  if (props.displayNameMap[p.user_id]) return props.displayNameMap[p.user_id]
  return `用户 ${p.user_id}`
}

const nameInitial = (p) => {
  const n = displayName(p).trim()
  return n ? n.charAt(0).toUpperCase() : '?'
}

const avatarStyle = (uid) => {
  const seed = String(uid || '0')
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) % 360
  return { background: `hsl(${h}, 60%, 48%)` }
}

const canOperate = (p) => {
  return props.isHost && p.user_id !== props.currentUserId
}

const toggleMenu = (uid) => {
  openMenuUid.value = openMenuUid.value === uid ? 0 : uid
}

const onTransfer = (p) => {
  openMenuUid.value = 0
  emit('transfer-host', p.user_id)
}

const onKick = (p) => {
  openMenuUid.value = 0
  emit('kick', p.user_id)
}

/** Task 15：主持人权限四件套第 4 件 —— 请他静音 / 请他开麦 */
const onMute = (p, mute) => {
  openMenuUid.value = 0
  emit('mute-member', { userId: p.user_id, mute })
}

const close = () => emit('close')
const onMaskClick = () => close()
</script>

<style scoped>
.panel-root {
  position: fixed;
  inset: 0;
  z-index: 200;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  justify-content: flex-end;
}

.panel {
  width: 620rpx;
  max-width: 88vw;
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

.list {
  flex: 1;
  padding: 12rpx 0;
}

.row {
  display: flex;
  align-items: center;
  gap: 20rpx;
  padding: 16rpx 32rpx;
  transition: background-color 0.12s ease;
}
.row:hover { background: rgba(255, 255, 255, 0.04); }

.avatar {
  flex-shrink: 0;
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32rpx;
  font-weight: 600;
  color: #FFFFFF;
}

.info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}
.info-name {
  display: flex;
  align-items: center;
  gap: 12rpx;
  min-width: 0;
}
.name-text {
  font-size: 28rpx;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tag {
  padding: 2rpx 10rpx;
  border-radius: 999rpx;
  font-size: 18rpx;
  line-height: 1.3;
  font-weight: 600;
  color: #FFFFFF;
  flex-shrink: 0;
}
.tag-self { background: rgba(59, 130, 246, 0.85); }
.tag-host { background: rgba(245, 158, 11, 0.9); }

.info-state {
  display: flex;
  align-items: center;
  gap: 14rpx;
  color: rgba(255, 255, 255, 0.6);
  font-size: 22rpx;
}
.state-icon {
  width: 28rpx;
  height: 28rpx;
  border-radius: 50%;
  background: rgba(59, 130, 246, 0.8);
  color: #FFFFFF;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.state-icon.off {
  background: rgba(107, 114, 128, 0.55);
  color: rgba(255, 255, 255, 0.65);
}

.actions {
  position: relative;
  width: 56rpx;
  height: 56rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12rpx;
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
}
.actions:hover { background: rgba(255, 255, 255, 0.08); }

.menu {
  position: absolute;
  top: 64rpx;
  right: 0;
  min-width: 220rpx;
  background: #374151;
  border-radius: 12rpx;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.45);
  overflow: hidden;
  z-index: 10;
}
.menu-item {
  padding: 20rpx 24rpx;
  font-size: 26rpx;
  color: #F3F4F6;
  transition: background-color 0.12s ease;
}
.menu-item:hover { background: rgba(255, 255, 255, 0.08); }
.menu-danger { color: #F87171; }

.empty-state {
  padding: 80rpx 32rpx;
  text-align: center;
  color: rgba(255, 255, 255, 0.45);
  font-size: 26rpx;
}
</style>
