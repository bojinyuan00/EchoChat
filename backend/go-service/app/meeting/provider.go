package meeting

import (
	"github.com/echochat/backend/app/meeting/controller"
	"github.com/echochat/backend/app/meeting/dao"
	"github.com/echochat/backend/app/meeting/service"
	"github.com/google/wire"
)

// MeetingSet 会议模块 Wire Provider Set
// 对外暴露：
//   - *service.MeetingService       —— REST 业务服务（Task 5）
//   - *service.MeetingBroadcaster   —— WS 广播中枢（Task 6 新增，供 SignalService 复用）
//   - *service.MeetingSignalService —— WS 信令事件业务逻辑（Task 6）
//   - *controller.MeetingController —— REST API 控制器
//   - *controller.MeetingWSHandler  —— WS 事件 Handler（Task 6，启动时自动注册到 Hub）
// 依赖的接口 NotifyPusher / UserInfoResolver / OnlineChecker 由上游 wire.Bind 绑定具体实现
var MeetingSet = wire.NewSet(
	dao.NewMeetingRoomDAO,
	dao.NewMeetingParticipantDAO,
	dao.NewMeetingChatDAO,
	service.NewMeetingBroadcaster,
	service.NewMeetingService,
	service.NewMeetingSignalService,
	controller.NewMeetingController,
	controller.NewMeetingWSHandler,

	// MediaOrchestrator 目前使用 Noop 实现（Task 7 将替换为 node_client.NodeClient）
	service.NewNoopMediaOrchestrator,
	wire.Bind(new(service.MediaOrchestrator), new(*service.NoopMediaOrchestrator)),
)
