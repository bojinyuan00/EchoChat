// Package constants 定义系统全局常量
package constants

// 用户账号状态
const (
	UserStatusActive   = 1 // 正常
	UserStatusDisabled = 2 // 禁用（管理员封禁）
	UserStatusDeleted  = 3 // 注销（用户主动注销）
)

// UserStatusMap 用户状态中文映射
var UserStatusMap = map[int]string{
	UserStatusActive:   "正常",
	UserStatusDisabled: "禁用",
	UserStatusDeleted:  "注销",
}

// 用户性别
const (
	GenderUnknown = 0 // 未知
	GenderMale    = 1 // 男
	GenderFemale  = 2 // 女
)
