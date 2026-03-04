package dto

// ====== 群聊管理 DTO ======

// CreateGroupRequest 创建群聊请求
// POST /api/v1/groups
type CreateGroupRequest struct {
	Name      string  `json:"name" binding:"required,max=100"`  // 群名称
	MemberIDs []int64 `json:"member_ids" binding:"required"`    // 初始成员用户 ID 列表（不含创建者自身）
	Avatar    string  `json:"avatar"`                           // 群头像 URL（可选）
}

// UpdateGroupRequest 更新群信息请求
// PUT /api/v1/groups/:id
type UpdateGroupRequest struct {
	Name         *string `json:"name"`          // 群名称
	Avatar       *string `json:"avatar"`        // 群头像 URL
	Notice       *string `json:"notice"`        // 群公告
	IsSearchable *bool   `json:"is_searchable"` // 是否可被搜索
}

// GroupDTO 群聊信息传输对象（返回给前端）
type GroupDTO struct {
	ID             int64  `json:"id"`              // 群 ID
	ConversationID int64  `json:"conversation_id"` // 关联会话 ID
	Name           string `json:"name"`            // 群名称
	Avatar         string `json:"avatar"`          // 群头像
	OwnerID        int64  `json:"owner_id"`        // 群主用户 ID
	Notice         string `json:"notice"`          // 群公告
	MaxMembers     int    `json:"max_members"`     // 最大成员数
	MemberCount    int    `json:"member_count"`    // 当前成员数
	IsSearchable   bool   `json:"is_searchable"`   // 是否可被搜索
	IsAllMuted     bool   `json:"is_all_muted"`    // 是否全体禁言
	Status         int    `json:"status"`          // 群状态：1=正常，2=已解散
	CreatedAt      string `json:"created_at"`      // 创建时间
}

// ====== 群成员 DTO ======

// GroupMemberDTO 群成员信息传输对象
type GroupMemberDTO struct {
	UserID         int64  `json:"user_id"`          // 用户 ID
	Nickname       string `json:"nickname"`         // 群内昵称
	UserNickname   string `json:"user_nickname"`    // 用户原始昵称
	Avatar         string `json:"avatar"`           // 用户头像
	Role           int    `json:"role"`             // 角色：0=普通成员，1=管理员，2=群主
	IsMuted        bool   `json:"is_muted"`         // 是否被禁言
	JoinedAt       string `json:"joined_at"`        // 加入时间
}

// InviteMembersRequest 邀请入群请求
// POST /api/v1/groups/:id/members
type InviteMembersRequest struct {
	UserIDs []int64 `json:"user_ids" binding:"required"` // 被邀请的用户 ID 列表
}

// SetRoleRequest 设置/取消管理员请求
// PUT /api/v1/groups/:id/members/:uid/role
type SetRoleRequest struct {
	Role int `json:"role" binding:"oneof=0 1"` // 目标角色：0=普通成员，1=管理员
}

// MuteRequest 禁言/解除禁言请求
// PUT /api/v1/groups/:id/members/:uid/mute
type MuteRequest struct {
	IsMuted bool `json:"is_muted"` // true=禁言, false=解除
}

// TransferOwnerRequest 转让群主请求
// PUT /api/v1/groups/:id/transfer
type TransferOwnerRequest struct {
	NewOwnerID int64 `json:"new_owner_id" binding:"required"` // 新群主用户 ID
}

// UpdateNicknameRequest 修改群昵称请求
// PUT /api/v1/groups/:id/members/me/nickname
type UpdateNicknameRequest struct {
	Nickname string `json:"nickname" binding:"max=50"` // 群内昵称
}

// ====== 入群申请 DTO ======

// JoinGroupRequest 申请入群请求
// POST /api/v1/groups/:id/join-requests
type JoinGroupRequest struct {
	Message string `json:"message"` // 申请附言
}

// ReviewJoinRequest 审批入群申请请求
// PUT /api/v1/groups/:id/join-requests/:rid
type ReviewJoinRequest struct {
	Action string `json:"action" binding:"required,oneof=approve reject"` // 操作：approve=通过, reject=拒绝
}

// JoinRequestDTO 入群申请传输对象
type JoinRequestDTO struct {
	ID           int64  `json:"id"`            // 申请 ID
	GroupID      int64  `json:"group_id"`      // 群 ID
	UserID       int64  `json:"user_id"`       // 申请人用户 ID
	UserNickname string `json:"user_nickname"` // 申请人昵称
	UserAvatar   string `json:"user_avatar"`   // 申请人头像
	Message      string `json:"message"`       // 申请附言
	Status       int    `json:"status"`        // 状态：0=待审批，1=通过，2=拒绝
	CreatedAt    string `json:"created_at"`    // 申请时间
}

// ====== 群搜索 DTO ======

// SearchGroupRequest 搜索群聊请求
// GET /api/v1/groups/search?keyword=xx&page=1&page_size=20
type SearchGroupRequest struct {
	Keyword  string `form:"keyword" binding:"required"` // 搜索关键词
	Page     int    `form:"page"`                       // 页码（默认 1）
	PageSize int    `form:"page_size"`                  // 每页数量（默认 20）
}

// SearchGroupResponse 搜索群聊响应
type SearchGroupResponse struct {
	List  []GroupDTO `json:"list"`  // 搜索结果
	Total int64      `json:"total"` // 总数
}

// ====== 已读回执 DTO ======

// MarkGroupReadRequest 群聊标记已读请求（WS 事件 im.message.read 的 data 字段）
type MarkGroupReadRequest struct {
	ConversationID int64   `json:"conversation_id"` // 会话 ID
	MessageIDs     []int64 `json:"message_ids"`     // 已读消息 ID 列表
}

// MessageReadCountDTO 消息已读计数传输对象
type MessageReadCountDTO struct {
	MessageID int64 `json:"message_id"` // 消息 ID
	ReadCount int   `json:"read_count"` // 已读人数
}

// MessageReadDetailDTO 消息已读详情传输对象
type MessageReadDetailDTO struct {
	UserID       int64  `json:"user_id"`       // 已读用户 ID
	UserNickname string `json:"user_nickname"` // 用户昵称
	UserAvatar   string `json:"user_avatar"`   // 用户头像
	ReadAt       string `json:"read_at"`       // 已读时间
}

// GetReadDetailRequest 获取已读详情请求
// GET /api/v1/im/messages/:id/reads?page=1&page_size=20
type GetReadDetailRequest struct {
	Page     int `form:"page"`      // 页码（默认 1）
	PageSize int `form:"page_size"` // 每页数量（默认 20）
}

// GetReadDetailResponse 获取已读详情响应
type GetReadDetailResponse struct {
	ReadList   []MessageReadDetailDTO `json:"read_list"`   // 已读用户列表
	UnreadList []MessageReadDetailDTO `json:"unread_list"` // 未读用户列表
	ReadCount  int                    `json:"read_count"`  // 已读人数
	TotalCount int                    `json:"total_count"` // 群成员总数（不含发送者）
}

// ====== 免打扰 DTO ======

// SetDoNotDisturbRequest 设置/取消免打扰请求
// PUT /api/v1/im/conversations/:id/dnd
type SetDoNotDisturbRequest struct {
	IsDoNotDisturb bool `json:"is_do_not_disturb"` // true=开启, false=关闭
}

// ====== 跨模块接口数据传输 ======

// GroupBrief 群简要信息（IM 模块通过 GroupInfoGetter 接口获取）
type GroupBrief struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
	IsAllMuted bool   `json:"is_all_muted"`
	Status     int    `json:"status"`
}
