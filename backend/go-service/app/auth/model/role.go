package model

import "time"

// Role 角色表模型，对应 auth_roles 表
type Role struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Code        string    `json:"code" gorm:"uniqueIndex;size:50;not null"`   // 角色代码: user/admin/super_admin
	Name        string    `json:"name" gorm:"size:50;not null"`               // 角色显示名称
	Description string    `json:"description" gorm:"size:200;default:''"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
}

// TableName 指定数据库表名
func (Role) TableName() string {
	return "auth_roles"
}

// UserRole 用户角色关联表模型，对应 auth_user_roles 表
type UserRole struct {
	UserID    int64     `json:"user_id" gorm:"primaryKey"`
	RoleID    int       `json:"role_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
}

// TableName 指定数据库表名
func (UserRole) TableName() string {
	return "auth_user_roles"
}
