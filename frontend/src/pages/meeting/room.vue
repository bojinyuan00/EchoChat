<!--
  会议主页面（Task 11）

  职责：
  - 组装 VideoGrid / VideoTile / MeetingToolbar / MemberPanel / InviteDialog / NetworkBadge
  - 把 Pinia 的 participants + remoteConsumers + 本地 track 投影成 Grid 所需的 tile 列表
  - 处理麦/摄/邀请/成员/挂断交互，挂断时提供主持人"结束会议 / 仅自己离开"双选

  数据流：
  - 进入本页前必须已通过 preview.vue 完成 createAndEnter / joinAndEnter
  - onLoad 读取 query.code，若 store 未在会或 code 不匹配则回退到 /pages/meeting/join
-->
<template>
  <view class="room">
    <!-- 顶部信息栏 -->
    <view class="top-bar">
      <view class="top-left">
        <text class="meeting-title">{{ meetingTitle }}</text>
        <view class="code-pill" @click="openInvite">
          <text class="code-text">{{ formattedCode }}</text>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
          </svg>
        </view>
      </view>
      <view class="top-right">
        <view class="state-pill" :class="stateClass">
          <view class="state-dot"></view>
          <text>{{ stateText }}</text>
        </view>
        <NetworkBadge :level="networkLevel" />
        <text class="timer">{{ durationText }}</text>
      </view>
    </view>

    <!-- 视频网格 -->
    <view class="body">
      <VideoGrid :tiles="tiles">
        <template #tile="{ tile }">
          <VideoTile
            :user-id="tile.userId"
            :name="tile.name"
            :is-local="tile.isLocal"
            :is-host="tile.isHost"
            :audio-enabled="tile.audioEnabled"
            :video-enabled="tile.videoEnabled"
            :audio-track="tile.audioTrack"
            :video-track="tile.videoTrack"
          />
        </template>
      </VideoGrid>
    </view>

    <!-- 底部工具条 -->
    <MeetingToolbar
      :audio-enabled="audioEnabled"
      :video-enabled="videoEnabled"
      :audio-loading="audioLoading"
      :video-loading="videoLoading"
      :member-count="tiles.length"
      :unread-chat-count="unreadChatCount"
      :allow-chat="allowChat"
      @mic-toggle="onMicToggle"
      @cam-toggle="onCamToggle"
      @invite="openInvite"
      @members="openMembers"
      @chat="openChat"
      @leave="onLeaveClick"
    />

    <!-- 成员抽屉 -->
    <MemberPanel
      :visible="memberVisible"
      :participants="participants"
      :current-user-id="myUserId"
      :host-id="hostId"
      :is-host="isHost"
      :display-name-map="displayNameMap"
      @close="memberVisible = false"
      @kick="onKickMember"
      @transfer-host="onTransferHost"
    />

    <!-- 邀请弹窗 -->
    <InviteDialog
      :visible="inviteVisible"
      :room-code="roomCode"
      :password="invitePassword"
      :meeting-title="meetingTitle"
      :join-link="joinLink"
      @close="inviteVisible = false"
    />

    <!-- 离会确认弹窗 -->
    <view v-if="leaveDialogVisible" class="leave-mask" @click.self="leaveDialogVisible = false">
      <view class="leave-modal">
        <view class="leave-title">确认{{ isHost ? '结束或离开' : '离开' }}会议？</view>
        <view class="leave-desc">
          {{ isHost ? '作为主持人，你可以选择结束会议（所有成员都会离开），或仅自己离开（自动转让给其他成员）。' : '离开后可通过会议号重新加入。' }}
        </view>
        <view class="leave-actions">
          <button class="leave-btn leave-cancel" @click="leaveDialogVisible = false">取消</button>
          <button v-if="isHost" class="leave-btn leave-leave" @click="confirmLeaveSelf">仅自己离开</button>
          <button class="leave-btn leave-danger" @click="confirmEndOrLeave">
            {{ isHost ? '结束会议' : '离开会议' }}
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { useMeetingStore } from '@/store/meeting'
import { useUserStore } from '@/store/user'
import {
  MEETING_LOCAL_STATE_CONNECTED,
  MEETING_LOCAL_STATE_CONNECTING,
  MEETING_LOCAL_STATE_JOINING,
  MEETING_LOCAL_STATE_RECONNECTING,
  MEETING_LOCAL_STATE_LEAVING,
  MEETING_LOCAL_STATE_ENDED,
  MEETING_ENDED_REASON_LABEL
} from '@/constants/meeting'

import VideoGrid from '@/components/meeting/VideoGrid.vue'
import VideoTile from '@/components/meeting/VideoTile.vue'
import MeetingToolbar from '@/components/meeting/MeetingToolbar.vue'
import MemberPanel from '@/components/meeting/MemberPanel.vue'
import InviteDialog from '@/components/meeting/InviteDialog.vue'
import NetworkBadge from '@/components/meeting/NetworkBadge.vue'

const meetingStore = useMeetingStore()
const userStore = useUserStore()

const memberVisible = ref(false)
const inviteVisible = ref(false)
const leaveDialogVisible = ref(false)
const audioLoading = ref(false)
const videoLoading = ref(false)
const networkLevel = ref(4) // Task 15 接入真实 getStats 后动态更新
const unreadChatCount = ref(0) // Task 14 接入 chat 后动态更新
const startedAt = ref(0)
const nowTs = ref(Date.now())
let timerHandle = null

// ============ 基础映射 ============

const myUserId = computed(() => userStore.userInfo?.id || 0)
const isHost = computed(() => meetingStore.isHost)
/**
 * 修正 participants：本地用户的 audio_enabled/video_enabled 优先使用 store
 * 本地状态（localAudioEnabled/localVideoEnabled），因为后端不会给自己广播
 * member.state.changed，若只依赖 REST 详情易和实际操作脱节
 */
const participants = computed(() => {
  const uid = myUserId.value
  return meetingStore.activeParticipants.map(p => {
    if (p.user_id !== uid) return p
    return {
      ...p,
      audio_enabled: meetingStore.localAudioEnabled,
      video_enabled: meetingStore.localVideoEnabled
    }
  })
})
const hostId = computed(() => meetingStore.currentRoom?.host_id || 0)
const roomCode = computed(() => meetingStore.currentRoom?.room_code || '')
const meetingTitle = computed(() => meetingStore.currentRoom?.title || '会议')
const allowChat = computed(() => {
  const settings = meetingStore.currentRoom?.settings
  if (!settings) return true
  try {
    const parsed = typeof settings === 'string' ? JSON.parse(settings) : settings
    return parsed?.allow_chat !== false
  } catch {
    return true
  }
})
const invitePassword = computed(() => meetingStore.draftCreatePayload?.password || '')

const formattedCode = computed(() => {
  const d = String(roomCode.value || '').replace(/\D/g, '')
  if (d.length !== 9) return d
  return `${d.slice(0, 3)}-${d.slice(3, 6)}-${d.slice(6)}`
})

const joinLink = computed(() => {
  // #ifdef H5
  const base = `${window.location.origin}${window.location.pathname}`
  return `${base}#/pages/meeting/join?code=${roomCode.value}`
  // #endif
  // #ifndef H5
  return `echochat://meeting/${roomCode.value}`
  // #endif
})

// ============ 状态 pill ============

const stateClass = computed(() => {
  switch (meetingStore.localState) {
    case MEETING_LOCAL_STATE_CONNECTED: return 'state-connected'
    case MEETING_LOCAL_STATE_RECONNECTING: return 'state-reconnecting'
    case MEETING_LOCAL_STATE_CONNECTING:
    case MEETING_LOCAL_STATE_JOINING: return 'state-connecting'
    case MEETING_LOCAL_STATE_LEAVING: return 'state-leaving'
    case MEETING_LOCAL_STATE_ENDED: return 'state-ended'
    default: return 'state-idle'
  }
})
const stateText = computed(() => {
  switch (meetingStore.localState) {
    case MEETING_LOCAL_STATE_CONNECTED: return '已连接'
    case MEETING_LOCAL_STATE_RECONNECTING: return '重连中'
    case MEETING_LOCAL_STATE_CONNECTING:
    case MEETING_LOCAL_STATE_JOINING: return '连接中'
    case MEETING_LOCAL_STATE_LEAVING: return '正在离开'
    case MEETING_LOCAL_STATE_ENDED: return '已结束'
    default: return '等待中'
  }
})

// ============ 计时 ============

const durationText = computed(() => {
  if (!startedAt.value) return '00:00'
  const diff = Math.max(0, Math.floor((nowTs.value - startedAt.value) / 1000))
  const h = Math.floor(diff / 3600)
  const m = Math.floor((diff % 3600) / 60)
  const s = diff % 60
  const pad = (n) => (n < 10 ? `0${n}` : String(n))
  return h > 0 ? `${pad(h)}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`
})

// ============ 媒体状态 ============

const audioEnabled = computed(() => meetingStore.localAudioEnabled)
const videoEnabled = computed(() => meetingStore.localVideoEnabled)

/** 监听 store.getLocalTrack 的变化：localProducers 变化即 bump 版本 */
const localTrackVersion = computed(() => {
  return `${meetingStore.localProducers.audio || ''}-${meetingStore.localProducers.video || ''}`
})

/** 远端 track 版本：让 tiles 在 slot.audio/slot.video 变化时重新计算 */
const remoteTrackVersion = computed(() => {
  const parts = []
  const map = meetingStore.remoteConsumers
  Object.keys(map).forEach((uid) => {
    const slot = map[uid]
    parts.push(`${uid}:${slot?.audio?.id || ''}:${slot?.video?.id || ''}`)
  })
  return parts.join('|')
})

/** 参与者姓名缓存：user_name 来自后端 DTO */
const displayNameMap = computed(() => {
  const m = {}
  participants.value.forEach(p => {
    if (p.user_name) m[p.user_id] = p.user_name
  })
  const me = userStore.userInfo
  if (me?.id) m[me.id] = me.nickname || me.username || m[me.id]
  return m
})

const nameOf = (uid) => {
  const me = userStore.userInfo
  if (me?.id === uid) return me.nickname || me.username || `你`
  return displayNameMap.value[uid] || `用户 ${uid}`
}

/** 把 store 状态投影为 VideoGrid 需要的 tile 列表 */
const tiles = computed(() => {
  // 监听 track 版本让 computed 跟随远端/本地 track 变化重算
  // eslint-disable-next-line @typescript-eslint/no-unused-expressions
  localTrackVersion.value
  // eslint-disable-next-line @typescript-eslint/no-unused-expressions
  remoteTrackVersion.value

  const list = []
  const myUid = myUserId.value
  participants.value.forEach(p => {
    if (!p.is_active && p.is_active !== undefined) return
    const isLocal = p.user_id === myUid
    const tile = {
      key: `u_${p.user_id}`,
      userId: p.user_id,
      name: isLocal ? '你' : nameOf(p.user_id),
      isLocal,
      isHost: p.user_id === hostId.value,
      audioEnabled: isLocal ? audioEnabled.value : (p.audio_enabled !== false),
      videoEnabled: isLocal ? videoEnabled.value : (p.video_enabled !== false),
      audioTrack: null,
      videoTrack: null
    }
    if (isLocal) {
      tile.audioTrack = meetingStore.getLocalTrack('audio')
      tile.videoTrack = meetingStore.getLocalTrack('video')
    } else {
      tile.audioTrack = meetingStore.getRemoteTrack(p.user_id, 'audio')
      tile.videoTrack = meetingStore.getRemoteTrack(p.user_id, 'video')
    }
    list.push(tile)
  })
  // 若本地用户尚未出现在 participants 里（极端情况：REST 延迟），兜底追加一个本地 tile
  if (myUid && !list.some(t => t.userId === myUid)) {
    list.unshift({
      key: `u_${myUid}`,
      userId: myUid,
      name: '你',
      isLocal: true,
      isHost: hostId.value === myUid,
      audioEnabled: audioEnabled.value,
      videoEnabled: videoEnabled.value,
      audioTrack: meetingStore.getLocalTrack('audio'),
      videoTrack: meetingStore.getLocalTrack('video')
    })
  }
  return list
})

// ============ 事件处理 ============

const onMicToggle = async () => {
  audioLoading.value = true
  try {
    if (audioEnabled.value) {
      await meetingStore.stopLocalAudio()
    } else {
      await meetingStore.startLocalAudio(meetingStore.devicePreview?.audioDeviceId)
    }
  } catch (err) {
    uni.showToast({ title: err?.message || '操作失败', icon: 'none' })
  } finally {
    audioLoading.value = false
  }
}

const onCamToggle = async () => {
  videoLoading.value = true
  try {
    if (videoEnabled.value) {
      await meetingStore.stopLocalVideo()
    } else {
      await meetingStore.startLocalVideo(meetingStore.devicePreview?.videoDeviceId)
    }
  } catch (err) {
    uni.showToast({ title: err?.message || '操作失败', icon: 'none' })
  } finally {
    videoLoading.value = false
  }
}

const openInvite = () => { inviteVisible.value = true }
const openMembers = () => { memberVisible.value = true }
const openChat = () => {
  // Task 14 会议聊天入口；当前阶段仅提示
  uni.showToast({ title: '聊天功能即将上线', icon: 'none' })
}

const onKickMember = async (uid) => {
  const confirmed = await new Promise(resolve => {
    uni.showModal({
      title: '踢出成员',
      content: `确认将 ${nameOf(uid)} 移出会议？`,
      success: (r) => resolve(r.confirm),
      fail: () => resolve(false)
    })
  })
  if (!confirmed) return
  try {
    await meetingStore.kickMember(uid)
    uni.showToast({ title: '已踢出', icon: 'success' })
  } catch (err) {
    uni.showToast({ title: err?.message || '操作失败', icon: 'none' })
  }
}

const onTransferHost = async (uid) => {
  const confirmed = await new Promise(resolve => {
    uni.showModal({
      title: '转让主持人',
      content: `确认将主持人身份转让给 ${nameOf(uid)}？`,
      success: (r) => resolve(r.confirm),
      fail: () => resolve(false)
    })
  })
  if (!confirmed) return
  try {
    await meetingStore.transferHost(uid)
    uni.showToast({ title: '已转让', icon: 'success' })
  } catch (err) {
    uni.showToast({ title: err?.message || '操作失败', icon: 'none' })
  }
}

const onLeaveClick = () => { leaveDialogVisible.value = true }

const confirmLeaveSelf = async () => {
  leaveDialogVisible.value = false
  try {
    await meetingStore.leave()
  } finally {
    redirectHome()
  }
}

const confirmEndOrLeave = async () => {
  leaveDialogVisible.value = false
  try {
    if (isHost.value) {
      await meetingStore.endMeeting()
      // room.ended WS 广播会触发 store 内 _onRoomEnded 清理 state；保险起见再 leave 一次容错
      await meetingStore.leave().catch(() => {})
    } else {
      await meetingStore.leave()
    }
  } catch (err) {
    uni.showToast({ title: err?.message || '操作失败', icon: 'none' })
  } finally {
    redirectHome()
  }
}

const redirectHome = () => {
  uni.reLaunch({ url: '/pages/index/index' })
}

// ============ 生命周期 ============

let stateStopWatch = null

onLoad((query) => {
  // 刷新页面 / 直链访问 room 时 store 为空，回跳 join 避免白屏
  if (!meetingStore.isInMeeting) {
    const paramCode = (query?.code || '').replace(/\D/g, '')
    uni.redirectTo({ url: `/pages/meeting/join${paramCode ? `?code=${paramCode}` : ''}` })
  }
})

onMounted(() => {
  const startIso = meetingStore.currentRoom?.started_at || meetingStore.currentRoom?.created_at
  startedAt.value = startIso ? Date.parse(startIso) : Date.now()
  timerHandle = setInterval(() => { nowTs.value = Date.now() }, 1000)

  // 会议被主持人/后端结束（ENDED 状态）时，弹提示并回首页
  stateStopWatch = watch(() => meetingStore.localState, (s) => {
    if (s === MEETING_LOCAL_STATE_ENDED) {
      const reason = meetingStore.lastEndedReason
      const label = MEETING_ENDED_REASON_LABEL?.[reason] || '会议已结束'
      uni.showToast({ title: label, icon: 'none' })
      setTimeout(redirectHome, 1200)
    }
  })
})

onBeforeUnmount(() => {
  if (timerHandle) {
    clearInterval(timerHandle)
    timerHandle = null
  }
  if (stateStopWatch) { stateStopWatch(); stateStopWatch = null }
})

onUnload(() => {
  // 物理返回 / reLaunch 时兜底清理，避免 store 残留脏状态
  if (meetingStore.isInMeeting && meetingStore.localState !== MEETING_LOCAL_STATE_LEAVING) {
    meetingStore.leave().catch(() => {})
  }
})
</script>

<style scoped>
.room {
  position: fixed;
  inset: 0;
  display: flex;
  flex-direction: column;
  background: #0F172A;
  color: #F3F4F6;
}

.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: calc(16rpx + env(safe-area-inset-top)) 24rpx 16rpx;
  gap: 16rpx;
  background: linear-gradient(to bottom, rgba(0, 0, 0, 0.55), rgba(0, 0, 0, 0));
  position: relative;
  z-index: 5;
}

.top-left {
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
  flex: 1;
}
.meeting-title {
  font-size: 28rpx;
  font-weight: 600;
  color: #F9FAFB;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.code-pill {
  display: inline-flex;
  align-items: center;
  gap: 8rpx;
  padding: 4rpx 14rpx;
  background: rgba(255, 255, 255, 0.12);
  border-radius: 999rpx;
  color: #F3F4F6;
  font-size: 22rpx;
  cursor: pointer;
  transition: background-color 0.15s ease;
  width: fit-content;
}
.code-pill:hover { background: rgba(255, 255, 255, 0.2); }
.code-text {
  font-family: 'SF Mono', 'Menlo', monospace;
  letter-spacing: 1rpx;
}

.top-right {
  display: flex;
  align-items: center;
  gap: 12rpx;
  flex-shrink: 0;
}
.state-pill {
  display: inline-flex;
  align-items: center;
  gap: 6rpx;
  padding: 4rpx 12rpx;
  border-radius: 999rpx;
  font-size: 20rpx;
  background: rgba(16, 185, 129, 0.2);
  color: #86EFAC;
}
.state-pill.state-reconnecting {
  background: rgba(245, 158, 11, 0.25);
  color: #FBBF24;
}
.state-pill.state-connecting { background: rgba(59, 130, 246, 0.2); color: #93C5FD; }
.state-pill.state-leaving, .state-pill.state-ended {
  background: rgba(107, 114, 128, 0.3);
  color: #D1D5DB;
}
.state-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 8rpx currentColor;
}
.state-pill.state-reconnecting .state-dot,
.state-pill.state-connecting .state-dot {
  animation: pulse 1.2s infinite ease-in-out;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}
.timer {
  font-size: 22rpx;
  font-family: 'SF Mono', 'Menlo', monospace;
  color: rgba(255, 255, 255, 0.75);
  letter-spacing: 1rpx;
}

.body {
  flex: 1;
  min-height: 0;
  background: #0B1220;
  overflow: hidden;
}

/* 离会弹窗 */
.leave-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 220;
  padding: 32rpx;
}
.leave-modal {
  width: 100%;
  max-width: 560rpx;
  background: #FFFFFF;
  color: #111827;
  border-radius: 24rpx;
  padding: 36rpx 32rpx 28rpx;
  box-shadow: 0 24rpx 48rpx rgba(0, 0, 0, 0.35);
}
.leave-title {
  font-size: 32rpx;
  font-weight: 600;
  margin-bottom: 16rpx;
}
.leave-desc {
  font-size: 26rpx;
  color: #4B5563;
  line-height: 1.6;
  margin-bottom: 32rpx;
}
.leave-actions {
  display: flex;
  gap: 16rpx;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.leave-btn {
  all: unset;
  cursor: pointer;
  padding: 18rpx 32rpx;
  border-radius: 12rpx;
  font-size: 26rpx;
  font-weight: 600;
  transition: background-color 0.12s ease;
}
.leave-cancel {
  background: #F3F4F6;
  color: #374151;
}
.leave-cancel:hover { background: #E5E7EB; }
.leave-leave {
  background: #DBEAFE;
  color: #1D4ED8;
}
.leave-leave:hover { background: #BFDBFE; }
.leave-danger {
  background: #EF4444;
  color: #FFFFFF;
}
.leave-danger:hover { background: #DC2626; }
</style>
