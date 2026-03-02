// Package admin 提供管理后台功能模块
package admin

import (
	"github.com/echochat/backend/app/admin/controller"
	"github.com/echochat/backend/app/admin/dao"
	"github.com/echochat/backend/app/admin/service"
	"github.com/google/wire"
)

// AdminSet Admin 模块依赖注入 Provider Set
// 提供管理端的 DAO、Service 和 Controller 组件
var AdminSet = wire.NewSet(
	dao.NewUserManageDAO,
	service.NewUserManageService,
	controller.NewUserManageController,
	service.NewOnlineManageService,
	controller.NewOnlineController,
	service.NewContactManageService,
	controller.NewContactManageController,
)
