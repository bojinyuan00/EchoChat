// Package auth 提供用户认证与授权模块
package auth

import (
	"github.com/echochat/backend/app/auth/controller"
	"github.com/echochat/backend/app/auth/dao"
	"github.com/echochat/backend/app/auth/service"
	"github.com/google/wire"
)

// AuthSet Auth 模块依赖注入 Provider Set
// 提供 DAO 层、Service 层和 Controller 层的所有组件
var AuthSet = wire.NewSet(
	dao.NewUserDAO,
	dao.NewRoleDAO,
	service.NewTokenStore,
	service.NewAuthService,
	controller.NewAuthController,
	controller.NewAdminAuthController,
)
