package dto

// SendFriendRequestReq 发送好友申请请求
type SendFriendRequestReq struct {
	TargetID int64  `json:"target_id" binding:"required"` // 目标用户 ID
	Message  string `json:"message"`                       // 申请附言
}

// HandleFriendRequestReq 处理好友申请请求
type HandleFriendRequestReq struct {
	RequestID int64 `json:"request_id" binding:"required"` // 申请记录 ID
}

// UpdateRemarkReq 更新好友备注请求
type UpdateRemarkReq struct {
	Remark string `json:"remark" binding:"max=50"` // 备注名
}

// MoveToGroupReq 移动好友到分组请求
type MoveToGroupReq struct {
	GroupID *int64 `json:"group_id"` // 目标分组 ID，nil 表示移出分组
}

// BlockUserReq 拉黑用户请求
type BlockUserReq struct {
	TargetID int64 `json:"target_id" binding:"required"` // 目标用户 ID
}

// CreateGroupReq 创建好友分组请求
type CreateGroupReq struct {
	Name string `json:"name" binding:"required,max=50"` // 分组名称
}

// UpdateGroupReq 修改好友分组请求
type UpdateGroupReq struct {
	Name      string `json:"name" binding:"required,max=50"` // 分组名称
	SortOrder *int   `json:"sort_order"`                      // 排序权重
}

// SearchUsersReq 搜索用户请求
type SearchUsersReq struct {
	Keyword  string `form:"keyword" binding:"required,min=1"` // 搜索关键词
	Page     int    `form:"page" binding:"min=1"`             // 页码
	PageSize int    `form:"page_size" binding:"min=1,max=50"` // 每页数量
}

// FriendInfo 好友信息（列表展示用）
type FriendInfo struct {
	ID        int64  `json:"id"`                   // friendship 记录 ID
	UserID    int64  `json:"user_id"`              // 好友用户 ID
	Username  string `json:"username"`             // 用户名
	Nickname  string `json:"nickname"`             // 昵称
	Avatar    string `json:"avatar"`               // 头像
	Remark    string `json:"remark"`               // 备注名
	GroupID   *int64 `json:"group_id"`             // 所属分组
	IsOnline  bool   `json:"is_online"`            // 是否在线
	CreatedAt string `json:"created_at"`           // 成为好友时间
}

// FriendRequestInfo 好友申请信息
type FriendRequestInfo struct {
	ID        int64  `json:"id"`          // 申请记录 ID
	UserID    int64  `json:"user_id"`     // 申请方用户 ID
	Username  string `json:"username"`    // 申请方用户名
	Nickname  string `json:"nickname"`    // 申请方昵称
	Avatar    string `json:"avatar"`      // 申请方头像
	Message   string `json:"message"`     // 申请附言
	Status    int    `json:"status"`      // 状态
	CreatedAt string `json:"created_at"`  // 申请时间
}

// GroupInfo 好友分组信息
type GroupInfo struct {
	ID          int64  `json:"id"`           // 分组 ID
	Name        string `json:"name"`         // 分组名称
	SortOrder   int    `json:"sort_order"`   // 排序权重
	FriendCount int    `json:"friend_count"` // 分组内好友数
}

// SearchUserInfo 搜索结果用户信息
type SearchUserInfo struct {
	ID       int64  `json:"id"`        // 用户 ID
	Username string `json:"username"`  // 用户名
	Nickname string `json:"nickname"`  // 昵称
	Avatar   string `json:"avatar"`    // 头像
	IsFriend bool   `json:"is_friend"` // 是否已是好友
}
