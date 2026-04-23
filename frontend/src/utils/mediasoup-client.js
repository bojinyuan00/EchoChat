/**
 * mediasoup-client 封装层
 *
 * 职责：
 * - 封装 Device/sendTransport/recvTransport/Producer/Consumer 的生命周期
 * - 把 Transport 的 connect/produce 回调桥接到 WS 信令（sendWithAck）
 * - 对上层（store/meeting.js）暴露同步友好的异步 API
 *
 * 平台约束（Task 9 决策 Q1=a1_h5_only）：
 * - mediasoup-client 仅在 H5 端可用；非 H5 平台调用时会在构造时抛 ERR_PLATFORM
 * - import 语句通过 uni-app 条件编译注释限定在 H5 构建
 *
 * 使用方式（仅限 store/meeting.js）：
 *   import { createMediaEngine } from '@/utils/mediasoup-client'
 *   const engine = markRaw(createMediaEngine({ roomCode, userId, sendWithAck }))
 *   await engine.loadDevice(rtpCapabilities)
 *   const sendTransport = await engine.ensureSendTransport()
 *   const producer = await engine.produce({ kind: 'audio', track })
 */

// #ifdef H5
import { Device } from 'mediasoup-client'
// #endif

import {
  MEETING_WS_TRANSPORT_CREATE,
  MEETING_WS_TRANSPORT_CONNECT,
  MEETING_WS_PRODUCE_START,
  MEETING_WS_CONSUME_START,
  MEETING_WS_CONSUME_RESUME,
  MEETING_WS_PRODUCER_CLOSE
} from '@/constants/meeting'

/**
 * 创建 MediaEngine 实例
 *
 * @param {Object} options
 * @param {string} options.roomCode - 会议号
 * @param {number} options.userId - 当前用户 ID
 * @param {Function} options.sendWithAck - WS 发送带 ACK 等待的方法（由 services/websocket.js 提供）
 * @param {Function} [options.logger] - 日志函数，默认 console.log
 * @returns {Object} MediaEngine 实例
 */
export function createMediaEngine({ roomCode, userId, sendWithAck, logger = defaultLogger }) {
  if (!roomCode) throw new Error('[MediaEngine] roomCode 不能为空')
  if (!userId) throw new Error('[MediaEngine] userId 不能为空')
  if (typeof sendWithAck !== 'function') {
    throw new Error('[MediaEngine] sendWithAck 必须是函数')
  }

  // #ifndef H5
  throw new Error('[MediaEngine] 当前平台不支持 mediasoup-client（仅支持 H5）')
  // #endif

  // #ifdef H5
  /** mediasoup Device 实例；Device.load 后持有 Router rtpCapabilities */
  let device = null

  /** 发送 Transport（本地 Producer 用），延迟创建，复用 */
  let sendTransport = null

  /** 接收 Transport（本地 Consumer 用），延迟创建，复用 */
  let recvTransport = null

  /** 本地 Producer 索引 Map<producerId, Producer> */
  const producers = new Map()

  /** 本地 Consumer 索引 Map<consumerId, Consumer> */
  const consumers = new Map()

  /** 是否已关闭，防止重复调用 close */
  let closed = false

  const ensureNotClosed = () => {
    if (closed) {
      throw new Error('[MediaEngine] 已关闭')
    }
  }

  const ensureDeviceLoaded = () => {
    ensureNotClosed()
    if (!device || !device.loaded) {
      throw new Error('[MediaEngine] Device 尚未 load')
    }
  }

  /**
   * 加载 Device
   * @param {Object} routerRtpCapabilities - 后端 JoinRoom 响应中的 rtp_capabilities
   */
  const loadDevice = async (routerRtpCapabilities) => {
    ensureNotClosed()
    if (device && device.loaded) {
      logger('debug', '[MediaEngine] Device 已 load，跳过')
      return
    }
    if (!routerRtpCapabilities) {
      throw new Error('[MediaEngine] routerRtpCapabilities 不能为空')
    }
    device = new Device()
    await device.load({ routerRtpCapabilities })
    logger('info', '[MediaEngine] Device load 成功', { canProduceAudio: device.canProduce('audio'), canProduceVideo: device.canProduce('video') })
  }

  /** 返回 Device.rtpCapabilities（用于后端 CreateConsumer） */
  const getRtpCapabilities = () => {
    ensureDeviceLoaded()
    return device.rtpCapabilities
  }

  /**
   * 通过 WS 请求后端创建 Transport，返回 mediasoup-client 可用的 Transport 元信息
   * @param {'send'|'recv'} direction
   */
  const requestTransportInfo = async (direction) => {
    const info = await sendWithAck(MEETING_WS_TRANSPORT_CREATE, {
      room_code: roomCode,
      direction
    })
    if (!info || !info.id) {
      throw new Error(`[MediaEngine] 创建 ${direction} Transport 失败：后端返回为空`)
    }
    return info
  }

  /**
   * 绑定 sendTransport 的 connect + produce 回调（把本地事件桥到 WS 信令）
   * mediasoup-client 约定：
   *   - transport.on('connect', ({ dtlsParameters }, callback, errback))
   *     需要在 DTLS 握手完成前告知远端 DTLS 参数
   *   - transport.on('produce', ({ kind, rtpParameters, appData }, callback, errback))
   *     需要在 Producer 创建前把 RTP 参数发给远端，拿到远端分配的 producerId 回调 callback({ id })
   */
  const bindSendTransportEvents = (transport) => {
    transport.on('connect', ({ dtlsParameters }, callback, errback) => {
      sendWithAck(MEETING_WS_TRANSPORT_CONNECT, {
        room_code: roomCode,
        transport_id: transport.id,
        dtls_parameters: dtlsParameters
      }).then(() => callback()).catch((err) => {
        logger('error', '[MediaEngine] sendTransport connect 失败', err)
        errback(err)
      })
    })

    transport.on('produce', ({ kind, rtpParameters, appData }, callback, errback) => {
      sendWithAck(MEETING_WS_PRODUCE_START, {
        room_code: roomCode,
        transport_id: transport.id,
        kind,
        rtp_parameters: rtpParameters,
        app_data: appData || {}
      }).then((resp) => {
        if (!resp || !resp.producer_id) {
          errback(new Error('后端返回 producer_id 为空'))
          return
        }
        callback({ id: resp.producer_id })
      }).catch((err) => {
        logger('error', '[MediaEngine] sendTransport produce 失败', err)
        errback(err)
      })
    })

    transport.on('connectionstatechange', (state) => {
      logger('debug', `[MediaEngine] sendTransport state=${state}`)
    })
  }

  /** recvTransport 只需要桥接 connect 回调（consume 由 store 主动调起） */
  const bindRecvTransportEvents = (transport) => {
    transport.on('connect', ({ dtlsParameters }, callback, errback) => {
      sendWithAck(MEETING_WS_TRANSPORT_CONNECT, {
        room_code: roomCode,
        transport_id: transport.id,
        dtls_parameters: dtlsParameters
      }).then(() => callback()).catch((err) => {
        logger('error', '[MediaEngine] recvTransport connect 失败', err)
        errback(err)
      })
    })
    transport.on('connectionstatechange', (state) => {
      logger('debug', `[MediaEngine] recvTransport state=${state}`)
    })
  }

  /** in-flight Promise：sendTransport 创建并发锁；避免突发并发下重复创建 */
  let sendTransportPromise = null
  /** in-flight Promise：recvTransport 创建并发锁；Task 15 后入者 burst 订阅时尤其关键 */
  let recvTransportPromise = null

  /** 按需创建 sendTransport（已存在则复用，并发调用共享同一个 in-flight Promise） */
  const ensureSendTransport = async () => {
    ensureDeviceLoaded()
    if (sendTransport && !sendTransport.closed) {
      return sendTransport
    }
    if (sendTransportPromise) {
      return sendTransportPromise
    }
    sendTransportPromise = (async () => {
      try {
        return await _createSendTransport()
      } finally {
        sendTransportPromise = null
      }
    })()
    return sendTransportPromise
  }

  /** 首次创建 sendTransport 的底层流程，原本 inline 在 ensureSendTransport 中 */
  const _createSendTransport = async () => {
    const info = await requestTransportInfo('send')
    sendTransport = device.createSendTransport({
      id: info.id,
      iceParameters: info.iceParameters,
      iceCandidates: info.iceCandidates,
      dtlsParameters: info.dtlsParameters,
      sctpParameters: info.sctpParameters
    })
    bindSendTransportEvents(sendTransport)
    logger('info', '[MediaEngine] sendTransport 创建成功', { id: sendTransport.id })
    return sendTransport
  }

  /** 按需创建 recvTransport（已存在则复用，并发调用共享同一个 in-flight Promise） */
  const ensureRecvTransport = async () => {
    ensureDeviceLoaded()
    if (recvTransport && !recvTransport.closed) {
      return recvTransport
    }
    if (recvTransportPromise) {
      return recvTransportPromise
    }
    recvTransportPromise = (async () => {
      try {
        return await _createRecvTransport()
      } finally {
        recvTransportPromise = null
      }
    })()
    return recvTransportPromise
  }

  /** 首次创建 recvTransport 的底层流程 */
  const _createRecvTransport = async () => {
    const info = await requestTransportInfo('recv')
    recvTransport = device.createRecvTransport({
      id: info.id,
      iceParameters: info.iceParameters,
      iceCandidates: info.iceCandidates,
      dtlsParameters: info.dtlsParameters,
      sctpParameters: info.sctpParameters
    })
    bindRecvTransportEvents(recvTransport)
    logger('info', '[MediaEngine] recvTransport 创建成功', { id: recvTransport.id })
    return recvTransport
  }

  /**
   * 在 sendTransport 上创建 Producer（推本地音/视频）
   * @param {Object} opts
   * @param {'audio'|'video'} opts.kind
   * @param {MediaStreamTrack} opts.track
   * @param {Object} [opts.encodings]
   * @param {Object} [opts.codecOptions]
   * @param {Object} [opts.appData]
   * @returns {Promise<Producer>}
   */
  const produce = async ({ kind, track, encodings, codecOptions, appData }) => {
    ensureDeviceLoaded()
    if (!device.canProduce(kind)) {
      throw new Error(`[MediaEngine] Device 不支持 produce kind=${kind}`)
    }
    const transport = await ensureSendTransport()
    const produceOpts = { track }
    if (encodings) produceOpts.encodings = encodings
    if (codecOptions) produceOpts.codecOptions = codecOptions
    produceOpts.appData = { user_id: userId, ...(appData || {}) }

    const producer = await transport.produce(produceOpts)
    producers.set(producer.id, producer)

    // 监听 Producer 关闭事件，清理本地索引；外部可见的关闭由 closeProducer 触发 WS
    producer.on('transportclose', () => {
      producers.delete(producer.id)
      logger('debug', `[MediaEngine] producer ${producer.id} transportclose`)
    })
    producer.on('trackended', () => {
      logger('warn', `[MediaEngine] producer ${producer.id} trackended，将触发关闭`)
      closeProducer(producer.id).catch((err) => logger('error', '关闭 Producer 失败', err))
    })

    logger('info', `[MediaEngine] producer 创建成功 kind=${kind} id=${producer.id}`)
    return producer
  }

  /**
   * 订阅远端 Producer：请求后端创建 Consumer → 本地 consume → 等 track 挂好后 resume
   *
   * 与 Task 9 决策 Q6=B 对齐的规范流程：
   *   1. WS consume.start → 拿到 { id, producerId, kind, rtpParameters } （Node 侧 paused）
   *   2. recvTransport.consume(...) 得到本地 Consumer（track 可用）
   *   3. 调用方把 track 挂到 <video>/<audio> 元素（DOM 就绪）
   *   4. 调用 consumer.resume()（此方法返回的 consumer 暴露 resume()）
   *
   * 本函数执行完 1+2 后返回 consumer，调用方挂 track 后必须调一次 engine.resumeConsumer(consumerId)
   * 以告知后端 → Node 把 Consumer 从 paused 切到 active
   *
   * @param {Object} opts
   * @param {string} opts.producerId - 要订阅的远端 Producer ID
   * @returns {Promise<Consumer>} 本地 Consumer 实例（此时仍 paused）
   */
  const consume = async ({ producerId }) => {
    ensureDeviceLoaded()
    if (!producerId) throw new Error('[MediaEngine] producerId 不能为空')

    const transport = await ensureRecvTransport()
    const info = await sendWithAck(MEETING_WS_CONSUME_START, {
      room_code: roomCode,
      transport_id: transport.id,
      producer_id: producerId,
      rtp_capabilities: device.rtpCapabilities
    })
    if (!info || !info.id) {
      throw new Error('[MediaEngine] 后端未返回 Consumer 元信息')
    }

    const consumer = await transport.consume({
      id: info.id,
      producerId: info.producerId || info.producer_id || producerId,
      kind: info.kind,
      rtpParameters: info.rtpParameters
    })
    consumers.set(consumer.id, consumer)

    consumer.on('transportclose', () => {
      consumers.delete(consumer.id)
      logger('debug', `[MediaEngine] consumer ${consumer.id} transportclose`)
    })

    logger('info', `[MediaEngine] consumer 创建成功 id=${consumer.id} kind=${consumer.kind}`)
    return consumer
  }

  /**
   * 通知后端 resume Consumer（track 已挂载到 DOM 后调用）
   * 对应 Task 9 决策 Q6=B 的 meeting.consume.resume WS 事件
   * @param {string} consumerId
   */
  const resumeConsumer = async (consumerId) => {
    ensureNotClosed()
    await sendWithAck(MEETING_WS_CONSUME_RESUME, {
      room_code: roomCode,
      consumer_id: consumerId
    })
    const local = consumers.get(consumerId)
    if (local && typeof local.resume === 'function') {
      await local.resume()
    }
    logger('info', `[MediaEngine] consumer ${consumerId} resumed`)
  }

  /**
   * 关闭指定 Producer（本地 close + WS 通知后端）
   * 幂等：producerId 不存在时静默返回
   */
  const closeProducer = async (producerId) => {
    if (closed) return
    const producer = producers.get(producerId)
    if (!producer) return
    try {
      if (!producer.closed) producer.close()
    } catch (e) {
      logger('warn', '[MediaEngine] producer.close 抛错', e)
    }
    producers.delete(producerId)
    try {
      await sendWithAck(MEETING_WS_PRODUCER_CLOSE, {
        room_code: roomCode,
        producer_id: producerId
      })
    } catch (e) {
      // 后端已清理也视为成功（幂等）；只记日志
      logger('warn', `[MediaEngine] producer.close WS 通知失败 ${producerId}`, e)
    }
  }

  /** 本地 Consumer 关闭（通常由 producer.new closed=true 广播触发，无需额外 WS） */
  const closeConsumer = (consumerId) => {
    const consumer = consumers.get(consumerId)
    if (!consumer) return
    try {
      if (!consumer.closed) consumer.close()
    } catch (e) {
      logger('warn', '[MediaEngine] consumer.close 抛错', e)
    }
    consumers.delete(consumerId)
  }

  /** 释放所有资源（本地 close，不触发 WS；由 store 在离会时调用） */
  const close = () => {
    if (closed) return
    closed = true
    try {
      producers.forEach((p) => { try { if (!p.closed) p.close() } catch {} })
      producers.clear()
      consumers.forEach((c) => { try { if (!c.closed) c.close() } catch {} })
      consumers.clear()
      if (sendTransport && !sendTransport.closed) sendTransport.close()
      if (recvTransport && !recvTransport.closed) recvTransport.close()
    } finally {
      sendTransport = null
      recvTransport = null
      device = null
    }
    logger('info', '[MediaEngine] 已关闭')
  }

  return {
    loadDevice,
    getRtpCapabilities,
    ensureSendTransport,
    ensureRecvTransport,
    produce,
    consume,
    resumeConsumer,
    closeProducer,
    closeConsumer,
    close,
    getDevice: () => device,
    getSendTransport: () => sendTransport,
    getRecvTransport: () => recvTransport,
    getProducer: (id) => producers.get(id),
    getConsumer: (id) => consumers.get(id)
  }
  // #endif
}

function defaultLogger(level, ...args) {
  const fn = console[level] || console.log
  fn.call(console, ...args)
}
