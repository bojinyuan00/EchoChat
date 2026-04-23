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

    <!-- 主区：左视频 + 右侧聊天面板（仅 chatVisible 时侧栏展开） -->
    <view class="body">
      <view ref="videoCol" class="video-col">
        <VideoGrid :tiles="gridTiles">
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
              :is-speaking="!!speakingMap[tile.userId]"
            />
          </template>
        </VideoGrid>

        <!-- 自视频浮窗（Task 15）：仅桌面端浮窗模式开启时显示；图钉切换后本地 tile 回到网格里 -->
        <SelfVideoFloat
          v-if="floatEnabled && selfTile"
          :user-id="selfTile.userId"
          :name="selfTile.name"
          :is-host="selfTile.isHost"
          :audio-enabled="selfTile.audioEnabled"
          :video-enabled="selfTile.videoEnabled"
          :audio-track="selfTile.audioTrack"
          :video-track="selfTile.videoTrack"
          :is-speaking="!!speakingMap[selfTile.userId]"
          :container="videoCol"
          @pin-click="togglePin"
        />
      </view>
      <view class="chat-col" :class="{ open: chatVisible }">
        <ChatPanel
          :visible="chatVisible"
          :messages="chatMessages"
          :current-user-id="myUserId"
          :display-name-map="displayNameMap"
          :has-more="chatHasMore"
          :loading-more="chatLoadingMore"
          @close="closeChat"
          @send="onChatSend"
          @load-more="onChatLoadMore"
          @request-initial-load="onChatRequestInitialLoad"
        />
      </view>
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
      :all-muted="allMuted"
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
      @mute-member="onMuteMember"
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
import ChatPanel from '@/components/meeting/ChatPanel.vue'
import SelfVideoFloat from '@/components/meeting/SelfVideoFloat.vue'

const meetingStore = useMeetingStore()
const userStore = useUserStore()

const memberVisible = ref(false)
const inviteVisible = ref(false)
const chatVisible = ref(false)
const leaveDialogVisible = ref(false)
const audioLoading = ref(false)
const videoLoading = ref(false)
const chatLoadingMore = ref(false)
const networkLevel = ref(4) // 持续接入 getStats 后动态更新（Task 16 进一步收敛）
const startedAt = ref(0)
const nowTs = ref(Date.now())
let timerHandle = null

/** 视频区容器引用：传给 SelfVideoFloat 计算吸附边界 */
const videoCol = ref(null)

/** 说话者映射由 meeting store 探测出（Task 15），浮窗和网格都订阅它来驱动流光动效 */
const speakingMap = computed(() => meetingStore.speakingMap)

/** 全员静音标记，用于 Task 15 工具栏冷色氛围（允许单人时恒为 false） */
const allMuted = computed(() => meetingStore.isAllMuted)

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
const chatMessages = computed(() => meetingStore.chatMessages)
const chatHasMore = computed(() => meetingStore.chatHasMore)
const unreadChatCount = computed(() => meetingStore.chatUnreadCount)

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
    let audioTrack = null
    let videoTrack = null
    if (isLocal) {
      audioTrack = meetingStore.getLocalTrack('audio')
      videoTrack = meetingStore.getLocalTrack('video')
    } else {
      audioTrack = meetingStore.getRemoteTrack(p.user_id, 'audio')
      videoTrack = meetingStore.getRemoteTrack(p.user_id, 'video')
    }
    // Task 15 回归修复：tile 是否渲染 <video>/<audio> 以 track 实际存在为准
    // 参会者的 audio_enabled/video_enabled 仅作为 MemberPanel 状态图标色彩依据，
    // 不再决定 VideoTile 媒体元素的挂载；否则一旦 state 字段滞后就会屏蔽实际画面
    const tile = {
      key: `u_${p.user_id}`,
      userId: p.user_id,
      name: isLocal ? '你' : nameOf(p.user_id),
      isLocal,
      isHost: p.user_id === hostId.value,
      audioEnabled: isLocal ? audioEnabled.value : !!audioTrack,
      videoEnabled: isLocal ? videoEnabled.value : !!videoTrack,
      audioTrack,
      videoTrack
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

// ============ Task 15 浮窗模式 ============

/**
 * 浮窗开关：从 store.uiPrefs 读取，默认桌面端开启浮窗
 * 点击图钉按钮时 togglePin 切换，本地 tile 会在浮窗 <-> 网格间迁移
 */
const floatEnabled = computed(() => meetingStore.uiPrefs?.selfVideoFloat !== false)

/**
 * 自视频 tile：浮窗模式下从 tiles 中剥离出来单独挂载
 * 若 floatEnabled=false 则返回 null，自视频仍保留在 Grid 里
 */
const selfTile = computed(() => {
  if (!floatEnabled.value) return null
  return tiles.value.find(t => t.isLocal) || null
})

/**
 * 网格 tiles：浮窗开启时剔除本地 tile（只渲染远端），否则保持原始
 * 这样 VideoGrid 的 layout-1/2/3/N 档位判断基于远端人数，符合用户认知
 */
const gridTiles = computed(() => {
  if (!floatEnabled.value) return tiles.value
  return tiles.value.filter(t => !t.isLocal)
})

const togglePin = () => {
  meetingStore.uiPrefs.selfVideoFloat = !meetingStore.uiPrefs.selfVideoFloat
}

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

const openInvite = () => { inviteVisible.value = !inviteVisible.value }
const openMembers = () => { memberVisible.value = !memberVisible.value }
const openChat = () => {
  // 点击工具栏聊天按钮时进行切换：已打开 → 关闭；未打开 → 打开并标记已读
  if (chatVisible.value) {
    chatVisible.value = false
    return
  }
  chatVisible.value = true
  meetingStore.markChatsRead()
}
const closeChat = () => { chatVisible.value = false }

watch(unreadChatCount, (count) => {
  if (count > 0 && chatVisible.value) {
    meetingStore.markChatsRead()
  }
})

const onChatSend = async (content) => {
  try {
    await meetingStore.sendChat(content)
  } catch (err) {
    uni.showToast({ title: err?.message || '发送失败', icon: 'none' })
    throw err
  }
}

const onChatLoadMore = async () => {
  if (chatLoadingMore.value) return
  chatLoadingMore.value = true
  try {
    await meetingStore.loadMoreChats(30)
  } catch (err) {
    uni.showToast({ title: err?.message || '加载失败', icon: 'none' })
  } finally {
    chatLoadingMore.value = false
  }
}

const onChatRequestInitialLoad = async () => {
  if (meetingStore.chatHistoryLoaded) return
  await onChatLoadMore()
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

const onMuteMember = async ({ userId: uid, mute }) => {
  try {
    await meetingStore.muteMember(uid, mute)
    uni.showToast({ title: mute ? '已请对方静音' : '已请对方开麦', icon: 'none' })
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
  /* 使用 fixed + 100vw/100vh 双重保险：
     某些祖先（uni-page-wrapper）带 transform 时 fixed 会相对该祖先而非视口；
     显式 100vw/100vh 可保证始终铺满整个浏览器窗口。 */
  position: fixed;
  inset: 0;
  width: 100vw;
  height: 100vh;
  min-width: 100vw;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: #0F172A;
  color: #F3F4F6;
  /* 必须高于 CustomTabBar（z-index:999），否则从带 tabBar 的页面
     navigateTo 进会议室后，上一页遗留的 tabBar 会悬浮覆盖 toolbar
     导致静音/摄像头/邀请/成员/聊天按钮全部被拦截 */
  z-index: 1000;
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
  flex-shrink: 0;
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
  display: flex;
  flex-direction: row;
}
.video-col {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  overflow: hidden;
}
.chat-col {
  width: 0;
  min-height: 0;
  flex-shrink: 0;
  overflow: hidden;
  transition: width 0.24s ease;
  border-left: 1rpx solid rgba(255, 255, 255, 0.06);
  background: #1F2937;
}
.chat-col.open { width: 640rpx; }

@media (max-width: 750px) {
  /* 小屏幕：聊天面板改为全屏浮层，避免挤压视频 */
  .chat-col { position: fixed; top: 0; right: 0; bottom: 0; z-index: 205; }
  .chat-col.open { width: 100vw; }
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
  position: relative;
}
/* 同样抹除 uni-app H5 自动注入的 ::after 遮罩（inset: 0 -1200px -80px 0），
   否则 cancel 按钮的 ::after 会覆盖到右侧确认按钮区域，
   导致点击 "取消" 命中 "结束会议" 逻辑。 */
.leave-btn::after {
  content: none !important;
  display: none !important;
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
