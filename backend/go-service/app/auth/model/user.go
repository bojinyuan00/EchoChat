// Package model 定义 auth 模块的数据库模型
package model

import "time"

// User 用户主表模型，对应 auth_users 表
type User struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`                            // 用户唯一标识，自增主键
	Username     string     `json:"username" gorm:"uniqueIndex;size:50;not null"`                  // 登录用户名，全局唯一
	Email        string     `json:"email" gorm:"uniqueIndex;size:100;not null"`                    // 邮箱地址，全局唯一，用于登录和找回密码
	PasswordHash string     `json:"-" gorm:"column:password_hash;size:255;not null"`               // bcrypt 加密后的密码哈希，JSON 序列化时隐藏
	Nickname     string     `json:"nickname" gorm:"size:50;not null;default:''"`                   // 用户昵称，用于页面展示，允许重复
	Avatar       string     `json:"avatar" gorm:"size:500;not null;default:''"`                    // 头像 URL 地址，为空则使用默认头像
	Gender       int        `json:"gender" gorm:"not null;default:0"`                              // 性别：0=未知, 1=男, 2=女（对应 constants.Gender*）
	Phone        *string    `json:"phone" gorm:"size:20"`                                          // 手机号，可选字段，指针类型允许 NULL
	Status       int        `json:"status" gorm:"not null;default:1"`                              // 账号状态：1=正常, 2=禁用, 3=注销（对应 constants.UserStatus*）
	LastLoginAt  *time.Time `json:"last_login_at" gorm:"type:timestamp(0)"`                        // 最后登录时间，首次注册时为 NULL
	LastLoginIP  *string    `json:"last_login_ip" gorm:"column:last_login_ip;size:50"`             // 最后登录 IP 地址，用于安全审计
	CreatedAt    time.Time  `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"`   // 账号创建时间，由 GORM 自动填充，精确到秒
	UpdatedAt    time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime;type:timestamp(0)"`   // 最后更新时间，由 GORM 自动更新，精确到秒
}

// TableName 指定数据库表名
func (User) TableName() string {
	return "auth_users"
}
