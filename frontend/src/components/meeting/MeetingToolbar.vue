<!--
  会议工具条（Task 11）

  职责：
  - 渲染麦/摄/邀请/成员/聊天/挂断 6 个主要操作按钮
  - 通过事件派发意图，不持有业务状态（状态由父组件 / Pinia 管理）

  交互：
  - 麦克风 / 摄像头按钮显示当前状态（开 = 蓝色，关 = 深灰）
  - 挂断按钮为红色，点击仅派发事件，确认弹窗由父页面承接（降低本组件耦合）
  - 按钮带 loading 态，禁用期间点击不派发事件
-->
<template>
  <view class="toolbar">
    <button class="btn" :class="{ active: audioEnabled, loading: audioLoading }" :disabled="audioLoading" @click="emitIf('mic-toggle')">
      <view class="icon">
        <svg v-if="audioEnabled" viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"></path>
          <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
          <line x1="12" y1="19" x2="12" y2="23"></line>
          <line x1="8" y1="23" x2="16" y2="23"></line>
        </svg>
        <svg v-else viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="1" y1="1" x2="23" y2="23"></line>
          <path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"></path>
          <path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2a7 7 0 0 1-.11 1.23"></path>
          <line x1="12" y1="19" x2="12" y2="23"></line>
          <line x1="8" y1="23" x2="16" y2="23"></line>
        </svg>
      </view>
      <text class="label">{{ audioEnabled ? '静音' : '解除' }}</text>
    </button>

    <button class="btn" :class="{ active: videoEnabled, loading: videoLoading }" :disabled="videoLoading" @click="emitIf('cam-toggle')">
      <view class="icon">
        <svg v-if="videoEnabled" viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="23 7 16 12 23 17 23 7"></polygon>
          <rect x="1" y="5" width="15" height="14" rx="2" ry="2"></rect>
        </svg>
        <svg v-else viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="1" y1="1" x2="23" y2="23"></line>
          <path d="M16 16v2a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h2m5.66 0H14a2 2 0 0 1 2 2v4m0 0l6-4v10"></path>
        </svg>
      </view>
      <text class="label">{{ videoEnabled ? '停止视频' : '开启视频' }}</text>
    </button>

    <button class="btn" @click="emitIf('invite')">
      <view class="icon">
        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
          <circle cx="8.5" cy="7" r="4"></circle>
          <line x1="20" y1="8" x2="20" y2="14"></line>
          <line x1="23" y1="11" x2="17" y2="11"></line>
        </svg>
      </view>
      <text class="label">邀请</text>
    </button>

    <button class="btn" @click="emitIf('members')">
      <view class="icon">
        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
          <circle cx="9" cy="7" r="4"></circle>
          <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
        </svg>
        <view v-if="memberCount > 0" class="badge">{{ memberCount > 99 ? '99+' : memberCount }}</view>
      </view>
      <text class="label">成员</text>
    </button>

    <button class="btn" :disabled="!allowChat" @click="emitIf('chat')">
      <view class="icon">
        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
        </svg>
        <view v-if="unreadChatCount > 0" class="badge badge-red">{{ unreadChatCount > 99 ? '99+' : unreadChatCount }}</view>
      </view>
      <text class="label">聊天</text>
    </button>

    <button class="btn btn-leave" @click="emitIf('leave')">
      <view class="icon">
        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91"></path>
          <line x1="23" y1="1" x2="1" y2="23"></line>
        </svg>
      </view>
      <text class="label">离开</text>
    </button>
  </view>
</template>

<script setup>
const props = defineProps({
  audioEnabled: { type: Boolean, default: false },
  videoEnabled: { type: Boolean, default: false },
  audioLoading: { type: Boolean, default: false },
  videoLoading: { type: Boolean, default: false },
  memberCount: { type: Number, default: 0 },
  unreadChatCount: { type: Number, default: 0 },
  allowChat: { type: Boolean, default: true }
})

const emit = defineEmits(['mic-toggle', 'cam-toggle', 'invite', 'members', 'chat', 'leave'])

/** 统一过滤 loading 态，避免重复点击 */
const emitIf = (name) => {
  if (name === 'mic-toggle' && props.audioLoading) return
  if (name === 'cam-toggle' && props.videoLoading) return
  emit(name)
}
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16rpx;
  padding: 18rpx 24rpx calc(18rpx + env(safe-area-inset-bottom)) 24rpx;
  background: rgba(17, 24, 39, 0.88);
  backdrop-filter: blur(16px);
  border-top: 1rpx solid rgba(255, 255, 255, 0.08);
  /* flex-shrink:0 保证 body 内容高度过大时不会把 toolbar 挤出视口；
     position:relative + z-index 提升按钮点击层级，避免 ChatPanel 遮挡 */
  flex-shrink: 0;
  position: relative;
  z-index: 15;
}

.btn {
  all: unset;
  cursor: pointer;
  min-width: 104rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6rpx;
  padding: 8rpx 8rpx;
  border-radius: 16rpx;
  transition: background-color 0.15s ease, transform 0.1s ease;
  color: rgba(255, 255, 255, 0.85);
  position: relative;
}
/* 关键修复：uni-app H5 会为 <button> 编译后的 <uni-button> 自动注入一个
   伪元素 ::after（尺寸约 2400rpx × 160rpx，向右下延伸），
   作为点击态反馈遮罩；若不抹除，会覆盖右侧所有兄弟按钮，
   导致除最后一个按钮外其他按钮都"点不到"。 */
.btn::after {
  content: none !important;
  display: none !important;
}
.btn:hover { background: rgba(255, 255, 255, 0.06); }
.btn:active { transform: translateY(1rpx); }
.btn[disabled] { opacity: 0.45; cursor: not-allowed; }

.icon {
  position: relative;
  width: 72rpx;
  height: 72rpx;
  border-radius: 50%;
  background: rgba(55, 65, 81, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #FFFFFF;
  transition: background-color 0.15s ease;
}
.btn.active .icon { background: #3B82F6; }
.btn.loading .icon { opacity: 0.6; }

.label {
  font-size: 22rpx;
  letter-spacing: 0.5rpx;
  color: inherit;
}

.btn-leave .icon { background: #EF4444; }
.btn-leave:hover .icon { background: #DC2626; }

.badge {
  position: absolute;
  top: -6rpx;
  right: -6rpx;
  min-width: 32rpx;
  height: 32rpx;
  padding: 0 6rpx;
  border-radius: 999rpx;
  background: #10B981;
  color: #FFFFFF;
  font-size: 18rpx;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2rpx 6rpx rgba(0, 0, 0, 0.4);
}
.badge.badge-red { background: #EF4444; }

@media (max-width: 420px) {
  .btn { min-width: 88rpx; }
  .icon { width: 64rpx; height: 64rpx; }
  .label { font-size: 20rpx; }
}
</style>
