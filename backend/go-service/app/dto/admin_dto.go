// Package dto 定义数据传输对象
package dto

// UserListRequest 管理端用户列表查询请求参数
type UserListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`              // 页码，从 1 开始
	PageSize int    `form:"page_size" binding:"required,min=1,max=100"` // 每页数量，1-100
	Keyword  string `form:"keyword"`                                    // 搜索关键词（匹配用户名或邮箱）
	Status   *int   `form:"status"`                                     // 状态筛选：1=正常, 2=禁用, 3=注销；为空则不筛选
}

// UserListResponse 管理端用户列表响应
type UserListResponse struct {
	Total int64           `json:"total"` // 符合条件的用户总数
	List  []AdminUserInfo `json:"list"`  // 当前页的用户列表
}

// AdminUserInfo 管理端用户信息（比 UserInfo 更详细，包含管理字段）
type AdminUserInfo struct {
	ID          int64      `json:"id"`                      // 用户 ID
	Username    string     `json:"username"`                // 用户名
	Email       string     `json:"email"`                   // 邮箱
	Nickname    string     `json:"nickname"`                // 昵称
	Avatar      string     `json:"avatar"`                  // 头像 URL
	Gender      int        `json:"gender"`                  // 性别：0=未知, 1=男, 2=女
	Phone       string     `json:"phone,omitempty"`         // 手机号
	Status      int        `json:"status"`                  // 账号状态
	StatusText  string     `json:"status_text"`             // 状态中文描述
	Roles       []RoleInfo `json:"roles"`                   // 角色详情列表（含 code/name/level）
	MaxLevel    int        `json:"max_level"`               // 用户最高权限等级（最小 level 值）
	LastLoginAt string     `json:"last_login_at,omitempty"` // 最后登录时间
	LastLoginIP string     `json:"last_login_ip,omitempty"` // 最后登录 IP
	CreatedAt   string     `json:"created_at"`              // 注册时间
	UpdatedAt   string     `json:"updated_at"`              // 更新时间
}

// RoleInfo 角色信息（用于前端角色列表展示和角色分配）
type RoleInfo struct {
	Code  string `json:"code"`  // 角色代码
	Name  string `json:"name"`  // 角色中文名称
	Level int    `json:"level"` // 角色等级，值越小权限越高
}

// UpdateUserStatusRequest 更新用户状态请求
type UpdateUserStatusRequest struct {
	Status int `json:"status" binding:"required,oneof=1 2"` // 目标状态：1=正常, 2=禁用
}

// SetRolesRequest 批量设置用户角色请求（替换原有的单角色分配）
type SetRolesRequest struct {
	RoleCodes []string `json:"role_codes" binding:"required,min=1"` // 角色代码列表
}

// AdminCreateUserRequest 管理员手动创建用户请求
type AdminCreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"` // 用户名
	Email    string `json:"email" binding:"required,email"`           // 邮箱
	Password string `json:"password" binding:"required,min=6,max=50"` // 初始密码
	Nickname string `json:"nickname" binding:"max=50"`                // 昵称（选填）
	RoleCode string `json:"role_code" binding:"omitempty"`            // 初始角色（选填，默认 user）
}

// ====== 管理端消息管理 DTO ======

// AdminMessageListRequest 管理端消息列表查询请求
type AdminMessageListRequest struct {
	Keyword        string `form:"keyword"`         // 搜索关键词（模糊匹配消息内容）
	Type           *int   `form:"type"`            // 消息类型筛选（1/2/3/5/10）
	SenderID       *int64 `form:"sender_id"`       // 发送者 ID
	ConversationID *int64 `form:"conversation_id"` // 会话 ID
	Status         *int   `form:"status"`          // 消息状态（1=正常/2=已撤回/3=已删除）
	StartTime      string `form:"start_time"`      // 开始时间（YYYY-MM-DD）
	EndTime        string `form:"end_time"`        // 结束时间（YYYY-MM-DD）
	Page           int    `form:"page"`            // 页码（默认1）
	PageSize       int    `form:"page_size"`       // 每页条数（默认20）
}

// AdminMessageListResponse 管理端消息列表响应
type AdminMessageListResponse struct {
	Total    int64             `json:"total"`     // 总条数
	List     []AdminMessageDTO `json:"list"`      // 消息列表
	Page     int               `json:"page"`      // 当前页码
	PageSize int               `json:"page_size"` // 每页条数
}

// AdminMessageDTO 管理端消息条目
type AdminMessageDTO struct {
	ID             int64   `json:"id"`                        // 消息 ID
	ConversationID int64   `json:"conversation_id"`           // 会话 ID
	SenderID       int64   `json:"sender_id"`                 // 发送者 ID
	SenderNickname string  `json:"sender_nickname"`           // 发送者昵称
	SenderAvatar   string  `json:"sender_avatar"`             // 发送者头像
	Type           int     `json:"type"`                      // 消息类型
	TypeLabel      string  `json:"type_label"`                // 类型中文标签
	Content        string  `json:"content"`                   // 消息内容
	Extra          *string `json:"extra,omitempty"`           // 扩展数据 JSON
	Status         int     `json:"status"`                    // 消息状态
	StatusLabel    string  `json:"status_label"`              // 状态中文标签
	CreatedAt      string  `json:"created_at"`                // 发送时间
}

// AdminMessageStatsRequest 管理端消息统计请求
type AdminMessageStatsRequest struct {
	Days int `form:"days"` // 统计天数（默认7，最大90）
}

// AdminMessageStatsResponse 管理端消息统计响应
type AdminMessageStatsResponse struct {
	TotalCount       int64              `json:"total_count"`       // 消息总数
	TodayCount       int64              `json:"today_count"`       // 今日消息数
	TypeDistribution []TypeDistItem     `json:"type_distribution"` // 类型分布
	DailyTrend       []DailyTrendItem   `json:"daily_trend"`       // 每日趋势
	ActiveUsers      []ActiveUserItem   `json:"active_users"`      // 活跃用户排行
	ActiveGroups     []ActiveGroupItem  `json:"active_groups"`     // 活跃群组排行
}

// TypeDistItem 消息类型分布条目
type TypeDistItem struct {
	Type  int    `json:"type"`  // 消息类型
	Label string `json:"label"` // 类型中文标签
	Count int64  `json:"count"` // 数量
}

// DailyTrendItem 每日消息趋势条目
type DailyTrendItem struct {
	Date  string `json:"date"`  // 日期（YYYY-MM-DD）
	Count int64  `json:"count"` // 数量
}

// ActiveUserItem 活跃用户排行条目
type ActiveUserItem struct {
	UserID   int64  `json:"user_id"`  // 用户 ID
	Nickname string `json:"nickname"` // 昵称
	Count    int64  `json:"count"`    // 消息数
}

// ActiveGroupItem 活跃群组排行条目
type ActiveGroupItem struct {
	GroupID int64  `json:"group_id"` // 群组 ID
	Name    string `json:"name"`     // 群名称
	Count   int64  `json:"count"`    // 消息数
}
