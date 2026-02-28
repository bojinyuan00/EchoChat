// Package dto 定义数据传输对象，用于 Controller 层的请求/响应参数绑定
package dto

// RegisterRequest 用户注册请求参数
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"` // 用户名，3-50 字符
	Email    string `json:"email" binding:"required,email"`          // 邮箱地址，必须符合邮箱格式
	Password string `json:"password" binding:"required,min=6,max=50"` // 密码，6-50 字符
	Nickname string `json:"nickname" binding:"max=50"`               // 昵称，可选，最多 50 字符
}

// LoginRequest 用户登录请求参数
type LoginRequest struct {
	Account  string `json:"account" binding:"required"`  // 登录账号，支持用户名或邮箱
	Password string `json:"password" binding:"required"` // 登录密码
}

// LoginResponse 登录成功响应数据
type LoginResponse struct {
	Token        string   `json:"token"`         // Access Token，用于接口认证
	RefreshToken string   `json:"refresh_token"` // Refresh Token，用于刷新 Access Token
	ExpiresIn    int64    `json:"expires_in"`    // Access Token 有效期（秒）
	User         UserInfo `json:"user"`          // 当前登录用户基本信息
}

// UserInfo 用户基本信息（用于响应返回，不含敏感字段）
type UserInfo struct {
	ID       int64    `json:"id"`       // 用户 ID
	Username string   `json:"username"` // 用户名
	Email    string   `json:"email"`    // 邮箱
	Nickname string   `json:"nickname"` // 昵称
	Avatar   string   `json:"avatar"`   // 头像 URL
	Gender   int      `json:"gender"`   // 性别：0=未知, 1=男, 2=女
	Roles    []string `json:"roles"`    // 角色代码列表
}

// RefreshTokenRequest 刷新 Token 请求参数
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"` // 原 Refresh Token
}

// UpdateProfileRequest 更新个人资料请求参数
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"max=50"`  // 新昵称
	Avatar   string `json:"avatar" binding:"max=500"`   // 新头像 URL
	Gender   *int   `json:"gender" binding:"omitempty,oneof=0 1 2"` // 新性别，指针类型区分「未传」和「传 0」
	Phone    string `json:"phone" binding:"max=20"`     // 新手机号
}

// ChangePasswordRequest 修改密码请求参数
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`         // 旧密码
	NewPassword string `json:"new_password" binding:"required,min=6,max=50"` // 新密码，6-50 字符
}
