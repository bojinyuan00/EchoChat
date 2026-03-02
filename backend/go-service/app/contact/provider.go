package contact

import (
	"github.com/echochat/backend/app/contact/controller"
	"github.com/echochat/backend/app/contact/dao"
	"github.com/echochat/backend/app/contact/service"
	"github.com/google/wire"
)

// ContactSet Contact 模块依赖注入 Provider Set
var ContactSet = wire.NewSet(
	dao.NewFriendshipDAO,
	dao.NewFriendGroupDAO,
	service.NewContactService,
	controller.NewContactController,
)
