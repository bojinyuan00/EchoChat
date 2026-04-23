/**
 * 会议模块 Pinia Store
 *
 * 职责：
 * - 管理当前入会状态（本地状态机：idle/joining/connecting/connected/reconnecting/leaving/ended）
 * - 维护房间信息、参与者列表、本地/远端媒体 Producer/Consumer
 * - 串联 REST API + WebSocket 信令 + mediasoup-client
 * - 监听 8 个服务端广播事件，实时更新 state
 *
 * 设计要点（Task 9）：
 * - WS 事件监听仅在进入会议室时注册、离会时注销（决策 Q3=a3_on_enter_room）
 * - mediasoup 的 Device/Transport/Producer/Consumer 用 markRaw 避免 Vue 代理
 * - mediasoup-client 仅 H5 可用，其他平台调用 enterRoom 会抛错
 *
 * 上层调用模式（极简）：
 *   const meetingStore = useMeetingStore()
 *   await meetingStore.createAndEnter({ title })      // 创建并入会
 *   await meetingStore.joinAndEnter(roomCode, password) // 加入已有会议
 *   await meetingStore.startLocalAudio()
 *   await meetingStore.startLocalVideo()
 *   await meetingStore.leave()                         // 离会
 */

import { defineStore } from 'pinia'
import { ref, computed, markRaw, reactive } from 'vue'

import meetingApi from '@/api/meeting'
import wsService from '@/services/websocket'
import { createMediaEngine } from '@/utils/mediasoup-client'
import { useUserStore } from '@/store/user'

import {
  MEETING_LOCAL_STATE_IDLE,
  MEETING_LOCAL_STATE_JOINING,
  MEETING_LOCAL_STATE_CONNECTING,
  MEETING_LOCAL_STATE_CONNECTED,
  MEETING_LOCAL_STATE_RECONNECTING,
  MEETING_LOCAL_STATE_LEAVING,
  MEETING_LOCAL_STATE_ENDED,
  MEETING_WS_ROOM_JOIN,
  MEETING_WS_ROOM_ENDED,
  MEETING_WS_MEMBER_JOINED,
  MEETING_WS_MEMBER_LEFT,
  MEETING_WS_MEMBER_STATE_CHANGED,
  MEETING_WS_MEMBER_KICKED,
  MEETING_WS_MEMBER_PRODUCER_NEW,
  MEETING_WS_HOST_CHANGED,
  MEETING_WS_CHAT_MESSAGE,
  MEETING_ROLE_HOST,
  MEETING_ENDED_REASON_LABEL,
  MEETING_ENDED_REASON_KICKED,
  MEETING_HOST_AUTO_REASON_LABEL
} from '@/constants/meeting'

/** 判断当前平台是否支持 mediasoup-client（仅 H5） */
const isMediaSupported = () => {
  // #ifdef H5
  return true
  // #endif
  // #ifndef H5
  return false
  // #endif
}

export const useMeetingStore = defineStore('meeting', () => {
  // ==================== State ====================

  /** 本地状态机：MEETING_LOCAL_STATE_* */
  const localState = ref(MEETING_LOCAL_STATE_IDLE)

  /** 当前会议房间 DTO（REST 响应中的 room 字段） */
  const currentRoom = ref(null)

  /** 当前用户的 participant DTO */
  const currentParticipant = ref(null)

  /** Router ID（Node mediasoup Router）—— 仅日志/调试用 */
  const routerID = ref('')

  /** 活跃参与者列表（含当前用户，按加入时间正序） */
  const participants = ref([])

  /** 会议聊天消息列表（按 ID 升序，最新在尾部） */
  const chatMessages = ref([])

  /** 会议聊天还有更早消息可加载（用于"加载更多"按钮） */
  const chatHasMore = ref(true)

  /** 会议聊天未读计数（来自他人且 ChatPanel 未打开期间累积） */
  const chatUnreadCount = ref(0)

  /** 会议聊天首次懒加载完成标记 */
  const chatHistoryLoaded = ref(false)

  /** 最近一次会议结束原因（room.ended_reason 或 room.ended.reason） */
  const lastEndedReason = ref('')

  /** 最近一次自动主持人转让原因（auto_reason） */
  const lastAutoHostReason = ref('')

  /** 本地媒体状态 */
  const localAudioEnabled = ref(false)
  const localVideoEnabled = ref(false)

  /**
   * 当前用户的本地 Producer 索引：{ audio: producerId?, video: producerId? }
   * 用于启停麦克风/摄像头
   */
  const localProducers = reactive({ audio: null, video: null })

  /**
   * 远端 Consumer 索引：{ [userId]: { audio?: Consumer, video?: Consumer, producerIds: Set } }
   * 使用 reactive 但 Consumer 实例本身是 markRaw，不会被深度代理
   */
  const remoteConsumers = reactive({})

  /**
   * 创建会议表单草稿（Task 10）
   * create.vue 填完表单后暂存于此，preview.vue 点"加入会议"时才真正调用 createAndEnter。
   * 字段：{ title, password?, enter_muted, allow_chat }
   */
  const draftCreatePayload = ref(null)

  /**
   * 加入会议密码草稿（Task 16 P0-4 修复）
   * 原实现把密码明文拼在 uni.navigateTo 的 URL query 里，浏览器历史 / DevTools 都会留痕，
   * 违背设计文档 §2.2.1 "邀请链接仅携带 token 不携带密码"的约束。
   * 新流程：join.vue 填完密码 → 写入本字段 → 只跳转 `?mode=join&code=XXX-XXX-XXX` → preview.vue 读取后立即清空。
   * 字段：{ code, password } 或 null
   */
  const draftJoinPayload = ref(null)

  /**
   * 设备预览页的设备选择与显示名（Task 10）
   * preview.vue 选完设备后写入，入会时作为 mediaPrefs 传给 createAndEnter / joinAndEnter
   * 字段：{ audioDeviceId, videoDeviceId, speakerDeviceId, displayName, startAudio, startVideo }
   */
  const devicePreview = ref({
    audioDeviceId: '',
    videoDeviceId: '',
    speakerDeviceId: '',
    displayName: '',
    startAudio: true,
    startVideo: true
  })

  /**
   * 标记：当前在 Store 内的事件监听器集合（进房注册，离房注销）
   * Key 是事件名，Value 是监听函数引用
   */
  const _listeners = new Map()

  /** MediaEngine 实例（H5 才有值），用 markRaw 包裹避免 Vue 深度代理 */
  let _engine = null

  /**
   * 说话者探测：{ [userId]: boolean }，由 UI 订阅驱动 VideoTile.speaking 边框
   * Task 15 原创特色 1——EchoChat 会议专属流光轮廓
   */
  const speakingMap = reactive({})

  /**
   * UI 偏好（内存态，不持久化到 storage）
   * - selfVideoFloat: 自视频是否走浮窗模式（桌面端默认 true）
   */
  const uiPrefs = reactive({
    selfVideoFloat: true
  })

  /**
   * 说话者探测内部状态：handle + 本地 WebAudio 实例
   * 所有字段均为 H5 浏览器 API，非 H5 平台永远保持 null
   */
  let _speakingTimer = null
  let _localAudioCtx = null
  let _localAnalyser = null
  let _localSource = null
  /** { [userId]: { inCount: number, outCount: number } }，用于防抖 1-in / 2-out */
  const _speakingDebounce = {}

  /**
   * Task 16 资源清理专项：_broadcastSelfState 重试期间的 setTimeout 句柄集合
   * 离会 / 结束会议时统一 clearTimeout，避免会议结束后仍有 stale 重试逻辑在 event loop 排队
   */
  const _pendingBroadcastTimers = new Set()

  // ==================== Getters ====================

  const isInMeeting = computed(() => {
    return localState.value !== MEETING_LOCAL_STATE_IDLE &&
           localState.value !== MEETING_LOCAL_STATE_ENDED
  })

  const isConnected = computed(() => localState.value === MEETING_LOCAL_STATE_CONNECTED)

  const isHost = computed(() => {
    const userStore = useUserStore()
    const uid = userStore.userInfo?.id
    return currentRoom.value && uid && currentRoom.value.host_id === uid
  })

  const activeParticipants = computed(() => {
    return participants.value.filter(p => p.is_active)
  })

  /**
   * 全员静音标记：用于 Task 15 原创特色 4——静音氛围色
   * 规则：仅当参与者 ≥ 2 人且所有人 audio_enabled=false 时 true
   * 单人会议不触发（独自一人不算"全员静音氛围"）
   */
  const isAllMuted = computed(() => {
    const list = activeParticipants.value
    if (list.length < 2) return false
    const userStore = useUserStore()
    const myUid = userStore.userInfo?.id
    return list.every(p => {
      // 本地 audio 以 localAudioEnabled 为准（后端不会给自己广播 state.changed）
      if (myUid && p.user_id === myUid) return !localAudioEnabled.value
      return p.audio_enabled === false
    })
  })

  // ==================== 私有工具 ====================

  const _log = (level, ...args) => {
    const fn = console[level] || console.log
    fn.call(console, ...args)
  }

  /**
   * 识别"已在其他会议中"这类僵尸残留错误，加上提示前端可调用 cleanupStaleMeetings 的 hint
   * 保留原始 err 对象引用，仅附加 staleMeetingHint 字段供上层 UI 判定
   */
  const _enrichStaleMeetingError = (err) => {
    if (!err) return err
    const msg = err.message || err.data?.message || ''
    if (msg.includes('已在其他会议中') || msg.includes('已在此会议中')) {
      err.staleMeetingHint = true
    }
    return err
  }

  const _reset = () => {
    localState.value = MEETING_LOCAL_STATE_IDLE
    currentRoom.value = null
    currentParticipant.value = null
    routerID.value = ''
    participants.value = []
    chatMessages.value = []
    chatHasMore.value = true
    chatUnreadCount.value = 0
    chatHistoryLoaded.value = false
    localAudioEnabled.value = false
    localVideoEnabled.value = false
    localProducers.audio = null
    localProducers.video = null
    Object.keys(remoteConsumers).forEach(k => delete remoteConsumers[k])
    _engine = null
  }

  /** 统一注册 WS 监听（进房调用一次） */
  const _registerListeners = () => {
    const handlers = {
      [MEETING_WS_ROOM_ENDED]: _onRoomEnded,
      [MEETING_WS_MEMBER_JOINED]: _onMemberJoined,
      [MEETING_WS_MEMBER_LEFT]: _onMemberLeft,
      [MEETING_WS_MEMBER_STATE_CHANGED]: _onMemberStateChanged,
      [MEETING_WS_MEMBER_KICKED]: _onMemberKicked,
      [MEETING_WS_MEMBER_PRODUCER_NEW]: _onProducerNew,
      [MEETING_WS_HOST_CHANGED]: _onHostChanged,
      [MEETING_WS_CHAT_MESSAGE]: _onChatMessage,
      // WS 连接生命周期：用于断线/重连期间维持会议会话，避免"连接断开 → 后端失去 roomCode 绑定 → 广播丢失"
      _connected: _onWsReconnected,
      _disconnected: _onWsDisconnected
    }
    Object.entries(handlers).forEach(([event, handler]) => {
      wsService.on(event, handler)
      _listeners.set(event, handler)
    })
  }

  const _unregisterListeners = () => {
    _listeners.forEach((handler, event) => {
      wsService.off(event, handler)
    })
    _listeners.clear()
  }

  // ==================== WS 事件回调 ====================

  const _onRoomEnded = (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    lastEndedReason.value = data.reason || ''
    _log('info', '[Meeting] 会议已结束', data.reason, MEETING_ENDED_REASON_LABEL[data.reason])
    _cleanupMedia()
    // 清理 producer/consumer 索引与本地开关，防止下次 createAndEnter/joinAndEnter 时
    // startLocalAudio/Video 因 `if (localProducers.audio) return` 被残留 ID 拦下，
    // 表现为"重新发起会议后看不到自己的画面"
    localProducers.audio = null
    localProducers.video = null
    localAudioEnabled.value = false
    localVideoEnabled.value = false
    Object.keys(remoteConsumers).forEach(k => delete remoteConsumers[k])
    localState.value = MEETING_LOCAL_STATE_ENDED
    _unregisterListeners()
  }

  /**
   * 合并 REST getRoom 返回的 participants 与当前内存状态
   * - REST DTO 目前不包含 audio_enabled/video_enabled（后端 DTO 未暴露该字段）
   * - WS 广播驱动的内存状态（p.audio_enabled / p.video_enabled）可能已更新；REST 覆盖会回退
   * - 策略：以 REST 返回的静态属性为主，保留本地已知的音视频开关状态；首次未知时留 undefined
   *   交由 state.changed 广播权威填充，避免误将"未知"冲成 false 引起 UI 错判
   */
  const _mergeParticipantsFromRest = (restList) => {
    const byUid = new Map()
    participants.value.forEach((p) => byUid.set(p.user_id, p))
    return restList.map((p) => {
      const prev = byUid.get(p.user_id)
      const merged = { ...p }
      if (typeof p.audio_enabled !== 'boolean' && prev && typeof prev.audio_enabled === 'boolean') {
        merged.audio_enabled = prev.audio_enabled
      }
      if (typeof p.video_enabled !== 'boolean' && prev && typeof prev.video_enabled === 'boolean') {
        merged.video_enabled = prev.video_enabled
      }
      return merged
    })
  }

  const _onMemberJoined = (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    // 后端广播已带 user_name/user_avatar，直接补齐昵称头像字段即可
    const existed = participants.value.find(p => p.user_id === data.user_id)
    if (existed) {
      if (data.user_name) existed.user_name = data.user_name
      if (data.user_avatar) existed.user_avatar = data.user_avatar
      existed.is_active = true
      return
    }
    // Task 15：新入会者的 audio_enabled/video_enabled 保持 undefined，
    // 等 startLocalAudio/Video 成功后的 meeting.member.state.changed 广播权威填充
    // MemberPanel 对 undefined 按 !p.audio_enabled 解读为 off（灰），语义"尚未开启"与预期一致
    participants.value.push({
      user_id: data.user_id,
      user_name: data.user_name || '',
      user_avatar: data.user_avatar || '',
      joined_at: data.joined_at,
      is_active: true,
      role: 0
    })
  }

  const _onMemberLeft = (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    const idx = participants.value.findIndex(p => p.user_id === data.user_id)
    if (idx >= 0) {
      participants.value[idx].is_active = false
      participants.value[idx].left_reason = data.reason
    }
    _cleanupRemoteUser(data.user_id)
  }

  const _onMemberStateChanged = (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    let p = participants.value.find(p => p.user_id === data.user_id)
    if (!p) {
      // 时序兜底：OnRoomJoin 触发的 existing state.changed 可能早于 REST getRoom 返回
      // 此时 participants 里还没有 data.user_id 条目。建个占位，等后续 REST / member.joined
      // 在 _mergeParticipantsFromRest 里补齐 user_name/avatar 即可，本处先把状态保留住
      p = {
        user_id: data.user_id,
        user_name: '',
        user_avatar: '',
        is_active: true,
        role: 0
      }
      participants.value.push(p)
    }
    if (typeof data.audio_enabled === 'boolean') p.audio_enabled = data.audio_enabled
    if (typeof data.video_enabled === 'boolean') p.video_enabled = data.video_enabled

    // Task 15：主持人通过 meeting.member.state.changed 带 target_user_id 指令静音他人，
    // 后端广播字段名参见 meeting_signal_service.OnMemberStateChanged：
    //   data.user_id = 被操作者（即当前收到广播的用户）
    //   data.changed_by = 发起操作者（主持人）
    const userStore = useUserStore()
    const myUid = userStore.userInfo?.id
    const isTargetMe = myUid && data.user_id === myUid
    const isByOthers = data.changed_by && data.changed_by !== myUid
    if (isTargetMe && isByOthers && typeof data.audio_enabled === 'boolean' && !data.audio_enabled) {
      // 被主持人请求静音 → 立即关闭本地 audio producer
      if (localAudioEnabled.value) {
        stopLocalAudio().catch((err) => _log('warn', '[Meeting] 被动静音失败', err))
      }
      // #ifdef H5
      try { uni.showToast({ title: '主持人请你静音', icon: 'none', duration: 2000 }) } catch {}
      // #endif
    }
  }

  const _onMemberKicked = (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    const userStore = useUserStore()
    if (data.user_id === userStore.userInfo?.id) {
      _log('warn', '[Meeting] 当前用户被踢出会议')
      lastEndedReason.value = MEETING_ENDED_REASON_KICKED
      _cleanupMedia()
      localState.value = MEETING_LOCAL_STATE_ENDED
      _unregisterListeners()
    } else {
      const idx = participants.value.findIndex(p => p.user_id === data.user_id)
      if (idx >= 0) {
        participants.value[idx].is_active = false
        participants.value[idx].left_reason = 'kicked'
      }
      _cleanupRemoteUser(data.user_id)
    }
  }

  /**
   * 收到 meeting.member.producer.new 广播 → 自动订阅
   * 后端在 Producer 创建 / 关闭时都广播本事件，closed=true 表示关闭
   */
  const _onProducerNew = async (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    const userStore = useUserStore()
    if (data.user_id === userStore.userInfo?.id) return

    if (data.closed) {
      _cleanupRemoteProducer(data.user_id, data.producer_id)
      return
    }

    if (!_engine) {
      _log('warn', '[Meeting] producer.new 收到但 MediaEngine 未就绪，跳过')
      return
    }

    try {
      const consumer = await _engine.consume({ producerId: data.producer_id })
      if (!remoteConsumers[data.user_id]) {
        // 外层 slot 保持 reactive 以便 slot.audio/slot.video 赋值能驱动 VideoTile 重渲染；
        // 仅 Consumer 实例和 Set 本身用 markRaw 隔离 Vue 代理
        remoteConsumers[data.user_id] = {
          audio: null,
          video: null,
          producerIds: markRaw(new Set())
        }
      }
      const slot = remoteConsumers[data.user_id]
      slot[consumer.kind] = markRaw(consumer)
      slot.producerIds.add(data.producer_id)

      // track 已在 consumer.track 中，调用方（页面）可从 store 拿 consumer.track 挂到 DOM
      // 这里先做最简化的 resume：页面渲染后若还需等 DOM 就绪，可改为在页面层调 resumeConsumer
      await _engine.resumeConsumer(consumer.id)
    } catch (err) {
      _log('error', '[Meeting] 订阅远端 Producer 失败', err)
    }
  }

  const _onHostChanged = (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    currentRoom.value.host_id = data.new_host_id
    lastAutoHostReason.value = data.auto_reason || ''
    _log('info', '[Meeting] 主持人变更',
      data.new_host_id,
      data.auto_reason ? `(auto: ${MEETING_HOST_AUTO_REASON_LABEL[data.auto_reason]})` : '(manual)')

    const p = participants.value.find(p => p.user_id === data.new_host_id)
    if (p) p.role = MEETING_ROLE_HOST
    const old = participants.value.find(p => p.user_id === data.old_host_id)
    if (old) old.role = 0
  }

  const _onChatMessage = (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    // WS 广播字段名与 REST 响应不同：id vs message_id；统一映射为 id/user_id/content/created_at
    const item = {
      id: data.message_id ?? data.id,
      user_id: data.user_id,
      user_name: data.user_name || '',
      user_avatar: data.user_avatar || '',
      content: data.content,
      created_at: data.created_at
    }
    // 防止重复：相同 id 已存在则忽略（sendChat 响应 + 广播同时到达的兜底）
    if (item.id && chatMessages.value.some(m => m.id === item.id)) return
    chatMessages.value.push(item)
    const userStore = useUserStore()
    if (item.user_id !== userStore.userInfo?.id) {
      chatUnreadCount.value += 1
    }
  }

  /**
   * WS 连接断开：仅当当前已进入会议（CONNECTED/CONNECTING）时切换到 RECONNECTING，
   * 供 UI 展示"重连中…"，避免用户误以为自己已离会。
   */
  const _onWsDisconnected = () => {
    if (!currentRoom.value) return
    if (
      localState.value === MEETING_LOCAL_STATE_CONNECTED ||
      localState.value === MEETING_LOCAL_STATE_CONNECTING
    ) {
      _log('warn', '[Meeting] WS 已断开，进入 reconnecting')
      localState.value = MEETING_LOCAL_STATE_RECONNECTING
    }
  }

  /**
   * WS 重连成功：若当前仍在某个房间内，自动重发 meeting.room.join 以恢复后端 WS↔roomCode 绑定。
   * 后端 WS Handler 对同一用户重连后，需要再次收到 meeting.room.join 才会将该连接加入房间广播组，
   * 否则即使 mediasoup 层仍在线，所有 meeting.* 广播都不会再送达本端。
   */
  const _onWsReconnected = async () => {
    if (!currentRoom.value) return
    // 仅在 RECONNECTING 状态触发（避免首次进房 _onOpen 与 _afterJoined 的首次 join 并发冲突）
    if (localState.value !== MEETING_LOCAL_STATE_RECONNECTING) return
    const roomCode = currentRoom.value.room_code
    try {
      _log('info', '[Meeting] WS 重连后重绑定房间', roomCode)
      await wsService.sendWithAck(MEETING_WS_ROOM_JOIN, { room_code: roomCode })
      // 重连期间可能错过若干广播，重新拉一次详情对齐参与者列表
      try {
        const detail = await meetingApi.getRoom(roomCode)
        if (detail && Array.isArray(detail.participants)) {
          participants.value = _mergeParticipantsFromRest(detail.participants)
        }
      } catch (e) {
        _log('warn', '[Meeting] 重绑定后拉取房间详情失败', e)
      }
      localState.value = MEETING_LOCAL_STATE_CONNECTED
      _log('info', '[Meeting] WS 重绑定房间成功，恢复 connected')
    } catch (err) {
      _log('warn', '[Meeting] WS 重绑定房间失败，保持 reconnecting 等待下次重连', err)
      // 保留 RECONNECTING 状态，下一次 WS 重连会再次触发本回调
    }
  }

  // ==================== 远端媒体清理 ====================

  /**
   * 精确关闭"某用户某个 producer 对应的 consumer"
   *
   * mediasoup-client 的 Consumer 对象自带 `producerId` 字段（transport.consume 时注入），
   * 利用它匹配 slot.audio / slot.video 中归属 producerId 的那一个；
   * 其他 kind 的 consumer 必须保留，否则会出现"关音频连带把视频画面也关掉"的连锁问题。
   *
   * 找不到匹配时（例如 producerId 异常），不再做 MVP 时期的"整槽关闭"兜底，
   * 而是只把 producerIds Set 条目移掉；真正该关的 consumer 由后续 transportclose / 离会清理接管
   */
  const _cleanupRemoteProducer = (userId, producerId) => {
    const slot = remoteConsumers[userId]
    if (!slot) return
    slot.producerIds.delete(producerId)
    ;['audio', 'video'].forEach(kind => {
      const consumer = slot[kind]
      if (!consumer) return
      if (consumer.producerId && consumer.producerId !== producerId) return
      if (!consumer.closed) {
        try { consumer.close() } catch {}
      }
      slot[kind] = null
    })
    if (!slot.audio && !slot.video && slot.producerIds.size === 0) {
      delete remoteConsumers[userId]
    }
  }

  const _cleanupRemoteUser = (userId) => {
    const slot = remoteConsumers[userId]
    if (!slot) return
    ;['audio', 'video'].forEach(kind => {
      if (slot[kind] && !slot[kind].closed) {
        try { slot[kind].close() } catch {}
      }
    })
    delete remoteConsumers[userId]
  }

  const _cleanupMedia = () => {
    _stopSpeakingDetection()
    // Task 16 资源清理：清理 _broadcastSelfState 挂起的 setTimeout 重试
    // 防止会议结束后重试逻辑仍在 event loop 排队造成 memory pin
    _pendingBroadcastTimers.forEach((handle) => {
      try { clearTimeout(handle) } catch {}
    })
    _pendingBroadcastTimers.clear()
    if (_engine) {
      try { _engine.close() } catch (e) { _log('warn', '_cleanupMedia engine.close', e) }
      _engine = null
    }
  }

  // ==================== 说话者探测（Task 15 原创特色 1） ====================

  /**
   * 音量阈值（0-1 线性，RMS / audioLevel 均归一化到此区间）
   * 0.05 为静默阈值，比正常说话（0.15-0.4）低 3-8 倍，可过滤键盘敲击背景噪声
   */
  const _SPEAKING_LEVEL_THRESHOLD = 0.05
  /** 探测轮询间隔（ms），500ms 已足够识别说话停顿 */
  const _SPEAKING_TICK_MS = 500

  /** 读取远端 Consumer 的即时音量（0-1） */
  const _readRemoteAudioLevel = (consumer) => {
    // #ifdef H5
    if (!consumer || !consumer.rtpReceiver) return 0
    try {
      const sources = consumer.rtpReceiver.getSynchronizationSources?.() || []
      if (sources.length === 0) return 0
      // audioLevel 是 0-1 的浮点数（W3C Spec），Chrome/Edge 原生支持
      // Firefox 可能返回 undefined，此时回退为 0（不影响其他远端检测）
      const level = sources[0].audioLevel
      return typeof level === 'number' ? level : 0
    } catch {
      return 0
    }
    // #endif
    // #ifndef H5
    return 0
    // #endif
  }

  /** 读取本地 AnalyserNode 的 RMS（0-1） */
  const _readLocalAudioLevel = () => {
    // #ifdef H5
    if (!_localAnalyser) return 0
    try {
      const buf = new Float32Array(_localAnalyser.fftSize)
      _localAnalyser.getFloatTimeDomainData(buf)
      let sum = 0
      for (let i = 0; i < buf.length; i++) sum += buf[i] * buf[i]
      const rms = Math.sqrt(sum / buf.length)
      // RMS 天然偏小，乘 3 放大到与 audioLevel 相似数量级
      return Math.min(1, rms * 3)
    } catch {
      return 0
    }
    // #endif
    // #ifndef H5
    return 0
    // #endif
  }

  /**
   * 防抖更新：level >= 阈值 1 tick 即判定 speaking=true，
   * level < 阈值连续 2 tick 才 false，避免说话时短暂停顿 UI 闪烁
   */
  const _updateSpeakingWithDebounce = (userId, level) => {
    if (!_speakingDebounce[userId]) {
      _speakingDebounce[userId] = { inCount: 0, outCount: 0 }
    }
    const state = _speakingDebounce[userId]
    const above = level >= _SPEAKING_LEVEL_THRESHOLD
    if (above) {
      state.inCount += 1
      state.outCount = 0
      if (state.inCount >= 1 && !speakingMap[userId]) {
        speakingMap[userId] = true
      }
    } else {
      state.inCount = 0
      state.outCount += 1
      if (state.outCount >= 2 && speakingMap[userId]) {
        speakingMap[userId] = false
      }
    }
  }

  /** 初始化本地 WebAudio 管道（仅在有本地 audio track 时建） */
  const _ensureLocalAnalyser = () => {
    // #ifdef H5
    const track = getLocalTrack('audio')
    if (!track) {
      _teardownLocalAnalyser()
      return
    }
    // 若已有管道且 track 未变，直接复用；否则重建（例：麦克风切换设备）
    if (_localAnalyser && _localSource && _localSource.mediaStream?.getAudioTracks()[0]?.id === track.id) {
      return
    }
    _teardownLocalAnalyser()
    try {
      const AC = window.AudioContext || window.webkitAudioContext
      if (!AC) return
      _localAudioCtx = new AC()
      const stream = new MediaStream([track])
      _localSource = _localAudioCtx.createMediaStreamSource(stream)
      _localAnalyser = _localAudioCtx.createAnalyser()
      _localAnalyser.fftSize = 1024
      _localSource.connect(_localAnalyser)
    } catch (e) {
      _log('warn', '[Meeting] 本地 AnalyserNode 初始化失败', e)
      _teardownLocalAnalyser()
    }
    // #endif
  }

  const _teardownLocalAnalyser = () => {
    // #ifdef H5
    try { _localSource?.disconnect?.() } catch {}
    try { _localAnalyser?.disconnect?.() } catch {}
    try { _localAudioCtx?.close?.() } catch {}
    _localSource = null
    _localAnalyser = null
    _localAudioCtx = null
    // #endif
  }

  /** 500ms 轮询：计算所有远端 + 本地的音量并防抖更新 speakingMap */
  const _speakingTick = () => {
    const userStore = useUserStore()
    const myUid = userStore.userInfo?.id

    // 远端：遍历所有有 audio consumer 的用户
    Object.keys(remoteConsumers).forEach(uidStr => {
      const uid = Number(uidStr)
      const slot = remoteConsumers[uid]
      const level = slot?.audio ? _readRemoteAudioLevel(slot.audio) : 0
      _updateSpeakingWithDebounce(uid, level)
    })

    // 本地：仅在本地 audio 开启时探测
    if (myUid) {
      if (localAudioEnabled.value) {
        _ensureLocalAnalyser()
        const level = _readLocalAudioLevel()
        _updateSpeakingWithDebounce(myUid, level)
      } else {
        // 关麦后直接清零并销毁 AnalyserNode
        if (speakingMap[myUid]) speakingMap[myUid] = false
        if (_speakingDebounce[myUid]) _speakingDebounce[myUid] = { inCount: 0, outCount: 0 }
        _teardownLocalAnalyser()
      }
    }
  }

  const _startSpeakingDetection = () => {
    if (_speakingTimer) return
    // #ifdef H5
    _speakingTimer = setInterval(_speakingTick, _SPEAKING_TICK_MS)
    // #endif
  }

  const _stopSpeakingDetection = () => {
    if (_speakingTimer) {
      clearInterval(_speakingTimer)
      _speakingTimer = null
    }
    _teardownLocalAnalyser()
    Object.keys(speakingMap).forEach(k => delete speakingMap[k])
    Object.keys(_speakingDebounce).forEach(k => delete _speakingDebounce[k])
  }

  // ==================== 进入会议（公共私有） ====================

  /**
   * 入会后的共同后置流程：加载 Device、创建两条 Transport、发 room.join 信令、订阅在线 Producer
   * @param {Object} opts
   * @param {Object} opts.rtpCapabilities - Router RTP capabilities
   * @param {Object} [opts.mediaPrefs] - 设备预览页传入的媒体偏好，入会成功后自动推流
   *   - startAudio: boolean 是否默认开麦
   *   - startVideo: boolean 是否默认开摄像头
   *   - audioDeviceId / videoDeviceId: 指定采集设备（可空→走系统默认）
   */
  const _afterJoined = async ({ rtpCapabilities, mediaPrefs }) => {
    if (!isMediaSupported()) {
      throw new Error('当前平台不支持会议（仅支持 H5 浏览器）')
    }
    localState.value = MEETING_LOCAL_STATE_CONNECTING

    const userStore = useUserStore()
    const uid = userStore.userInfo?.id
    if (!uid) throw new Error('当前用户未登录')
    _engine = markRaw(createMediaEngine({
      roomCode: currentRoom.value.room_code,
      userId: uid,
      sendWithAck: wsService.sendWithAck.bind(wsService),
      logger: _log
    }))
    await _engine.loadDevice(rtpCapabilities)

    _registerListeners()

    // 提前拉取参与者列表：让 OnRoomJoin 触发的 existing state.changed / producer.new
    // 到达时 participants 已就绪，避免后入者的 _onMemberStateChanged 找不到 participant 而丢状态
    try {
      const detail = await meetingApi.getRoom(currentRoom.value.room_code)
      if (detail && Array.isArray(detail.participants)) {
        participants.value = _mergeParticipantsFromRest(detail.participants)
      }
    } catch (e) {
      _log('warn', '[Meeting] 拉取参与者详情失败，保留广播缓存', e)
    }

    // 告知后端 WS 我已绑定本房间（用于 WS 连接 ↔ roomCode 映射）
    // 后端在此事件内部异步 pushExistingRoomState，补推房间其他用户的 producer.new 与 state.changed
    await wsService.sendWithAck(MEETING_WS_ROOM_JOIN, {
      room_code: currentRoom.value.room_code
    })

    // 按需先创建 sendTransport + recvTransport，避免页面点"开麦"时首帧有额外抖动
    await _engine.ensureSendTransport()
    await _engine.ensureRecvTransport()

    localState.value = MEETING_LOCAL_STATE_CONNECTED

    // 启动说话者探测（Task 15）：500ms 轮询，远端读 RTP audioLevel，本地用 WebAudio RMS
    _startSpeakingDetection()

    // mediaPrefs 自动推流：失败仅告警，不阻断入会流程，允许用户在会议室内手动重试开麦/开摄像头
    // 预览页已申请过 track 时会通过 mediaPrefs.audioTrack/videoTrack 透传过来，避免 iOS 二次 getUserMedia
    if (mediaPrefs) {
      if (mediaPrefs.startAudio) {
        try {
          await startLocalAudio(mediaPrefs.audioDeviceId, mediaPrefs.audioTrack)
        } catch (e) {
          _log('warn', '[Meeting] 自动开麦失败', e)
          // #ifdef H5
          try { uni.showToast({ title: `自动开麦失败：${e?.message || e?.name || e}`, icon: 'none', duration: 3500 }) } catch {}
          // #endif
        }
      }
      if (mediaPrefs.startVideo) {
        try {
          await startLocalVideo(mediaPrefs.videoDeviceId, mediaPrefs.videoTrack)
        } catch (e) {
          _log('warn', '[Meeting] 自动开摄像头失败', e)
          // #ifdef H5
          try { uni.showToast({ title: `自动开摄像头失败：${e?.message || e?.name || e}`, icon: 'none', duration: 3500 }) } catch {}
          // #endif
        }
      }
    }
  }

  // ==================== Actions：会议生命周期 ====================

  /**
   * 创建即时会议并立即入会
   * @param {Object} payload - { title, password?, max_members?, enter_muted?, allow_chat? }
   * @param {Object} [mediaPrefs] - 设备预览页的媒体偏好，入会成功后自动推流（可空）
   */
  const createAndEnter = async (payload, mediaPrefs, _retried = false) => {
    if (isInMeeting.value) {
      throw new Error('你当前已在其他会议中')
    }
    // 上一次会议若以 ENDED 收尾（被主持人/后端结束），Store 中仍会残留 participants/remoteConsumers/
    // devicePreview 等旧状态；显式 _reset 一次避免污染新会议（典型症状：再次入会看不到自己）
    _reset()
    localState.value = MEETING_LOCAL_STATE_JOINING
    try {
      // 首次调用时抑制失败 toast：失败后若是 stale 残留会自动清理 + 重试，对用户无感
      const resp = await meetingApi.createRoom(payload, _retried ? {} : { silent: true })
      currentRoom.value = resp.room
      routerID.value = resp.router_id || ''
      // 创建者 host participant 尚未通过 REST 返回，用本地 userStore 占位
      const userStore = useUserStore()
      currentParticipant.value = {
        user_id: userStore.userInfo?.id,
        role: MEETING_ROLE_HOST,
        is_active: true,
        joined_at: new Date().toISOString()
      }
      await _afterJoined({ rtpCapabilities: resp.rtp_capabilities, mediaPrefs })
      return currentRoom.value
    } catch (err) {
      _log('error', '[Meeting] 创建并入会失败', err)
      _reset()
      const enriched = _enrichStaleMeetingError(err)
      // 刷新/崩溃残留：后端仍标记当前用户为活跃成员 → 自动清理后重试一次
      if (enriched?.staleMeetingHint && !_retried) {
        _log('warn', '[Meeting] 检测到僵尸会议残留，自动清理并重试创建')
        try { await cleanupStaleMeetings() } catch (e) { _log('warn', '[Meeting] 自动清理失败', e) }
        return createAndEnter(payload, mediaPrefs, true)
      }
      throw enriched
    }
  }

  /**
   * 加入已有会议
   * @param {string} roomCode
   * @param {string} [password]
   * @param {Object} [mediaPrefs] - 设备预览页的媒体偏好（可空）
   */
  const joinAndEnter = async (roomCode, password = '', mediaPrefs, _retried = false) => {
    if (isInMeeting.value) {
      throw new Error('你当前已在其他会议中')
    }
    // 与 createAndEnter 同理：消除上一次 ENDED 会议残留的 Store 状态
    _reset()
    localState.value = MEETING_LOCAL_STATE_JOINING
    try {
      // 首次调用时抑制失败 toast：失败后若是 stale 残留会自动清理 + 重试，对用户无感
      const resp = await meetingApi.joinRoom(roomCode, { password }, _retried ? {} : { silent: true })
      currentRoom.value = resp.room
      currentParticipant.value = resp.participant
      routerID.value = resp.router_id || ''
      await _afterJoined({ rtpCapabilities: resp.rtp_capabilities, mediaPrefs })
      return currentRoom.value
    } catch (err) {
      _log('error', '[Meeting] 加入会议失败', err)
      _reset()
      const enriched = _enrichStaleMeetingError(err)
      // 刷新/崩溃残留：后端仍记该用户为活跃成员 → 自动清理后重试一次
      if (enriched?.staleMeetingHint && !_retried) {
        _log('warn', '[Meeting] 检测到僵尸会议残留，自动清理并重试加入')
        try { await cleanupStaleMeetings() } catch (e) { _log('warn', '[Meeting] 自动清理失败', e) }
        return joinAndEnter(roomCode, password, mediaPrefs, true)
      }
      throw enriched
    }
  }

  /** 主动离会（关闭媒体 + REST leave） */
  const leave = async () => {
    if (!isInMeeting.value || !currentRoom.value) return
    localState.value = MEETING_LOCAL_STATE_LEAVING
    const code = currentRoom.value.room_code
    try {
      _cleanupMedia()
      _unregisterListeners()
      await meetingApi.leaveRoom(code)
    } catch (err) {
      _log('warn', '[Meeting] leave API 失败（忽略）', err)
    } finally {
      _reset()
    }
  }

  /**
   * 主持人结束会议（后端会广播 room.ended 给所有人）
   * Task 16 资源清理：API 成功后立即本地兜底 _cleanupMedia
   * - 避免"API 成功但 WS 断线导致 room.ended 广播丢失 → 本地 engine/timer/AudioContext 泄漏"
   * - 同时触发 _onRoomEnded 走 ENDED 状态，UI 展示"会议已结束"提示页
   * - 若之后广播到达也是幂等（_engine 已 null + closed 判定）
   */
  const endMeeting = async () => {
    if (!currentRoom.value) return
    const code = currentRoom.value.room_code
    try {
      await meetingApi.endRoom(code)
    } catch (err) {
      _log('warn', '[Meeting] end API 失败', err)
      throw err
    }
    _onRoomEnded({ data: { room_code: code, reason: 'host_ended' } })
  }

  /**
   * Task 16 资源清理：退出"会议已结束"提示页时的清理
   * - _onRoomEnded 刻意保留了 currentRoom / participants / chatMessages 供结束页渲染
   * - 但若用户长时间停留在结束页（或直接 reLaunch 去别处），pinia state 会一直占内存
   * - 调用方：room.vue 的 onUnload / 结束页"返回首页"按钮
   * - 仅在 ENDED 状态下执行 _reset；其他状态用 leave() 通道
   */
  const exitEndedRoom = () => {
    if (localState.value === MEETING_LOCAL_STATE_ENDED) {
      _reset()
    }
  }

  // ==================== Actions：本地推流 ====================

  /**
   * 开启本地音频
   * @param {string} [deviceId] - 指定麦克风 deviceId（从 enumerateDevices 获取），为空则走系统默认
   */
  const startLocalAudio = async (deviceId, existingTrack) => {
    if (!_engine) throw new Error('未连接')
    if (localProducers.audio) return
    // 优先复用预览页已申请的 track，避开 iOS Safari/Chrome 连续两次 getUserMedia
    // 触发 NotReadableError（预览页 track.stop() 后相机硬件有释放延迟）
    let track = existingTrack && existingTrack.readyState === 'live' ? existingTrack : null
    if (!track) {
      const audioConstraints = deviceId ? { deviceId: { exact: deviceId } } : true
      const stream = await navigator.mediaDevices.getUserMedia({ audio: audioConstraints })
      track = stream.getAudioTracks()[0]
    }
    const producer = await _engine.produce({ kind: 'audio', track })
    localProducers.audio = producer.id
    localAudioEnabled.value = true
    _broadcastSelfState({ audio_enabled: true })
  }

  /**
   * 开启本地视频
   * @param {string} [deviceId] - 指定摄像头 deviceId，为空则走系统默认
   */
  const startLocalVideo = async (deviceId, existingTrack) => {
    if (!_engine) throw new Error('未连接')
    if (localProducers.video) return
    // 优先复用预览页已申请的 track，避开 iOS Safari/Chrome 连续两次 getUserMedia
    // 触发 NotReadableError（预览页 track.stop() 后相机硬件有释放延迟）
    let track = existingTrack && existingTrack.readyState === 'live' ? existingTrack : null
    if (!track) {
      const videoConstraints = {
        width: { ideal: 1280 },
        height: { ideal: 720 },
        frameRate: { ideal: 24, max: 30 },
        ...(deviceId ? { deviceId: { exact: deviceId } } : {})
      }
      const stream = await navigator.mediaDevices.getUserMedia({ video: videoConstraints })
      track = stream.getVideoTracks()[0]
    }
    const producer = await _engine.produce({ kind: 'video', track })
    localProducers.video = producer.id
    localVideoEnabled.value = true
    _broadcastSelfState({ video_enabled: true })
  }

  /** 关闭本地音频 */
  const stopLocalAudio = async () => {
    if (!_engine || !localProducers.audio) return
    await _engine.closeProducer(localProducers.audio)
    localProducers.audio = null
    localAudioEnabled.value = false
    _broadcastSelfState({ audio_enabled: false })
  }

  /** 关闭本地视频 */
  const stopLocalVideo = async () => {
    if (!_engine || !localProducers.video) return
    await _engine.closeProducer(localProducers.video)
    localProducers.video = null
    localVideoEnabled.value = false
    _broadcastSelfState({ video_enabled: false })
  }

  /**
   * Task 15：向房间其他成员广播自身音视频状态
   * - 后端 meeting.member.state.changed 语义：操作自己无需 target_user_id
   * - 后端会广播给房间其他成员（不回显给本人）
   *
   * Task 16 P1-6：增加 sendWithAck 重试与 member_state Hash 最终一致性保障
   * - 旧版失败仅 warn，WS 抖动期其他成员的"图标灰色"问题会重现
   * - 现实现最多 2 次指数退避重试（700ms → 2100ms）；所有重试失败再输出 error 日志
   * - 设计约束：总耗时上限 ~6s，避免用户切换过快时累积旧状态覆盖新状态
   *   （每次调用前先记录 targetSnapshot，重试前若 currentRoom 已变则放弃；
   *    若 patch 同字段已被后续 broadcast 覆盖，也放弃旧 patch 的重试）
   */
  const _broadcastSelfState = (patch) => {
    if (!currentRoom.value) return
    if (!patch || typeof patch !== 'object') return
    const payload = { room_code: currentRoom.value.room_code }
    if (typeof patch.audio_enabled === 'boolean') payload.audio_enabled = patch.audio_enabled
    if (typeof patch.video_enabled === 'boolean') payload.video_enabled = patch.video_enabled
    if (Object.keys(payload).length <= 1) return

    const targetRoomCode = currentRoom.value.room_code
    const maxAttempts = 3
    const backoffMs = [0, 700, 2100]

    const attemptSend = (attempt) => {
      if (!currentRoom.value || currentRoom.value.room_code !== targetRoomCode) {
        _log('info', '[Meeting] 切换会议/已离会，放弃 state 重试', payload)
        return
      }
      if (typeof patch.audio_enabled === 'boolean' && localAudioEnabled.value !== patch.audio_enabled) {
        _log('info', '[Meeting] audio_enabled 已被后续动作覆盖，放弃旧 patch 重试', patch)
        return
      }
      if (typeof patch.video_enabled === 'boolean' && localVideoEnabled.value !== patch.video_enabled) {
        _log('info', '[Meeting] video_enabled 已被后续动作覆盖，放弃旧 patch 重试', patch)
        return
      }
      wsService.sendWithAck(MEETING_WS_MEMBER_STATE_CHANGED, payload, 3000)
        .catch((err) => {
          if (attempt + 1 >= maxAttempts) {
            _log('error', '[Meeting] 上报本地音视频状态失败（已用尽重试）', err, payload)
            return
          }
          const delay = backoffMs[attempt + 1] || 2100
          _log('warn', `[Meeting] 上报本地音视频状态失败，${delay}ms 后重试 (${attempt + 2}/${maxAttempts})`, err)
          // Task 16 资源清理：把 setTimeout 句柄登记到全局 Set
          // 离会/结束会议时 _cleanupMedia() 统一 clearTimeout，避免 stale 重试排队
          const handle = setTimeout(() => {
            _pendingBroadcastTimers.delete(handle)
            attemptSend(attempt + 1)
          }, delay)
          _pendingBroadcastTimers.add(handle)
        })
    }
    attemptSend(0)
  }

  /**
   * 取得某个远端用户的 track（供页面挂到 <video>/<audio>）
   * @param {number} userId
   * @param {'audio'|'video'} kind
   * @returns {MediaStreamTrack|null}
   */
  const getRemoteTrack = (userId, kind) => {
    const slot = remoteConsumers[userId]
    if (!slot || !slot[kind]) return null
    return slot[kind].track
  }

  /** 取得本地 Producer 对应的 track（预览用） */
  const getLocalTrack = (kind) => {
    if (!_engine) return null
    const producerId = localProducers[kind]
    if (!producerId) return null
    const producer = _engine.getProducer(producerId)
    return producer ? producer.track : null
  }

  // ==================== Actions：聊天/管理 ====================

  /**
   * 发送会议聊天
   * 后端广播 meeting.chat.message 给房间内非发送者，发送方依赖 REST 响应里的 message 上屏；
   * 若后端未把 message 字段回传，则退化为等待广播（不上屏）
   */
  const sendChat = async (content) => {
    if (!currentRoom.value) throw new Error('未入会')
    const resp = await meetingApi.sendChat(currentRoom.value.room_code, content)
    const msg = resp?.message
    if (msg && !chatMessages.value.some(m => m.id === msg.id)) {
      chatMessages.value.push(msg)
    }
    return msg
  }

  /** 加载更早的聊天消息，按 ID 游标向前翻页；首次进入时 beforeId=0 表示拿最新一页 */
  const loadMoreChats = async (limit = 30) => {
    if (!currentRoom.value) return { list: [], has_more: false }
    const oldest = chatMessages.value[0]
    const beforeId = oldest?.id || 0
    const resp = await meetingApi.listChats(currentRoom.value.room_code, { before_id: beforeId, limit })
    const list = (resp?.list || []).slice().sort((a, b) => (a.id || 0) - (b.id || 0))
    if (list.length) {
      // 去重后按时间序插入到头部
      const existing = new Set(chatMessages.value.map(m => m.id))
      const fresh = list.filter(m => !existing.has(m.id))
      if (fresh.length) chatMessages.value.unshift(...fresh)
    }
    chatHasMore.value = !!resp?.has_more
    chatHistoryLoaded.value = true
    return resp
  }

  /** 读掉所有会议聊天未读（ChatPanel 打开时调用） */
  const markChatsRead = () => {
    chatUnreadCount.value = 0
  }

  /** 兼容旧调用：保留 loadChatHistory 名称，内部调 loadMoreChats */
  const loadChatHistory = loadMoreChats

  const transferHost = async (targetUserID) => {
    if (!currentRoom.value) throw new Error('未入会')
    await meetingApi.transferHost(currentRoom.value.room_code, targetUserID)
  }

  const kickMember = async (userId) => {
    if (!currentRoom.value) throw new Error('未入会')
    await meetingApi.kickMember(currentRoom.value.room_code, userId)
  }

  /**
   * 主持人静音他人 / 请他开麦（Task 15 主持人权限四件套第 4 件）
   *
   * 通过 WS meeting.member.state.changed 携带 target_user_id，
   * 后端已实现 host 校验与广播逻辑（meeting_signal_service.go），前端仅需发送。
   *
   * @param {number} targetUserId 被操作对象 user_id
   * @param {boolean} mute true=静音，false=请他开麦
   */
  const muteMember = async (targetUserId, mute = true) => {
    if (!currentRoom.value) throw new Error('未入会')
    if (!isHost.value) throw new Error('仅主持人可操作')
    await wsService.sendWithAck(MEETING_WS_MEMBER_STATE_CHANGED, {
      room_code: currentRoom.value.room_code,
      target_user_id: targetUserId,
      audio_enabled: !mute
    }, 3000)
  }

  const inviteUsers = async (inviteeIds) => {
    if (!currentRoom.value) throw new Error('未入会')
    return meetingApi.inviteUsers(currentRoom.value.room_code, inviteeIds)
  }

  /**
   * 清理遗留的僵尸活跃会议
   *
   * 使用场景：前端崩溃 / 强制刷新 / 关闭标签页未调 `/leave` 时，后端
   * `meeting_participants` 表仍标记用户为活跃参会者，再次创建/加入会议会
   * 被后端以"你当前已在其他会议中"拒绝。本方法拉取用户全部 `status=Active`
   * 会议并依次调用 `/leave`，顺带处理可能多条的僵尸记录（比如 DAO 异常）。
   *
   * @returns {Promise<{ cleanedCodes: string[] }>}
   */
  const cleanupStaleMeetings = async () => {
    const resp = await meetingApi.listMyMeetings({ status: 1, limit: 50 })
    const list = (resp && resp.list) || []
    const cleanedCodes = []
    for (const room of list) {
      try {
        // silent: true —— 清理流程中 leave 失败（如 "你不在此会议中"）
        // 属于幂等预期错误，不应弹 toast 打扰用户
        await meetingApi.leaveRoom(room.room_code, { silent: true })
        cleanedCodes.push(room.room_code)
      } catch (err) {
        _log('warn', '[Meeting] 清理遗留会议失败（忽略继续）', room.room_code, err)
      }
    }
    if (isInMeeting.value) {
      // 本地状态可能因 leave 网络错误等处于脏状态，一并 reset
      _reset()
    }
    return { cleanedCodes }
  }

  // ==================== 导出 ====================

  return {
    // state
    localState,
    currentRoom,
    currentParticipant,
    routerID,
    participants,
    chatMessages,
    chatHasMore,
    chatUnreadCount,
    chatHistoryLoaded,
    lastEndedReason,
    lastAutoHostReason,
    localAudioEnabled,
    localVideoEnabled,
    localProducers,
    remoteConsumers,
    draftCreatePayload,
    draftJoinPayload,
    devicePreview,
    speakingMap,
    uiPrefs,

    // getters
    isInMeeting,
    isConnected,
    isHost,
    activeParticipants,
    isAllMuted,

    // lifecycle
    createAndEnter,
    joinAndEnter,
    leave,
    endMeeting,
    exitEndedRoom,

    // media
    startLocalAudio,
    startLocalVideo,
    stopLocalAudio,
    stopLocalVideo,
    getRemoteTrack,
    getLocalTrack,

    // chat & admin
    sendChat,
    loadMoreChats,
    loadChatHistory,
    markChatsRead,
    transferHost,
    kickMember,
    muteMember,
    inviteUsers,
    cleanupStaleMeetings,

    // util
    isMediaSupported
  }
})
