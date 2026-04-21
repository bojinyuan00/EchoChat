package service

import (
	"context"
	"errors"
	"time"

	authModel "github.com/echochat/backend/app/auth/model"
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/notify/dao"
	"github.com/echochat/backend/app/notify/model"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNotificationNotFound = errors.New("通知不存在")
)

// NotifyService 通知业务服务
// 负责：
//  1. 面向用户的列表 / 未读数 / 已读操作（REST API 层调用）
//  2. 面向上游业务的 Pusher 接口实现（持久化 + WS 推送）
//  3. 断线重连时的未读补偿推送
type NotifyService struct {
	notifyDAO    *dao.NotificationDAO
	pubsub       *ws.PubSub
	userResolver UserInfoResolver
}

// NewNotifyService 创建 NotifyService 实例
func NewNotifyService(
	notifyDAO *dao.NotificationDAO,
	pubsub *ws.PubSub,
	userResolver UserInfoResolver,
) *NotifyService {
	return &NotifyService{
		notifyDAO:    notifyDAO,
		pubsub:       pubsub,
		userResolver: userResolver,
	}
}

// ====== 面向用户的查询与操作 ======

// GetList 分页查询通知列表
func (s *NotifyService) GetList(ctx context.Context, userID int64, req *dto.GetNotificationsRequest) (*dto.NotificationListResponse, error) {
	funcName := "service.notify_service.GetList"

	types := typesByCategory(req.Category)
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	list, err := s.notifyDAO.List(ctx, &dao.ListRequest{
		UserID:   userID,
		Types:    types,
		IsRead:   req.IsRead,
		BeforeID: req.BeforeID,
		Limit:    limit + 1, // 多查一条判断是否还有下一页
	})
	if err != nil {
		return nil, err
	}

	hasMore := false
	if len(list) > limit {
		hasMore = true
		list = list[:limit]
	}

	items := s.toNotificationDTOs(ctx, list)
	logs.Debug(ctx, funcName, "查询通知列表",
		zap.Int64("user_id", userID), zap.Int("count", len(items)), zap.Bool("has_more", hasMore))

	return &dto.NotificationListResponse{
		List:    items,
		HasMore: hasMore,
	}, nil
}

// GetUnreadCount 查询未读数（总数 + 按分类）
func (s *NotifyService) GetUnreadCount(ctx context.Context, userID int64) (*dto.UnreadCountResponse, error) {
	countMap, err := s.notifyDAO.CountUnreadByType(ctx, userID)
	if err != nil {
		return nil, err
	}
	total := 0
	byCat := map[string]int{
		constants.NotifyCategoryFriend:  0,
		constants.NotifyCategoryGroup:   0,
		constants.NotifyCategoryMeeting: 0,
		constants.NotifyCategorySystem:  0,
	}
	for t, c := range countMap {
		total += int(c)
		byCat[constants.NotifyCategoryOfType(t)] += int(c)
	}
	return &dto.UnreadCountResponse{
		Total:      total,
		ByCategory: byCat,
	}, nil
}

// MarkRead 标记单条通知为已读
func (s *NotifyService) MarkRead(ctx context.Context, userID, id int64) error {
	affected, err := s.notifyDAO.MarkRead(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotificationNotFound
		}
		return err
	}
	if affected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

// MarkAllRead 将用户某分类（或全部）通知标记为已读
// category 为空或 all 时清空全部未读
func (s *NotifyService) MarkAllRead(ctx context.Context, userID int64, category string) (int64, error) {
	types := typesByCategory(category)
	return s.notifyDAO.MarkAllRead(ctx, userID, types)
}

// PushUnreadTotalOnConnect 向刚建立连接的用户推送未读总数补偿
// 由 WS handler 的 OnConnected 钩子调用（通过 NotifyConnectHook 接口适配）
func (s *NotifyService) PushUnreadTotalOnConnect(ctx context.Context, userID int64) {
	funcName := "service.notify_service.PushUnreadTotalOnConnect"
	resp, err := s.GetUnreadCount(ctx, userID)
	if err != nil {
		logs.Warn(ctx, funcName, "查询未读数失败", zap.Int64("user_id", userID), zap.Error(err))
		return
	}
	if resp.Total == 0 {
		return
	}
	s.publishWS(ctx, userID, constants.WSEventNotifyUnreadTotal, map[string]interface{}{
		"total":       resp.Total,
		"by_category": resp.ByCategory,
	})
}

// ====== Pusher 接口实现 ======

// Push 写入通知并推送 WS（单用户）
// 实现策略：
//  1. 同步入库（保证持久化）
//  2. 异步 WS 推送（避免阻塞上游业务事务）
//  3. 任一步失败不回滚另一步，全部仅记录 Warn 日志
func (s *NotifyService) Push(ctx context.Context, payload *PushPayload) {
	funcName := "service.notify_service.Push"
	if payload == nil || payload.UserID <= 0 || payload.Type == "" {
		logs.Warn(ctx, funcName, "无效的推送载荷", zap.Any("payload", payload))
		return
	}

	n := s.buildModel(payload)
	if err := s.notifyDAO.Create(ctx, n); err != nil {
		logs.Warn(ctx, funcName, "通知入库失败，放弃 WS 推送",
			zap.Int64("user_id", payload.UserID), zap.String("type", payload.Type), zap.Error(err))
		return
	}

	go s.safePushWS(n, payload)
}

// PushBatch 批量写入通知并推送 WS
// 适用于管理员广播 / 群操作批量通知场景
// 优化：一次 BatchCreate 入库，然后分别 WS 推送
func (s *NotifyService) PushBatch(ctx context.Context, payloads []*PushPayload) {
	funcName := "service.notify_service.PushBatch"
	if len(payloads) == 0 {
		return
	}

	models := make([]*model.Notification, 0, len(payloads))
	valid := make([]*PushPayload, 0, len(payloads))
	for _, p := range payloads {
		if p == nil || p.UserID <= 0 || p.Type == "" {
			continue
		}
		models = append(models, s.buildModel(p))
		valid = append(valid, p)
	}

	if err := s.notifyDAO.BatchCreate(ctx, models); err != nil {
		logs.Warn(ctx, funcName, "通知批量入库失败，放弃 WS 推送",
			zap.Int("count", len(models)), zap.Error(err))
		return
	}

	// 入库成功后逐条异步推送
	for i, n := range models {
		go s.safePushWS(n, valid[i])
	}
}

// ====== 内部辅助 ======

// buildModel 由 PushPayload 构造 Notification 持久化模型
// 空 Title 使用类型默认文案；空 TargetType 按业务类型映射为默认值
func (s *NotifyService) buildModel(p *PushPayload) *model.Notification {
	title := p.Title
	if title == "" {
		if zh, ok := constants.NotifyTypeMap[p.Type]; ok {
			title = zh
		}
	}
	targetType := p.TargetType
	if targetType == "" {
		targetType = defaultTargetType(p.Type)
	}
	return &model.Notification{
		UserID:     p.UserID,
		Type:       p.Type,
		Title:      title,
		Content:    p.Content,
		Extra:      marshalExtra(p.Extra),
		ActorID:    p.ActorID,
		TargetType: targetType,
		TargetID:   p.TargetID,
		IsRead:     false,
	}
}

// safePushWS 序列化并通过 Redis Pub/Sub 推送 notify.new 事件
// 失败仅记录 Warn，不影响持久化的通知记录
func (s *NotifyService) safePushWS(n *model.Notification, p *PushPayload) {
	defer func() {
		if r := recover(); r != nil {
			logs.Warn(context.Background(), "service.notify_service.safePushWS",
				"推送协程 panic", zap.Any("panic", r))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dtoItem := s.toNotificationDTO(ctx, n)
	s.publishWS(ctx, n.UserID, constants.WSEventNotifyNew, dtoItem)
}

// publishWS 序列化推送消息并发布到用户频道
func (s *NotifyService) publishWS(ctx context.Context, userID int64, event string, data interface{}) {
	if s.pubsub == nil {
		return
	}
	push := ws.NewPushMessage(event, data)
	bytes, err := ws.MarshalPush(push)
	if err != nil {
		logs.Warn(ctx, "service.notify_service.publishWS", "序列化推送失败",
			zap.String("event", event), zap.Error(err))
		return
	}
	if err := s.pubsub.Publish(ctx, userID, bytes); err != nil {
		logs.Warn(ctx, "service.notify_service.publishWS", "WS 推送失败",
			zap.Int64("user_id", userID), zap.String("event", event), zap.Error(err))
	}
}

// toNotificationDTO 将 model 转为对外 DTO，补全 actor 信息
func (s *NotifyService) toNotificationDTO(ctx context.Context, n *model.Notification) dto.NotificationDTO {
	item := dto.NotificationDTO{
		ID:         n.ID,
		Type:       n.Type,
		Category:   constants.NotifyCategoryOfType(n.Type),
		Title:      n.Title,
		Content:    n.Content,
		Extra:      n.Extra,
		ActorID:    n.ActorID,
		TargetType: n.TargetType,
		TargetID:   n.TargetID,
		IsRead:     n.IsRead,
		CreatedAt:  n.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if n.ReadAt != nil {
		item.ReadAt = n.ReadAt.Format("2006-01-02 15:04:05")
	}

	if n.ActorID != nil && *n.ActorID > 0 && s.userResolver != nil {
		users, err := s.userResolver.GetUsersByIDs(ctx, []int64{*n.ActorID})
		if err == nil && len(users) > 0 {
			item.ActorName = users[0].Nickname
			item.ActorAvatar = users[0].Avatar
		}
	}
	return item
}

// toNotificationDTOs 批量转换 DTO 并一次性补全 actor 信息，避免 N+1 查询
func (s *NotifyService) toNotificationDTOs(ctx context.Context, list []model.Notification) []dto.NotificationDTO {
	items := make([]dto.NotificationDTO, 0, len(list))
	if len(list) == 0 {
		return items
	}

	actorSet := make(map[int64]struct{})
	for i := range list {
		if list[i].ActorID != nil && *list[i].ActorID > 0 {
			actorSet[*list[i].ActorID] = struct{}{}
		}
	}
	userMap := make(map[int64]*authModel.User)
	if len(actorSet) > 0 && s.userResolver != nil {
		ids := make([]int64, 0, len(actorSet))
		for id := range actorSet {
			ids = append(ids, id)
		}
		users, err := s.userResolver.GetUsersByIDs(ctx, ids)
		if err == nil {
			for i := range users {
				userMap[users[i].ID] = &users[i]
			}
		}
	}

	for i := range list {
		n := &list[i]
		item := dto.NotificationDTO{
			ID:         n.ID,
			Type:       n.Type,
			Category:   constants.NotifyCategoryOfType(n.Type),
			Title:      n.Title,
			Content:    n.Content,
			Extra:      n.Extra,
			ActorID:    n.ActorID,
			TargetType: n.TargetType,
			TargetID:   n.TargetID,
			IsRead:     n.IsRead,
			CreatedAt:  n.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if n.ReadAt != nil {
			item.ReadAt = n.ReadAt.Format("2006-01-02 15:04:05")
		}
		if n.ActorID != nil {
			if u, ok := userMap[*n.ActorID]; ok {
				item.ActorName = u.Nickname
				item.ActorAvatar = u.Avatar
			}
		}
		items = append(items, item)
	}
	return items
}

// typesByCategory 把前端的 category 参数转换为 type 列表
// all 或空字符串返回 nil（不按类型过滤）
func typesByCategory(category string) []string {
	if category == "" || category == constants.NotifyCategoryAll {
		return nil
	}
	if list, ok := constants.NotifyTypesByCategory[category]; ok {
		return list
	}
	return nil
}

// defaultTargetType 根据通知类型推导默认业务对象类型
// 用于 PushPayload.TargetType 为空时的兜底
func defaultTargetType(notifyType string) string {
	switch constants.NotifyCategoryOfType(notifyType) {
	case constants.NotifyCategoryFriend:
		return constants.NotifyTargetUser
	case constants.NotifyCategoryGroup:
		return constants.NotifyTargetGroup
	case constants.NotifyCategoryMeeting:
		return constants.NotifyTargetMeeting
	default:
		return constants.NotifyTargetSystem
	}
}

// ====== 管理员广播 ======

// Broadcast 向所有正常用户发送系统公告
// 查询活跃用户 ID → 构造 payload → 批量入库 + 推送
// 返回受影响用户数
func (s *NotifyService) Broadcast(ctx context.Context, operatorID int64, req *dto.BroadcastNotificationRequest) (int, error) {
	funcName := "service.notify_service.Broadcast"
	logs.Info(ctx, funcName, "管理员广播",
		zap.Int64("operator_id", operatorID), zap.String("title", req.Title))

	userIDs, err := s.notifyDAO.ListAllActiveUserIDs(ctx)
	if err != nil {
		return 0, err
	}
	if len(userIDs) == 0 {
		return 0, nil
	}

	targetType := req.TargetType
	if targetType == "" {
		targetType = constants.NotifyTargetSystem
	}

	payloads := make([]*PushPayload, 0, len(userIDs))
	for _, uid := range userIDs {
		payloads = append(payloads, &PushPayload{
			UserID:     uid,
			Type:       constants.NotifyTypeSystemBroadcast,
			Title:      req.Title,
			Content:    req.Content,
			ActorID:    &operatorID,
			TargetType: targetType,
			TargetID:   req.TargetID,
			Extra:      req.Extra,
		})
	}
	s.PushBatch(ctx, payloads)
	return len(payloads), nil
}
