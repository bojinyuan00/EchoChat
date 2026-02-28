package model

import "time"

// Role 角色表模型，对应 auth_roles 表
// 系统预置三种角色：user（普通用户）、admin（管理员）、super_admin（超级管理员）
type Role struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`                          // 角色唯一标识，自增主键
	Code        string    `json:"code" gorm:"uniqueIndex;size:50;not null"`                    // 角色代码，全局唯一，如 user/admin/super_admin（对应 constants.RoleCode*）
	Name        string    `json:"name" gorm:"size:50;not null"`                                // 角色中文显示名称，如「普通用户」「管理员」
	Description string    `json:"description" gorm:"size:200;default:''"`                      // 角色描述说明，用于后台管理界面展示
	CreatedAt   time.Time `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"` // 创建时间，由 GORM 自动填充，精确到秒
}

// TableName 指定数据库表名
func (Role) TableName() string {
	return "auth_roles"
}

// UserRole 用户角色关联表模型，对应 auth_user_roles 表
// 采用联合主键（user_id + role_id），一个用户可拥有多个角色
type UserRole struct {
	UserID    int64     `json:"user_id" gorm:"primaryKey"`                                   // 关联的用户 ID，外键指向 auth_users.id
	RoleID    int       `json:"role_id" gorm:"primaryKey"`                                   // 关联的角色 ID，外键指向 auth_roles.id
	CreatedAt time.Time `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"` // 角色分配时间，由 GORM 自动填充，精确到秒
}

// TableName 指定数据库表名
func (UserRole) TableName() string {
	return "auth_user_roles"
}
