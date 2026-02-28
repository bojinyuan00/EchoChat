// Package auth 提供用户认证与授权模块
package auth

import (
	"github.com/echochat/backend/app/auth/dao"
	"github.com/google/wire"
)

// AuthSet Auth 模块依赖注入 Provider Set
var AuthSet = wire.NewSet(
	dao.NewUserDAO,
	dao.NewRoleDAO,
)
