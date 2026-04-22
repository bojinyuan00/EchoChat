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
    localState.value = MEETING_LOCAL_STATE_ENDED
    _unregisterListeners()
  }

  const _onMemberJoined = (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    // 后端广播目前只带基础字段，完整 DTO 通过 GET /rooms/:code 二次拉取
    if (!participants.value.find(p => p.user_id === data.user_id)) {
      participants.value.push({
        user_id: data.user_id,
        joined_at: data.joined_at,
        is_active: true,
        role: 0
      })
    }
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
    const p = participants.value.find(p => p.user_id === data.user_id)
    if (p) {
      if (typeof data.audio_enabled === 'boolean') p.audio_enabled = data.audio_enabled
      if (typeof data.video_enabled === 'boolean') p.video_enabled = data.video_enabled
    }
  }

  const _onMemberKicked = (msg) => {
    const data = msg.data || {}
    if (!currentRoom.value || data.room_code !== currentRoom.value.room_code) return
    const userStore = useUserStore()
    if (data.user_id === userStore.userInfo?.id) {
      _log('warn', '[Meeting] 当前用户被踢出会议')
      lastEndedReason.value = 'kicked'
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
          participants.value = detail.participants
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

  const _cleanupRemoteProducer = (userId, producerId) => {
    const slot = remoteConsumers[userId]
    if (!slot) return
    slot.producerIds.delete(producerId)
    // 无法从 producerId 直接反查 consumer；简单策略：关 slot 内所有 consumer 然后重建（MVP）
    // 但更稳的：把 Consumer 索引按 producerId 存一遍
    ;['audio', 'video'].forEach(kind => {
      const consumer = slot[kind]
      if (consumer && !consumer.closed) {
        try { consumer.close() } catch {}
        slot[kind] = null
      }
    })
    if (slot.producerIds.size === 0) {
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
    if (_engine) {
      try { _engine.close() } catch (e) { _log('warn', '_cleanupMedia engine.close', e) }
      _engine = null
    }
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

    // 告知后端 WS 我已绑定本房间（用于 WS 连接 ↔ roomCode 映射）
    await wsService.sendWithAck(MEETING_WS_ROOM_JOIN, {
      room_code: currentRoom.value.room_code
    })

    // 按需先创建 sendTransport + recvTransport，避免页面点"开麦"时首帧有额外抖动
    await _engine.ensureSendTransport()
    await _engine.ensureRecvTransport()

    // 拉取最新参与者列表（广播里只有 ID，这里补齐完整 DTO）
    try {
      const detail = await meetingApi.getRoom(currentRoom.value.room_code)
      if (detail && Array.isArray(detail.participants)) {
        participants.value = detail.participants
      }
    } catch (e) {
      _log('warn', '[Meeting] 拉取参与者详情失败，保留广播缓存', e)
    }

    localState.value = MEETING_LOCAL_STATE_CONNECTED

    // mediaPrefs 自动推流：失败仅告警，不阻断入会流程，允许用户在会议室内手动重试开麦/开摄像头
    if (mediaPrefs) {
      if (mediaPrefs.startAudio) {
        try { await startLocalAudio(mediaPrefs.audioDeviceId) }
        catch (e) { _log('warn', '[Meeting] 自动开麦失败', e) }
      }
      if (mediaPrefs.startVideo) {
        try { await startLocalVideo(mediaPrefs.videoDeviceId) }
        catch (e) { _log('warn', '[Meeting] 自动开摄像头失败', e) }
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
    localState.value = MEETING_LOCAL_STATE_JOINING
    try {
      const resp = await meetingApi.createRoom(payload)
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
    localState.value = MEETING_LOCAL_STATE_JOINING
    try {
      const resp = await meetingApi.joinRoom(roomCode, { password })
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

  /** 主持人结束会议（会广播 room.ended 给所有人） */
  const endMeeting = async () => {
    if (!currentRoom.value) return
    const code = currentRoom.value.room_code
    try {
      await meetingApi.endRoom(code)
    } catch (err) {
      _log('warn', '[Meeting] end API 失败', err)
      throw err
    }
    // room.ended WS 广播会触发 _onRoomEnded 清理
  }

  // ==================== Actions：本地推流 ====================

  /**
   * 开启本地音频
   * @param {string} [deviceId] - 指定麦克风 deviceId（从 enumerateDevices 获取），为空则走系统默认
   */
  const startLocalAudio = async (deviceId) => {
    if (!_engine) throw new Error('未连接')
    if (localProducers.audio) return
    const audioConstraints = deviceId ? { deviceId: { exact: deviceId } } : true
    const stream = await navigator.mediaDevices.getUserMedia({ audio: audioConstraints })
    const track = stream.getAudioTracks()[0]
    const producer = await _engine.produce({ kind: 'audio', track })
    localProducers.audio = producer.id
    localAudioEnabled.value = true
  }

  /**
   * 开启本地视频
   * @param {string} [deviceId] - 指定摄像头 deviceId，为空则走系统默认
   */
  const startLocalVideo = async (deviceId) => {
    if (!_engine) throw new Error('未连接')
    if (localProducers.video) return
    const videoConstraints = {
      width: 640,
      height: 480,
      ...(deviceId ? { deviceId: { exact: deviceId } } : {})
    }
    const stream = await navigator.mediaDevices.getUserMedia({ video: videoConstraints })
    const track = stream.getVideoTracks()[0]
    const producer = await _engine.produce({ kind: 'video', track })
    localProducers.video = producer.id
    localVideoEnabled.value = true
  }

  /** 关闭本地音频 */
  const stopLocalAudio = async () => {
    if (!_engine || !localProducers.audio) return
    await _engine.closeProducer(localProducers.audio)
    localProducers.audio = null
    localAudioEnabled.value = false
  }

  /** 关闭本地视频 */
  const stopLocalVideo = async () => {
    if (!_engine || !localProducers.video) return
    await _engine.closeProducer(localProducers.video)
    localProducers.video = null
    localVideoEnabled.value = false
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
        await meetingApi.leaveRoom(room.room_code)
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
    devicePreview,

    // getters
    isInMeeting,
    isConnected,
    isHost,
    activeParticipants,

    // lifecycle
    createAndEnter,
    joinAndEnter,
    leave,
    endMeeting,

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
    inviteUsers,
    cleanupStaleMeetings,

    // util
    isMediaSupported
  }
})
