package dto

// ====== 会议模块基础 DTO（Phase 2e-2）======

// MeetingRoomDTO 会议房间传输对象
// 用于 CreateRoom / GetRoom / JoinRoom / ListMyMeetings 的响应体
// 字段与 model.MeetingRoom 对齐，敏感字段（password_hash）不对外暴露
type MeetingRoomDTO struct {
	ID           int64   `json:"id"`
	RoomCode     string  `json:"room_code"`
	Title        string  `json:"title"`
	HostID       int64   `json:"host_id"`
	HostName     string  `json:"host_name,omitempty"`
	HostAvatar   string  `json:"host_avatar,omitempty"`
	Type         int     `json:"type"`
	HasPassword  bool    `json:"has_password"`
	MaxMembers   int     `json:"max_members"`
	Status       int     `json:"status"`
	StatusLabel  string  `json:"status_label,omitempty"`
	ScheduledAt  string  `json:"scheduled_at,omitempty"`
	StartedAt    string  `json:"started_at,omitempty"`
	EndedAt      string  `json:"ended_at,omitempty"`
	EndedReason  *string `json:"ended_reason,omitempty"`
	Settings     string  `json:"settings"`
	CreatedAt    string  `json:"created_at"`
	OnlineCount  int     `json:"online_count"`
}

// MeetingParticipantDTO 会议参与者传输对象
// 用于 GetRoom 的成员列表响应；活跃用户（left_at=NULL）在前
type MeetingParticipantDTO struct {
	ID         int64   `json:"id"`
	RoomID     int64   `json:"room_id"`
	UserID     int64   `json:"user_id"`
	UserName   string  `json:"user_name,omitempty"`
	UserAvatar string  `json:"user_avatar,omitempty"`
	Role       int     `json:"role"`
	RoleLabel  string  `json:"role_label,omitempty"`
	JoinedAt   string  `json:"joined_at"`
	LeftAt     string  `json:"left_at,omitempty"`
	LeftReason *string `json:"left_reason,omitempty"`
	Duration   int     `json:"duration"`
	IsActive   bool    `json:"is_active"`
}

// MeetingChatDTO 会议内聊天消息传输对象
// 用于 SendChat 的响应 + ListChats 的列表项
type MeetingChatDTO struct {
	ID         int64  `json:"id"`
	RoomID     int64  `json:"room_id"`
	UserID     int64  `json:"user_id"`
	UserName   string `json:"user_name,omitempty"`
	UserAvatar string `json:"user_avatar,omitempty"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
}

// ====== REST API 请求 / 响应 ======

// CreateMeetingRoomRequest 创建即时会议请求
// POST /api/v1/meeting/rooms
type CreateMeetingRoomRequest struct {
	Title      string `json:"title" binding:"required,min=1,max=200"`  // 会议标题
	Password   string `json:"password" binding:"omitempty,min=4,max=20"` // 可选入会密码（纯文本入参，服务端 bcrypt 哈希存储）
	MaxMembers int    `json:"max_members" binding:"omitempty,min=2,max=8"` // 可选上限（MVP 硬上限 8，留参数为 Phase 2e-3 扩展）
}

// CreateMeetingRoomResponse 创建会议响应
type CreateMeetingRoomResponse struct {
	Room MeetingRoomDTO `json:"room"`
}

// JoinMeetingRoomRequest 加入会议请求
// POST /api/v1/meeting/rooms/:code/join
type JoinMeetingRoomRequest struct {
	Password string `json:"password" binding:"omitempty,max=20"` // 入会密码（如房间设密则必传）
}

// JoinMeetingRoomResponse 加入会议响应
type JoinMeetingRoomResponse struct {
	Room        MeetingRoomDTO        `json:"room"`
	Participant MeetingParticipantDTO `json:"participant"`
	RouterID    string                `json:"router_id,omitempty"` // mediasoup Router ID（Task 7 接入 Node 后填充）
}

// LeaveMeetingRoomResponse 离开会议响应
type LeaveMeetingRoomResponse struct {
	Duration int `json:"duration"` // 本次参会时长（秒）
}

// GetMeetingRoomResponse 会议详情响应
// GET /api/v1/meeting/rooms/:code
type GetMeetingRoomResponse struct {
	Room         MeetingRoomDTO          `json:"room"`
	Participants []MeetingParticipantDTO `json:"participants"`
	OnlineCount  int                     `json:"online_count"`
}

// ListMyMeetingsRequest 我的会议列表查询参数
// GET /api/v1/meeting/rooms/mine
type ListMyMeetingsRequest struct {
	Status   *int  `form:"status"`     // 状态过滤：nil=全部，0/1/2
	BeforeID int64 `form:"before_id"`  // 游标分页
	Limit    int   `form:"limit"`      // 页大小，默认 20，最大 50
}

// ListMyMeetingsResponse 我的会议列表响应
type ListMyMeetingsResponse struct {
	List    []MeetingRoomDTO `json:"list"`
	HasMore bool             `json:"has_more"`
}

// InviteUsersRequest 邀请用户请求
// POST /api/v1/meeting/rooms/:code/invite
type InviteUsersRequest struct {
	InviteeIDs []int64 `json:"invitee_ids" binding:"required,min=1,max=50,dive,gt=0"` // 被邀请用户 ID 数组（去重后）
}

// InviteUsersResponse 邀请结果响应
type InviteUsersResponse struct {
	Pushed  int `json:"pushed"`   // 成功发送邀请通知的数量（在线+离线都算）
	Skipped int `json:"skipped"` // 跳过的数量（已在会中 / 重复 ID 等）
}

// KickMemberRequest 踢人请求
// POST /api/v1/meeting/rooms/:code/kick
type KickMemberRequest struct {
	UserID int64 `json:"user_id" binding:"required,gt=0"`
}

// TransferHostRequest 主持人转让请求
// POST /api/v1/meeting/rooms/:code/transfer-host
type TransferHostRequest struct {
	TargetUserID int64 `json:"target_user_id" binding:"required,gt=0"`
}

// SendMeetingChatRequest 发送会议内聊天请求
// POST /api/v1/meeting/rooms/:code/chats
type SendMeetingChatRequest struct {
	Content string `json:"content" binding:"required,min=1,max=500"`
}

// SendMeetingChatResponse 发送聊天响应
type SendMeetingChatResponse struct {
	Message MeetingChatDTO `json:"message"`
}

// ListMeetingChatsRequest 会议内聊天列表查询参数
// GET /api/v1/meeting/rooms/:code/chats
type ListMeetingChatsRequest struct {
	BeforeID int64 `form:"before_id"` // 游标分页
	Limit    int   `form:"limit"`     // 页大小，默认 30，最大 100
}

// ListMeetingChatsResponse 会议内聊天列表响应
type ListMeetingChatsResponse struct {
	List    []MeetingChatDTO `json:"list"`
	HasMore bool             `json:"has_more"`
}

// RedeemInviteTokenResponse 邀请 Token 兑换响应
// POST /api/v1/meeting/invite-tokens/:token/redeem
// 兑换成功仅返回会议号 + 邀请人，前端自行调 JoinRoom 入会
type RedeemInviteTokenResponse struct {
	RoomCode   string `json:"room_code"`
	InviterID  int64  `json:"inviter_id"`
	HasPassword bool  `json:"has_password"`
}
