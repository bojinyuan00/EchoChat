package im

import (
	"github.com/echochat/backend/app/im/controller"
	"github.com/echochat/backend/app/im/dao"
	"github.com/echochat/backend/app/im/handler"
	"github.com/echochat/backend/app/im/service"
	"github.com/echochat/backend/pkg/ws"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ProvideConversationDAO 创建 ConversationDAO 实例
func ProvideConversationDAO(db *gorm.DB) *dao.ConversationDAO {
	return dao.NewConversationDAO(db)
}

// ProvideMessageDAO 创建 MessageDAO 实例
func ProvideMessageDAO(db *gorm.DB) *dao.MessageDAO {
	return dao.NewMessageDAO(db)
}

// ProvideIMService 创建 IMService 实例
// friendChecker 和 userInfoGetter 由 contact 模块的 FriendshipDAO 隐式实现
func ProvideIMService(
	convDAO *dao.ConversationDAO,
	msgDAO *dao.MessageDAO,
	pubsub *ws.PubSub,
	rdb *redis.Client,
	friendChecker service.FriendChecker,
	userInfoGetter service.UserInfoGetter,
) *service.IMService {
	return service.NewIMService(convDAO, msgDAO, pubsub, rdb, friendChecker, userInfoGetter)
}

// ProvideIMEventHandler 创建 IM WS 事件处理器并注册事件到 Hub
func ProvideIMEventHandler(imService *service.IMService, hub *ws.Hub) *handler.EventHandler {
	return handler.NewEventHandler(imService, hub)
}

// ProvideOfflinePusher 创建离线消息推送器
func ProvideOfflinePusher(imService *service.IMService, convDAO *dao.ConversationDAO, pubsub *ws.PubSub) *handler.OfflinePusher {
	return handler.NewOfflinePusher(imService, convDAO, pubsub)
}

// ProvideIMController 创建 IM REST 控制器
func ProvideIMController(imService *service.IMService) *controller.IMController {
	return controller.NewIMController(imService)
}

// IMSet IM 模块 Wire Provider Set
var IMSet = wire.NewSet(
	ProvideConversationDAO,
	ProvideMessageDAO,
	ProvideIMService,
	ProvideIMEventHandler,
	ProvideOfflinePusher,
	ProvideIMController,
)
