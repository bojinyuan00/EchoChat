// Package group 群聊管理模块
package group

import (
	"github.com/echochat/backend/app/group/controller"
	"github.com/echochat/backend/app/group/dao"
	"github.com/echochat/backend/app/group/service"
	"github.com/google/wire"
)

// GroupSet 群聊模块 Wire Provider Set
var GroupSet = wire.NewSet(
	dao.NewGroupDAO,
	dao.NewJoinRequestDAO,
	dao.NewMessageReadDAO,
	service.NewGroupService,
	controller.NewGroupController,
)
