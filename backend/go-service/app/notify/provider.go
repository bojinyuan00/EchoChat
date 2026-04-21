// Package notify 通知中心模块
// 统一持久化好友 / 群聊 / 会议 / 系统广播四类通知，并通过 WS 实时推送
package notify

import (
	"github.com/echochat/backend/app/notify/controller"
	"github.com/echochat/backend/app/notify/dao"
	"github.com/echochat/backend/app/notify/service"
	"github.com/echochat/backend/app/notify/task"
	"github.com/google/wire"
)

// NotifySet 通知模块 Wire Provider Set
// 对外暴露：
//   - *service.NotifyService    —— 业务服务，同时实现 Pusher / ConnectHook 接口，供上游模块绑定注入
//   - *controller.NotificationController —— REST API 控制器
//   - *task.CleanupTask         —— 过期通知清理定时任务
var NotifySet = wire.NewSet(
	dao.NewNotificationDAO,
	service.NewNotifyService,
	controller.NewNotificationController,
	task.NewCleanupTask,
)
