// Package dto 定义数据传输对象
package dto

// UserListRequest 管理端用户列表查询请求参数
type UserListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`     // 页码，从 1 开始
	PageSize int    `form:"page_size" binding:"required,min=1,max=100"` // 每页数量，1-100
	Keyword  string `form:"keyword"`                           // 搜索关键词（匹配用户名或邮箱）
	Status   *int   `form:"status"`                            // 状态筛选：1=正常, 2=禁用, 3=注销；为空则不筛选
}

// UserListResponse 管理端用户列表响应
type UserListResponse struct {
	Total int64          `json:"total"` // 符合条件的用户总数
	List  []AdminUserInfo `json:"list"` // 当前页的用户列表
}

// AdminUserInfo 管理端用户信息（比 UserInfo 更详细，包含管理字段）
type AdminUserInfo struct {
	ID          int64    `json:"id"`                     // 用户 ID
	Username    string   `json:"username"`               // 用户名
	Email       string   `json:"email"`                  // 邮箱
	Nickname    string   `json:"nickname"`               // 昵称
	Avatar      string   `json:"avatar"`                 // 头像 URL
	Gender      int      `json:"gender"`                 // 性别：0=未知, 1=男, 2=女
	Phone       string   `json:"phone,omitempty"`        // 手机号
	Status      int      `json:"status"`                 // 账号状态
	StatusText  string   `json:"status_text"`            // 状态中文描述
	Roles       []string `json:"roles"`                  // 角色代码列表
	LastLoginAt string   `json:"last_login_at,omitempty"` // 最后登录时间
	LastLoginIP string   `json:"last_login_ip,omitempty"` // 最后登录 IP
	CreatedAt   string   `json:"created_at"`             // 注册时间
	UpdatedAt   string   `json:"updated_at"`             // 更新时间
}

// UpdateUserStatusRequest 更新用户状态请求
type UpdateUserStatusRequest struct {
	Status int `json:"status" binding:"required,oneof=1 2"` // 目标状态：1=正常, 2=禁用
}

// AssignRoleRequest 分配角色请求
type AssignRoleRequest struct {
	RoleCode string `json:"role_code" binding:"required"` // 角色代码：user/admin/super_admin
}

// AdminCreateUserRequest 管理员手动创建用户请求
type AdminCreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"` // 用户名
	Email    string `json:"email" binding:"required,email"`           // 邮箱
	Password string `json:"password" binding:"required,min=6,max=50"` // 初始密码
	Nickname string `json:"nickname" binding:"max=50"`                // 昵称（选填）
	RoleCode string `json:"role_code" binding:"omitempty"`            // 初始角色（选填，默认 user）
}
