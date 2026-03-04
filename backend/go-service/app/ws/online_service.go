package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	onlineSetKey       = "echo:user:online"         // 所有在线用户 ID 集合
	statusKeyPrefix    = "echo:user:status:"         // 用户状态 Key 前缀
	statusTTL          = 60 * time.Second            // 状态 TTL，心跳续期
)

// UserStatus 用户在线状态信息（存入 Redis）
type UserStatus struct {
	UserID    int64  `json:"user_id"`
	ConnectAt string `json:"connect_at"` // 连接建立时间
	IP        string `json:"ip"`         // 连接 IP
}

// FriendIDsGetter 获取好友 ID 列表的接口（避免直接依赖 contact 模块）
type FriendIDsGetter interface {
	GetFriendIDs(ctx context.Context, userID int64) ([]int64, error)
}

// OnlineService 在线状态管理服务
type OnlineService struct {
	rdb           *redis.Client
	hub           *ws.Hub
	pubsub        *ws.PubSub
	friendGetter  FriendIDsGetter
}

// NewOnlineService 创建 OnlineService 实例
// 启动时清理 Redis 中残留的在线状态数据，防止服务重启后出现"幽灵在线"
func NewOnlineService(rdb *redis.Client, hub *ws.Hub, pubsub *ws.PubSub, friendGetter FriendIDsGetter) *OnlineService {
	svc := &OnlineService{
		rdb:          rdb,
		hub:          hub,
		pubsub:       pubsub,
		friendGetter: friendGetter,
	}
	svc.cleanStaleOnlineData()
	return svc
}

// cleanStaleOnlineData 服务启动时清理上次残留的在线状态
// 单实例部署下，启动时不可能有合法的在线用户，直接清空即可
func (s *OnlineService) cleanStaleOnlineData() {
	ctx := context.Background()
	funcName := "ws.online_service.cleanStaleOnlineData"

	members, err := s.rdb.SMembers(ctx, onlineSetKey).Result()
	if err != nil {
		logs.Warn(ctx, funcName, "读取旧在线集合失败", zap.Error(err))
		return
	}
	if len(members) == 0 {
		return
	}

	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, onlineSetKey)
	for _, m := range members {
		pipe.Del(ctx, statusKeyPrefix+m)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		logs.Error(ctx, funcName, "清理旧在线数据失败", zap.Error(err))
	} else {
		logs.Info(ctx, funcName, "已清理残留在线状态",
			zap.Int("stale_count", len(members)))
	}
}

// UserOnline 用户上线：写入 Redis + 通知在线好友
func (s *OnlineService) UserOnline(ctx context.Context, userID int64, ip string) {
	funcName := "ws.online_service.UserOnline"

	pipe := s.rdb.Pipeline()
	pipe.SAdd(ctx, onlineSetKey, userID)

	status := &UserStatus{
		UserID:    userID,
		ConnectAt: time.Now().Format("2006-01-02 15:04:05"),
		IP:        ip,
	}
	statusJSON, err := json.Marshal(status)
	if err != nil {
		logs.Error(ctx, funcName, "序列化用户状态失败", zap.Int64("user_id", userID), zap.Error(err))
		return
	}
	pipe.Set(ctx, statusKey(userID), statusJSON, statusTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		logs.Error(ctx, funcName, "写入在线状态失败",
			zap.Int64("user_id", userID), zap.Error(err))
		return
	}

	logs.Info(ctx, funcName, "用户上线", zap.Int64("user_id", userID))

	if s.friendGetter != nil {
		friendIDs, err := s.friendGetter.GetFriendIDs(ctx, userID)
		if err != nil {
			logs.Warn(ctx, funcName, "获取好友列表失败，跳过上线通知", zap.Error(err))
		} else if len(friendIDs) > 0 {
			s.NotifyFriendsStatusChange(ctx, userID, true, friendIDs)
		}
	}
}

// UserOffline 用户下线：清除 Redis + 通知在线好友
func (s *OnlineService) UserOffline(ctx context.Context, userID int64) {
	funcName := "ws.online_service.UserOffline"

	pipe := s.rdb.Pipeline()
	pipe.SRem(ctx, onlineSetKey, userID)
	pipe.Del(ctx, statusKey(userID))

	if _, err := pipe.Exec(ctx); err != nil {
		logs.Error(ctx, funcName, "清除在线状态失败",
			zap.Int64("user_id", userID), zap.Error(err))
	}

	logs.Info(ctx, funcName, "用户下线", zap.Int64("user_id", userID))

	if s.friendGetter != nil {
		friendIDs, err := s.friendGetter.GetFriendIDs(ctx, userID)
		if err != nil {
			logs.Warn(ctx, funcName, "获取好友列表失败，跳过下线通知", zap.Error(err))
		} else if len(friendIDs) > 0 {
			s.NotifyFriendsStatusChange(ctx, userID, false, friendIDs)
		}
	}
}

// HeartbeatRenew 心跳续期：延长状态 TTL
func (s *OnlineService) HeartbeatRenew(ctx context.Context, userID int64) {
	if err := s.rdb.Expire(ctx, statusKey(userID), statusTTL).Err(); err != nil {
		logs.Warn(ctx, "ws.online_service.HeartbeatRenew", "心跳续期失败",
			zap.Int64("user_id", userID), zap.Error(err))
	}
}

// IsOnline 检查用户是否在线（Redis 查询）
func (s *OnlineService) IsOnline(ctx context.Context, userID int64) bool {
	ok, err := s.rdb.SIsMember(ctx, onlineSetKey, userID).Result()
	if err != nil {
		return false
	}
	return ok
}

// GetOnlineUserIDs 获取所有在线用户 ID
func (s *OnlineService) GetOnlineUserIDs(ctx context.Context) ([]int64, error) {
	members, err := s.rdb.SMembers(ctx, onlineSetKey).Result()
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(members))
	for _, m := range members {
		var id int64
		if _, err := fmt.Sscanf(m, "%d", &id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// GetOnlineCount 获取在线用户总数
func (s *OnlineService) GetOnlineCount(ctx context.Context) (int64, error) {
	return s.rdb.SCard(ctx, onlineSetKey).Result()
}

// BatchCheckOnline 批量检查用户在线状态
func (s *OnlineService) BatchCheckOnline(ctx context.Context, userIDs []int64) map[int64]bool {
	result := make(map[int64]bool, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}

	pipe := s.rdb.Pipeline()
	cmds := make(map[int64]*redis.BoolCmd, len(userIDs))
	for _, uid := range userIDs {
		cmds[uid] = pipe.SIsMember(ctx, onlineSetKey, uid)
	}
	pipe.Exec(ctx)

	for uid, cmd := range cmds {
		result[uid], _ = cmd.Result()
	}
	return result
}

// NotifyFriendsStatusChange 通知好友状态变更（通过 PubSub 推送）
func (s *OnlineService) NotifyFriendsStatusChange(ctx context.Context, userID int64, online bool, friendIDs []int64) {
	funcName := "ws.online_service.NotifyFriendsStatusChange"

	event := "user.status.offline"
	if online {
		event = "user.status.online"
	}

	push := ws.NewPushMessage(event, map[string]interface{}{
		"user_id": userID,
	})

	for _, fid := range friendIDs {
		if err := s.pubsub.PublishToUser(ctx, fid, push); err != nil {
			logs.Warn(ctx, funcName, "推送状态变更失败",
				zap.Int64("target_user", fid), zap.Error(err))
		}
	}
}

func statusKey(userID int64) string {
	return fmt.Sprintf("%s%d", statusKeyPrefix, userID)
}
