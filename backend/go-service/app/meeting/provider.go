package meeting

import (
	"github.com/echochat/backend/app/meeting/controller"
	"github.com/echochat/backend/app/meeting/dao"
	"github.com/echochat/backend/app/meeting/service"
	"github.com/google/wire"
)

// MeetingSet 会议模块 Wire Provider Set
// 对外暴露：
//   - *service.MeetingService  —— 业务服务，供未来 ws handler / media-server 回调使用
//   - *controller.MeetingController —— REST API 控制器
// 依赖的接口 NotifyPusher / UserInfoResolver / OnlineChecker 由上游 wire.Bind 绑定具体实现
var MeetingSet = wire.NewSet(
	dao.NewMeetingRoomDAO,
	dao.NewMeetingParticipantDAO,
	dao.NewMeetingChatDAO,
	service.NewMeetingService,
	controller.NewMeetingController,

	// MediaOrchestrator 目前使用 Noop 实现（Task 7 将替换为 node_client.NodeClient）
	service.NewNoopMediaOrchestrator,
	wire.Bind(new(service.MediaOrchestrator), new(*service.NoopMediaOrchestrator)),
)
