package constants

// 角色代码
const (
	RoleUser       = "user"        // 普通用户
	RoleAdmin      = "admin"       // 管理员
	RoleSuperAdmin = "super_admin" // 超级管理员
)

// RoleNameMap 角色代码到中文名的映射
var RoleNameMap = map[string]string{
	RoleUser:       "普通用户",
	RoleAdmin:      "管理员",
	RoleSuperAdmin: "超级管理员",
}
