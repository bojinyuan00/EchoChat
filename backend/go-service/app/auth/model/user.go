// Package model 定义 auth 模块的数据库模型
package model

import "time"

// User 用户主表模型，对应 auth_users 表
type User struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string     `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Email        string     `json:"email" gorm:"uniqueIndex;size:100;not null"`
	PasswordHash string     `json:"-" gorm:"column:password_hash;size:255;not null"` // JSON 序列化时隐藏密码
	Nickname     string     `json:"nickname" gorm:"size:50;not null;default:''"`
	Avatar       string     `json:"avatar" gorm:"size:500;not null;default:''"`
	Gender       int        `json:"gender" gorm:"not null;default:0"`           // 0=未知, 1=男, 2=女
	Phone        *string    `json:"phone" gorm:"size:20"`                       // 可选字段
	Status       int        `json:"status" gorm:"not null;default:1"`           // 1=正常, 2=禁用, 3=注销
	LastLoginAt  *time.Time `json:"last_login_at"`
	LastLoginIP  *string    `json:"last_login_ip" gorm:"column:last_login_ip;size:50"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

// TableName 指定数据库表名
func (User) TableName() string {
	return "auth_users"
}
