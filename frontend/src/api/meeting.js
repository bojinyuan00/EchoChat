/**
 * 会议模块 REST API
 *
 * 对应后端路由：/api/v1/meeting/*（见 backend/go-service/app/meeting/router.go）
 *
 * 设计原则：
 * - 本文件仅做 REST 封装，业务层（store/meeting.js）统一调本文件方法
 * - WebRTC 媒体层信令通过 WS（services/websocket.js + sendWithAck）发送，不在此处
 * - 响应中的 DTO 字段命名与后端 DTO 完全一致（下划线），前端按需再转换
 */

import { get, post } from '@/utils/request'

/**
 * 统一拆包：utils/request.js 返回完整 envelope { code, message, data, trace_id, time }，
 * 业务层只关心 data 负载；这里一次性拆包，避免每个 store/页面都写 .data.xxx
 * 对无 body 的响应（如 leave/end）返回 null 而非 undefined，方便调用方判空
 */
const unwrap = (promise) => promise.then((resp) => (resp && 'data' in resp ? (resp.data ?? null) : (resp ?? null)))

// ==================== 会议房间（6 接口） ====================

/**
 * 创建即时会议
 * @param {Object} payload - { title, password?, max_members? }
 * @returns {Promise<{ room: MeetingRoomDTO }>}
 */
const createRoom = (payload) => {
  return unwrap(post('/api/v1/meeting/rooms', payload))
}

/**
 * 查询会议详情（含参与者列表）
 * @param {string} roomCode - 会议号（XXX-XXX-XXX 或 9 位纯数字）
 * @returns {Promise<{ room, participants, online_count }>}
 */
const getRoom = (roomCode) => {
  return unwrap(get(`/api/v1/meeting/rooms/${roomCode}`))
}

/**
 * 加入会议
 * @param {string} roomCode - 会议号
 * @param {Object} [payload] - { password? }
 * @returns {Promise<{ room, participant, router_id }>}
 */
const joinRoom = (roomCode, payload = {}) => {
  return unwrap(post(`/api/v1/meeting/rooms/${roomCode}/join`, payload))
}

/**
 * 离开会议
 * @param {string} roomCode
 * @returns {Promise<{ duration }>}
 */
const leaveRoom = (roomCode) => {
  return unwrap(post(`/api/v1/meeting/rooms/${roomCode}/leave`))
}

/**
 * 结束会议（仅主持人可调）
 * @param {string} roomCode
 * @returns {Promise<void>}
 */
const endRoom = (roomCode) => {
  return unwrap(post(`/api/v1/meeting/rooms/${roomCode}/end`))
}

/**
 * 我的会议列表（历史 + 进行中）
 * @param {Object} [params] - { status?, before_id?, limit? }
 * @returns {Promise<{ list, has_more }>}
 */
const listMyMeetings = (params = {}) => {
  return unwrap(get('/api/v1/meeting/rooms/mine', params))
}

// ==================== 会议成员管理（3 接口） ====================

/**
 * 转让主持人
 * @param {string} roomCode
 * @param {number} targetUserID
 * @returns {Promise<void>}
 */
const transferHost = (roomCode, targetUserID) => {
  return unwrap(post(`/api/v1/meeting/rooms/${roomCode}/transfer-host`, {
    target_user_id: targetUserID
  }))
}

/**
 * 移除成员（踢人，仅主持人）
 * @param {string} roomCode
 * @param {number} userID
 * @returns {Promise<void>}
 */
const kickMember = (roomCode, userID) => {
  return unwrap(post(`/api/v1/meeting/rooms/${roomCode}/kick`, { user_id: userID }))
}

/**
 * 邀请用户（批量）
 * @param {string} roomCode
 * @param {number[]} inviteeIDs - 被邀请用户 ID 数组（1~50）
 * @returns {Promise<{ pushed, skipped }>}
 */
const inviteUsers = (roomCode, inviteeIDs) => {
  return unwrap(post(`/api/v1/meeting/rooms/${roomCode}/invite`, {
    invitee_ids: inviteeIDs
  }))
}

// ==================== 邀请兑换（1 接口） ====================

/**
 * 兑换邀请 Token（从通知点进链接时使用）
 * @param {string} token - 邀请链接中的 token 字符串
 * @returns {Promise<{ room_code, inviter_id, has_password }>}
 */
const redeemInvite = (token) => {
  return unwrap(post(`/api/v1/meeting/invite-tokens/${token}/redeem`))
}

// ==================== 会议内聊天（2 接口） ====================

/**
 * 发送会议内聊天消息
 * @param {string} roomCode
 * @param {string} content - 1~500 字
 * @returns {Promise<{ message }>}
 */
const sendChat = (roomCode, content) => {
  return unwrap(post(`/api/v1/meeting/rooms/${roomCode}/chats`, { content }))
}

/**
 * 查询会议聊天历史（游标分页）
 * @param {string} roomCode
 * @param {Object} [params] - { before_id?, limit? }
 * @returns {Promise<{ list, has_more }>}
 */
const listChats = (roomCode, params = {}) => {
  return unwrap(get(`/api/v1/meeting/rooms/${roomCode}/chats`, params))
}

export default {
  createRoom,
  getRoom,
  joinRoom,
  leaveRoom,
  endRoom,
  listMyMeetings,
  transferHost,
  kickMember,
  inviteUsers,
  redeemInvite,
  sendChat,
  listChats
}
