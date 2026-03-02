package ws

import (
	"context"
	"fmt"
	"sync"

	"github.com/echochat/backend/pkg/logs"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const channelPrefix = "echo:ws:channel:"

// PubSub 封装 Redis Pub/Sub，提供按用户频道的消息发布和订阅
type PubSub struct {
	rdb         *redis.Client
	hub         *Hub
	subs        map[int64]context.CancelFunc // userID -> 订阅取消函数
	mu          sync.Mutex
}

// NewPubSub 创建 PubSub 实例
func NewPubSub(rdb *redis.Client, hub *Hub) *PubSub {
	return &PubSub{
		rdb:  rdb,
		hub:  hub,
		subs: make(map[int64]context.CancelFunc),
	}
}

// channelName 生成用户专属的 Redis 频道名
func channelName(userID int64) string {
	return fmt.Sprintf("%s%d", channelPrefix, userID)
}

// Publish 向指定用户的频道发布消息
func (ps *PubSub) Publish(ctx context.Context, userID int64, data []byte) error {
	channel := channelName(userID)
	return ps.rdb.Publish(ctx, channel, data).Err()
}

// Subscribe 订阅指定用户的频道
// 收到消息后自动转发给本地 Hub 中对应的 Client
func (ps *PubSub) Subscribe(userID int64) {
	ps.mu.Lock()
	if cancel, ok := ps.subs[userID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ps.subs[userID] = cancel
	ps.mu.Unlock()

	channel := channelName(userID)
	sub := ps.rdb.Subscribe(ctx, channel)

	go func() {
		defer sub.Close()
		ch := sub.Channel()

		logs.Info(nil, "ws.pubsub.Subscribe", "开始订阅用户频道",
			zap.Int64("user_id", userID), zap.String("channel", channel))

		for {
			select {
			case <-ctx.Done():
				logs.Info(nil, "ws.pubsub.Subscribe", "取消订阅用户频道",
					zap.Int64("user_id", userID))
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				ps.hub.SendToUser(userID, []byte(msg.Payload))
			}
		}
	}()
}

// Unsubscribe 取消订阅指定用户的频道
func (ps *PubSub) Unsubscribe(userID int64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if cancel, ok := ps.subs[userID]; ok {
		cancel()
		delete(ps.subs, userID)
	}
}

// PublishToUser 便捷方法：序列化推送消息并发布到用户频道
func (ps *PubSub) PublishToUser(ctx context.Context, userID int64, msg *PushMessage) error {
	data, err := MarshalPush(msg)
	if err != nil {
		return fmt.Errorf("序列化推送消息失败: %w", err)
	}
	return ps.Publish(ctx, userID, data)
}
