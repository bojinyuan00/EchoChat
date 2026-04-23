// Package provider 提供全局依赖注入配置
// 使用 Wire 编译时依赖注入，集中管理所有模块的 Provider Set
package provider

import (
	adminController "github.com/echochat/backend/app/admin/controller"
	authController "github.com/echochat/backend/app/auth/controller"
	"github.com/echochat/backend/app/auth/service"
	contactController "github.com/echochat/backend/app/contact/controller"
	fileController "github.com/echochat/backend/app/file/controller"
	groupController "github.com/echochat/backend/app/group/controller"
	imController "github.com/echochat/backend/app/im/controller"
	imHandler "github.com/echochat/backend/app/im/handler"
	meetingController "github.com/echochat/backend/app/meeting/controller"
	meetingService "github.com/echochat/backend/app/meeting/service"
	meetingTask "github.com/echochat/backend/app/meeting/task"
	notifyController "github.com/echochat/backend/app/notify/controller"
	notifyService "github.com/echochat/backend/app/notify/service"
	notifyTask "github.com/echochat/backend/app/notify/task"
	wsApp "github.com/echochat/backend/app/ws"
	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/db"
	"github.com/echochat/backend/pkg/storage"
	"github.com/echochat/backend/pkg/ws"
	"github.com/google/wire"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// App 应用根容器，持有基础设施组件和各模块的 Controller/Service
type App struct {
	Config                *config.Config
	DB                    *gorm.DB
	Redis                 *redis.Client
	MinioClient           *minio.Client                               // MinIO 对象存储客户端
	AuthService           *service.AuthService                        // Auth 认证服务
	AuthController        *authController.AuthController              // 前台认证控制器
	AdminAuthController   *authController.AdminAuthController         // 后台认证控制器
	UserManageController    *adminController.UserManageController       // 管理端用户管理控制器
	OnlineController        *adminController.OnlineController           // 管理端在线监控控制器
	ContactManageController *adminController.ContactManageController    // 管理端好友关系管理控制器
	GroupManageController   *adminController.GroupManageController      // 管理端群组管理控制器
	MessageManageController *adminController.MessageManageController    // 管理端消息管理控制器
	WSHandler             *wsApp.Handler                              // WebSocket 连接处理器
	Hub                   *ws.Hub                                     // WebSocket Hub 连接管理
	PubSub                *ws.PubSub                                  // Redis Pub/Sub 消息路由
	OnlineService         *wsApp.OnlineService                        // 在线状态管理服务
	ContactController     *contactController.ContactController        // 联系人控制器
	IMController          *imController.IMController                 // IM 即时通讯控制器
	IMEventHandler        *imHandler.EventHandler                    // IM WS 事件处理器
	OfflinePusher         *imHandler.OfflinePusher                   // 离线消息推送器
	FileController        *fileController.FileController             // 文件上传控制器
	GroupController       *groupController.GroupController           // 群聊管理控制器
	NotifyService         *notifyService.NotifyService               // 通知业务服务（兼 Pusher、ConnectHook）
	NotifyController      *notifyController.NotificationController   // 通知控制器
	NotifyCleanupTask     *notifyTask.CleanupTask                    // 通知清理定时任务
	MeetingService        *meetingService.MeetingService             // 会议 REST 业务服务（Task 5 落地）
	MeetingSignalService  *meetingService.MeetingSignalService       // 会议 WS 信令业务服务（Task 6 落地）
	MeetingLifecycleSvc   *meetingService.MeetingLifecycleService    // 会议生命周期状态机（Task 8 落地）
	MeetingController     *meetingController.MeetingController       // 会议 REST 控制器
	MeetingWSHandler      *meetingController.MeetingWSHandler        // 会议 WS 事件 Handler（构造时自动注册路由到 Hub）
	MeetingCleanupTask    *meetingTask.MeetingCleanupTask            // 会议生命周期兜底定时任务（Task 8 落地）
}

// NewApp 创建应用实例
func NewApp(
	cfg *config.Config,
	gormDB *gorm.DB,
	redisClient *redis.Client,
	minioClient *minio.Client,
	authService *service.AuthService,
	authCtrl *authController.AuthController,
	adminAuthCtrl *authController.AdminAuthController,
	userManageCtrl *adminController.UserManageController,
	onlineCtrl *adminController.OnlineController,
	contactManageCtrl *adminController.ContactManageController,
	groupManageCtrl *adminController.GroupManageController,
	msgManageCtrl *adminController.MessageManageController,
	wsHandler *wsApp.Handler,
	hub *ws.Hub,
	pubsub *ws.PubSub,
	onlineService *wsApp.OnlineService,
	contactCtrl *contactController.ContactController,
	imCtrl *imController.IMController,
	imEventHandler *imHandler.EventHandler,
	offlinePusher *imHandler.OfflinePusher,
	fileCtrl *fileController.FileController,
	groupCtrl *groupController.GroupController,
	notifySvc *notifyService.NotifyService,
	notifyCtrl *notifyController.NotificationController,
	notifyCleanup *notifyTask.CleanupTask,
	meetingSvc *meetingService.MeetingService,
	meetingSignalSvc *meetingService.MeetingSignalService,
	meetingLifecycleSvc *meetingService.MeetingLifecycleService,
	meetingCtrl *meetingController.MeetingController,
	meetingWSHandler *meetingController.MeetingWSHandler,
	meetingCleanup *meetingTask.MeetingCleanupTask,
) *App {
	wsHandler.SetOfflinePusher(offlinePusher)
	wsHandler.SetNotifyConnectHook(notifySvc)
	wsHandler.SetMeetingDisconnectHook(meetingSignalSvc)

	return &App{
		Config:                  cfg,
		DB:                      gormDB,
		Redis:                   redisClient,
		MinioClient:             minioClient,
		AuthService:             authService,
		AuthController:          authCtrl,
		AdminAuthController:     adminAuthCtrl,
		UserManageController:    userManageCtrl,
		OnlineController:        onlineCtrl,
		ContactManageController: contactManageCtrl,
		GroupManageController:   groupManageCtrl,
		MessageManageController: msgManageCtrl,
		WSHandler:               wsHandler,
		Hub:                     hub,
		PubSub:                  pubsub,
		OnlineService:           onlineService,
		ContactController:       contactCtrl,
		IMController:            imCtrl,
		IMEventHandler:          imEventHandler,
		OfflinePusher:           offlinePusher,
		FileController:          fileCtrl,
		GroupController:         groupCtrl,
		NotifyService:           notifySvc,
		NotifyController:        notifyCtrl,
		NotifyCleanupTask:       notifyCleanup,
		MeetingService:          meetingSvc,
		MeetingSignalService:    meetingSignalSvc,
		MeetingLifecycleSvc:     meetingLifecycleSvc,
		MeetingController:       meetingCtrl,
		MeetingWSHandler:        meetingWSHandler,
		MeetingCleanupTask:      meetingCleanup,
	}
}

// provideDBConfig 从全局 Config 中提取 DatabaseConfig
func provideDBConfig(cfg *config.Config) *config.DatabaseConfig {
	return &cfg.Database
}

// provideRedisConfig 从全局 Config 中提取 RedisConfig
func provideRedisConfig(cfg *config.Config) *config.RedisConfig {
	return &cfg.Redis
}

// provideJWTConfig 从全局 Config 中提取 JWTConfig
func provideJWTConfig(cfg *config.Config) *config.JWTConfig {
	return &cfg.JWT
}

// provideMinioConfig 从全局 Config 中提取 MinioConfig
func provideMinioConfig(cfg *config.Config) *config.MinioConfig {
	return &cfg.Minio
}

// provideServerConfig 从全局 Config 中提取 ServerConfig
// Task 16 Nit：WebSocket CheckOrigin 白名单收敛需要 server.ws_allowed_origins + server.mode
func provideServerConfig(cfg *config.Config) *config.ServerConfig {
	return &cfg.Server
}

// InfraSet 基础设施层 Provider Set
var InfraSet = wire.NewSet(
	provideDBConfig,
	provideRedisConfig,
	provideJWTConfig,
	provideMinioConfig,
	provideServerConfig,
	db.NewPostgres,
	db.NewRedis,
	storage.NewMinioClient,
	NewApp,
)
