/**
 * 会议模块前端常量
 *
 * 与后端 backend/go-service/app/constants/meeting.go 严格一致
 * 新增/修改任一常量时必须同步维护两端，否则运行时会因事件名不匹配而静默失败
 *
 * 版本说明：
 * - Task 6（2026-04-20）落地 13 个 WS 事件
 * - Task 9（2026-04-22）新增 meeting.consume.resume，总计 14 个事件
 */

// ==================== 会议类型 ====================

export const MEETING_TYPE_INSTANT = 1    // 即时会议（MVP 仅此）
export const MEETING_TYPE_SCHEDULED = 2  // 预约会议（Phase 2e-3）

export const MEETING_TYPE_LABEL = {
  [MEETING_TYPE_INSTANT]: '即时会议',
  [MEETING_TYPE_SCHEDULED]: '预约会议'
}

// ==================== 会议状态 ====================

export const MEETING_STATUS_PENDING = 0  // 未开始（仅预约会议）
export const MEETING_STATUS_ACTIVE = 1   // 进行中
export const MEETING_STATUS_ENDED = 2    // 已结束

export const MEETING_STATUS_LABEL = {
  [MEETING_STATUS_PENDING]: '未开始',
  [MEETING_STATUS_ACTIVE]: '进行中',
  [MEETING_STATUS_ENDED]: '已结束'
}

// ==================== 参与者角色 ====================

export const MEETING_ROLE_PARTICIPANT = 0  // 普通参会人
export const MEETING_ROLE_HOST = 1         // 主持人
export const MEETING_ROLE_COHOST = 2       // 联合主持人（保留，MVP 不用）

export const MEETING_ROLE_LABEL = {
  [MEETING_ROLE_PARTICIPANT]: '参会人',
  [MEETING_ROLE_HOST]: '主持人',
  [MEETING_ROLE_COHOST]: '联合主持人'
}

// ==================== 会议结束原因 ====================

// 注意：下列 4 个值必须与后端 backend/go-service/app/constants/meeting.go::MeetingEndedReason* 保持一致
export const MEETING_ENDED_REASON_HOST_ENDED = 'host_ended'
export const MEETING_ENDED_REASON_EMPTY_TTL = 'empty_ttl'
export const MEETING_ENDED_REASON_ADMIN_FORCE = 'admin_force'
export const MEETING_ENDED_REASON_SYSTEM_ERROR = 'system_error'
// 下列为"前端专用"原因（非后端 meeting_rooms.ended_reason 值），
// 用于 store 将当前用户的"单端终止"归因到 ENDED 本地状态时展示
export const MEETING_ENDED_REASON_KICKED = 'kicked'             // 当前用户被主持人移除（由 member.kicked 触发）

export const MEETING_ENDED_REASON_LABEL = {
  [MEETING_ENDED_REASON_HOST_ENDED]: '主持人结束',
  [MEETING_ENDED_REASON_EMPTY_TTL]: '空房超时',
  [MEETING_ENDED_REASON_ADMIN_FORCE]: '管理员强制结束',
  [MEETING_ENDED_REASON_SYSTEM_ERROR]: '系统异常',
  [MEETING_ENDED_REASON_KICKED]: '您已被主持人移出会议'
}

// ==================== 离会原因 ====================

export const MEETING_LEFT_REASON_SELF = 'self'
export const MEETING_LEFT_REASON_KICKED = 'kicked'
export const MEETING_LEFT_REASON_HOST_END = 'host_end'
export const MEETING_LEFT_REASON_EMPTY_TTL = 'empty_ttl'
export const MEETING_LEFT_REASON_DISCONNECT = 'disconnect'

export const MEETING_LEFT_REASON_LABEL = {
  [MEETING_LEFT_REASON_SELF]: '主动离开',
  [MEETING_LEFT_REASON_KICKED]: '被移除',
  [MEETING_LEFT_REASON_HOST_END]: '会议结束',
  [MEETING_LEFT_REASON_EMPTY_TTL]: '空房超时',
  [MEETING_LEFT_REASON_DISCONNECT]: '网络断开'
}

// ==================== 主持人变更自动转让原因（Task 8 引入） ====================

export const MEETING_HOST_AUTO_REASON_GRACE_EXPIRED = 'host_grace_expired'   // 主持人宽限期到期自动转让
export const MEETING_HOST_AUTO_REASON_HOST_KICKED = 'host_kicked'            // （保留）主持人被管理员踢出自动转让

export const MEETING_HOST_AUTO_REASON_LABEL = {
  [MEETING_HOST_AUTO_REASON_GRACE_EXPIRED]: '主持人掉线超时自动转让',
  [MEETING_HOST_AUTO_REASON_HOST_KICKED]: '主持人被移除自动转让'
}

// ==================== 默认配置常量 ====================

export const MEETING_MVP_MAX_MEMBERS = 8             // MVP 单会议硬上限
export const MEETING_HOST_GRACE_SECONDS = 120        // 主持人掉线宽限期（Task 8）
export const MEETING_EMPTY_ROOM_TTL_SECONDS = 300    // 空房销毁 TTL（Task 8）
export const MEETING_WS_ACK_TIMEOUT_MS = 10000       // WS 事件 ACK 等待超时（Task 9）

// ==================== WS 事件名（14 个，与后端 constants/meeting.go 一一对应） ====================

// 房间组（3 个）
export const MEETING_WS_ROOM_JOIN = 'meeting.room.join'     // C→S
export const MEETING_WS_ROOM_LEAVE = 'meeting.room.leave'   // C→S
export const MEETING_WS_ROOM_ENDED = 'meeting.room.ended'   // S→C

// 成员组（6 个）
export const MEETING_WS_MEMBER_JOINED = 'meeting.member.joined'              // S→C
export const MEETING_WS_MEMBER_LEFT = 'meeting.member.left'                  // S→C
export const MEETING_WS_MEMBER_STATE_CHANGED = 'meeting.member.state.changed' // 双向
export const MEETING_WS_MEMBER_KICKED = 'meeting.member.kicked'              // S→C（定向）
export const MEETING_WS_MEMBER_PRODUCER_NEW = 'meeting.member.producer.new'  // S→C
export const MEETING_WS_HOST_CHANGED = 'meeting.host.changed'                // S→C

// 媒体组（6 个，含 Task 9 新增 consume.resume）
export const MEETING_WS_TRANSPORT_CREATE = 'meeting.transport.create'   // C→S
export const MEETING_WS_TRANSPORT_CONNECT = 'meeting.transport.connect' // C→S
export const MEETING_WS_PRODUCE_START = 'meeting.produce.start'         // C→S
export const MEETING_WS_CONSUME_START = 'meeting.consume.start'         // C→S
export const MEETING_WS_CONSUME_RESUME = 'meeting.consume.resume'       // C→S（Task 9 新增）
export const MEETING_WS_PRODUCER_CLOSE = 'meeting.producer.close'       // C→S

// 聊天（1 个）
export const MEETING_WS_CHAT_MESSAGE = 'meeting.chat.message'           // S→C

/** C→S 客户端可主动发起的事件白名单 */
export const MEETING_WS_CLIENT_EVENTS = [
  MEETING_WS_ROOM_JOIN,
  MEETING_WS_ROOM_LEAVE,
  MEETING_WS_MEMBER_STATE_CHANGED,
  MEETING_WS_TRANSPORT_CREATE,
  MEETING_WS_TRANSPORT_CONNECT,
  MEETING_WS_PRODUCE_START,
  MEETING_WS_CONSUME_START,
  MEETING_WS_CONSUME_RESUME,
  MEETING_WS_PRODUCER_CLOSE
]

/** S→C 服务端广播事件白名单（用于 Store 注册监听时统一处理） */
export const MEETING_WS_SERVER_EVENTS = [
  MEETING_WS_ROOM_ENDED,
  MEETING_WS_MEMBER_JOINED,
  MEETING_WS_MEMBER_LEFT,
  MEETING_WS_MEMBER_STATE_CHANGED,
  MEETING_WS_MEMBER_KICKED,
  MEETING_WS_MEMBER_PRODUCER_NEW,
  MEETING_WS_HOST_CHANGED,
  MEETING_WS_CHAT_MESSAGE
]

// ==================== 本地会议状态机（前端专用，与后端 meeting_rooms.status 无关） ====================

// Task 9 用于 store/meeting.js 驱动 UI 的本地状态
export const MEETING_LOCAL_STATE_IDLE = 'idle'                 // 未进入会议
export const MEETING_LOCAL_STATE_JOINING = 'joining'           // REST JoinRoom 中
export const MEETING_LOCAL_STATE_CONNECTING = 'connecting'     // Device load + Transport 创建中
export const MEETING_LOCAL_STATE_CONNECTED = 'connected'       // 媒体层就绪，可推流/收流
export const MEETING_LOCAL_STATE_RECONNECTING = 'reconnecting' // WS 断线重连期间（Task 8 host 宽限期共用）
export const MEETING_LOCAL_STATE_LEAVING = 'leaving'           // REST LeaveRoom 中
export const MEETING_LOCAL_STATE_ENDED = 'ended'               // 会议已结束（room.ended 收到后进入）

export const MEETING_LOCAL_STATE_LABEL = {
  [MEETING_LOCAL_STATE_IDLE]: '未入会',
  [MEETING_LOCAL_STATE_JOINING]: '加入中',
  [MEETING_LOCAL_STATE_CONNECTING]: '连接中',
  [MEETING_LOCAL_STATE_CONNECTED]: '已连接',
  [MEETING_LOCAL_STATE_RECONNECTING]: '重连中',
  [MEETING_LOCAL_STATE_LEAVING]: '离开中',
  [MEETING_LOCAL_STATE_ENDED]: '已结束'
}
