package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/im/dao"
	"github.com/echochat/backend/app/im/model"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	unreadKeyPrefix = "echo:im:unread:" // Redis 未读总数 key 前缀，完整 key: echo:im:unread:{user_id}
	defaultPageSize = 30                // 历史消息默认拉取条数
	maxPageSize     = 100               // 历史消息最大拉取条数
)

var (
	ErrNotFriend       = errors.New("对方不是你的好友")
	ErrEmptyContent    = errors.New("消息内容不能为空")
	ErrConvNotFound    = errors.New("会话不存在")
	ErrMsgNotFound     = errors.New("消息不存在")
	ErrNotSender       = errors.New("只能撤回自己发送的消息")
	ErrRecallTimeout   = errors.New("超过撤回时限")
	ErrNotMember       = errors.New("你不是该会话的成员")
	ErrDuplicateMsg    = errors.New("重复消息")
	ErrInvalidMsgType  = errors.New("不支持的消息类型")
)

// IMService 即时通讯核心业务服务
type IMService struct {
	convDAO       *dao.ConversationDAO
	msgDAO        *dao.MessageDAO
	pubsub        *ws.PubSub
	rdb           *redis.Client
	friendChecker FriendChecker
	userInfoGetter UserInfoGetter
}

// NewIMService 创建 IMService 实例
func NewIMService(
	convDAO *dao.ConversationDAO,
	msgDAO *dao.MessageDAO,
	pubsub *ws.PubSub,
	rdb *redis.Client,
	friendChecker FriendChecker,
	userInfoGetter UserInfoGetter,
) *IMService {
	return &IMService{
		convDAO:        convDAO,
		msgDAO:         msgDAO,
		pubsub:         pubsub,
		rdb:            rdb,
		friendChecker:  friendChecker,
		userInfoGetter: userInfoGetter,
	}
}

// SendMessage 发送消息（核心流程）
// 1. 校验好友关系
// 2. 查找或创建会话
// 3. 幂等去重（client_msg_id）
// 4. 写入消息 + 更新会话最后消息 + 递增对方未读数
// 5. 通过 PubSub 推送给接收方
func (s *IMService) SendMessage(ctx context.Context, senderID int64, req *dto.SendMessageRequest) (*dto.MessageDTO, error) {
	funcName := "service.im_service.SendMessage"
	logs.Info(ctx, funcName, "发送消息",
		zap.Int64("sender_id", senderID),
		zap.Int64("conversation_id", req.ConversationID),
		zap.Int64("target_user_id", req.TargetUserID))

	if req.Content == "" {
		return nil, ErrEmptyContent
	}
	if req.Type == 0 {
		req.Type = constants.MessageTypeText
	}
	if req.Type != constants.MessageTypeText {
		return nil, ErrInvalidMsgType
	}

	convID := req.ConversationID
	var peerID int64

	if convID == 0 && req.TargetUserID > 0 {
		isFriend, err := s.friendChecker.IsFriend(ctx, senderID, req.TargetUserID)
		if err != nil {
			logs.Error(ctx, funcName, "检查好友关系失败", zap.Error(err))
			return nil, err
		}
		if !isFriend {
			return nil, ErrNotFriend
		}

		conv, err := s.getOrCreatePrivateConversation(ctx, senderID, req.TargetUserID)
		if err != nil {
			return nil, err
		}
		convID = conv.ID
		peerID = req.TargetUserID
	} else if convID > 0 {
		member, err := s.convDAO.GetMember(ctx, convID, senderID)
		if err != nil {
			return nil, ErrNotMember
		}
		if member == nil {
			return nil, ErrNotMember
		}
		peerID, err = s.convDAO.GetPeerUserID(ctx, convID, senderID)
		if err != nil {
			logs.Error(ctx, funcName, "查询对方用户 ID 失败", zap.Error(err))
			return nil, err
		}
	} else {
		return nil, ErrConvNotFound
	}

	if req.ClientMsgID != "" {
		existing, err := s.msgDAO.FindByClientMsgID(ctx, convID, req.ClientMsgID)
		if err != nil {
			logs.Error(ctx, funcName, "幂等去重查询失败", zap.Error(err))
			return nil, err
		}
		if existing != nil {
			return s.toMessageDTO(existing), ErrDuplicateMsg
		}
	}

	msg := &model.Message{
		ConversationID: convID,
		SenderID:       senderID,
		Type:           req.Type,
		Content:        req.Content,
		Status:         constants.MessageStatusNormal,
		ClientMsgID:    req.ClientMsgID,
	}
	if err := s.msgDAO.Create(ctx, msg); err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.convDAO.UpdateLastMessage(ctx, convID, msg.ID, truncateContent(req.Content, 100), senderID, now); err != nil {
		logs.Error(ctx, funcName, "更新最后消息失败", zap.Error(err))
	}

	if err := s.convDAO.RestoreMember(ctx, convID, peerID); err != nil {
		logs.Error(ctx, funcName, "恢复对方会话视图失败", zap.Error(err))
	}

	if err := s.convDAO.IncrementUnread(ctx, convID, peerID); err != nil {
		logs.Error(ctx, funcName, "递增未读计数失败", zap.Error(err))
	}

	s.incrementTotalUnread(ctx, peerID)

	s.pushToUser(ctx, peerID, "im.message.new", map[string]interface{}{
		"id":              msg.ID,
		"conversation_id": convID,
		"sender_id":       senderID,
		"type":            msg.Type,
		"content":         msg.Content,
		"client_msg_id":   msg.ClientMsgID,
		"created_at":      msg.CreatedAt.Format("2006-01-02 15:04:05"),
	})

	return s.toMessageDTO(msg), nil
}

// RecallMessage 撤回消息（2分钟内）
func (s *IMService) RecallMessage(ctx context.Context, senderID int64, messageID int64) error {
	funcName := "service.im_service.RecallMessage"
	logs.Info(ctx, funcName, "撤回消息",
		zap.Int64("sender_id", senderID), zap.Int64("message_id", messageID))

	msg, err := s.msgDAO.GetByID(ctx, messageID)
	if err != nil {
		return ErrMsgNotFound
	}
	if msg.SenderID != senderID {
		return ErrNotSender
	}
	if time.Since(msg.CreatedAt).Seconds() > float64(constants.MessageRecallTimeLimit) {
		return ErrRecallTimeout
	}

	if err := s.msgDAO.UpdateStatus(ctx, messageID, constants.MessageStatusRecalled); err != nil {
		logs.Error(ctx, funcName, "更新消息状态失败", zap.Error(err))
		return err
	}

	memberIDs, err := s.convDAO.GetConversationMemberIDs(ctx, msg.ConversationID)
	if err != nil {
		logs.Error(ctx, funcName, "获取会话成员失败", zap.Error(err))
		return nil
	}
	for _, uid := range memberIDs {
		if uid == senderID {
			continue
		}
		s.pushToUser(ctx, uid, "im.message.recalled", map[string]interface{}{
			"message_id":      messageID,
			"conversation_id": msg.ConversationID,
		})
	}

	return nil
}

// GetConversationList 获取会话列表（含对方用户信息）
func (s *IMService) GetConversationList(ctx context.Context, userID int64) (*dto.ConversationListResponse, error) {
	funcName := "service.im_service.GetConversationList"
	logs.Debug(ctx, funcName, "获取会话列表", zap.Int64("user_id", userID))

	convs, err := s.convDAO.GetUserConversations(ctx, userID)
	if err != nil {
		return nil, err
	}

	peerIDs := make([]int64, 0, len(convs))
	convIDToPeerID := make(map[int64]int64, len(convs))
	for _, c := range convs {
		peerID, err := s.convDAO.GetPeerUserID(ctx, c.ID, userID)
		if err != nil {
			logs.Error(ctx, funcName, "查询对方用户 ID 失败", zap.Int64("conv_id", c.ID), zap.Error(err))
			continue
		}
		peerIDs = append(peerIDs, peerID)
		convIDToPeerID[c.ID] = peerID
	}

	userMap := make(map[int64]*userBrief)
	if len(peerIDs) > 0 {
		users, err := s.userInfoGetter.GetUsersByIDs(ctx, peerIDs)
		if err != nil {
			logs.Error(ctx, funcName, "批量查询用户信息失败", zap.Error(err))
		} else {
			for i := range users {
				u := users[i]
				userMap[u.ID] = &userBrief{Nickname: u.Nickname, Avatar: u.Avatar}
			}
		}
	}

	list := make([]dto.ConversationDTO, 0, len(convs))
	for _, c := range convs {
		peerID := convIDToPeerID[c.ID]
		item := dto.ConversationDTO{
			ID:              c.ID,
			Type:            c.Type,
			PeerUserID:      peerID,
			LastMsgContent:  c.LastMsgContent,
			LastMsgSenderID: c.LastMsgSenderID,
			IsPinned:        c.IsPinned,
			UnreadCount:     c.UnreadCount,
		}
		if c.LastMsgTime != nil {
			item.LastMsgTime = c.LastMsgTime.Format("2006-01-02 15:04:05")
		}
		if brief, ok := userMap[peerID]; ok {
			item.PeerNickname = brief.Nickname
			item.PeerAvatar = brief.Avatar
		}
		list = append(list, item)
	}

	return &dto.ConversationListResponse{List: list}, nil
}

// GetHistoryMessages 获取历史消息（游标分页）
func (s *IMService) GetHistoryMessages(ctx context.Context, userID int64, req *dto.HistoryMessageRequest) (*dto.HistoryMessageResponse, error) {
	funcName := "service.im_service.GetHistoryMessages"
	logs.Debug(ctx, funcName, "查询历史消息",
		zap.Int64("user_id", userID), zap.Int64("conversation_id", req.ConversationID))

	member, err := s.convDAO.GetMember(ctx, req.ConversationID, userID)
	if err != nil || member == nil {
		return nil, ErrNotMember
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	messages, err := s.msgDAO.GetByConversation(ctx, req.ConversationID, req.BeforeID, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	list := make([]dto.MessageDTO, 0, len(messages))
	for _, m := range messages {
		list = append(list, *s.toMessageDTO(&m))
	}

	return &dto.HistoryMessageResponse{List: list, HasMore: hasMore}, nil
}

// MarkRead 标记会话已读（清零未读 + 更新 Redis 总未读数）
func (s *IMService) MarkRead(ctx context.Context, userID int64, conversationID int64) error {
	funcName := "service.im_service.MarkRead"
	logs.Info(ctx, funcName, "标记已读",
		zap.Int64("user_id", userID), zap.Int64("conversation_id", conversationID))

	member, err := s.convDAO.GetMember(ctx, conversationID, userID)
	if err != nil || member == nil {
		return ErrNotMember
	}

	if member.UnreadCount == 0 {
		return nil
	}

	latestMsgID, err := s.msgDAO.GetLatestMessageID(ctx, conversationID)
	if err != nil {
		logs.Error(ctx, funcName, "获取最新消息 ID 失败", zap.Error(err))
		return err
	}

	if err := s.convDAO.ClearUnread(ctx, conversationID, userID, latestMsgID); err != nil {
		logs.Error(ctx, funcName, "清零未读失败", zap.Error(err))
		return err
	}

	s.decrementTotalUnread(ctx, userID, member.UnreadCount)
	return nil
}

// PinConversation 置顶/取消置顶会话
func (s *IMService) PinConversation(ctx context.Context, userID int64, conversationID int64, isPinned bool) error {
	funcName := "service.im_service.PinConversation"
	logs.Info(ctx, funcName, "更新置顶状态",
		zap.Int64("user_id", userID), zap.Int64("conversation_id", conversationID), zap.Bool("is_pinned", isPinned))

	member, err := s.convDAO.GetMember(ctx, conversationID, userID)
	if err != nil || member == nil {
		return ErrNotMember
	}
	return s.convDAO.UpdateMemberPinned(ctx, conversationID, userID, isPinned)
}

// DeleteConversation 删除会话（软删除，仅影响当前用户视图）
func (s *IMService) DeleteConversation(ctx context.Context, userID int64, conversationID int64) error {
	funcName := "service.im_service.DeleteConversation"
	logs.Info(ctx, funcName, "删除会话",
		zap.Int64("user_id", userID), zap.Int64("conversation_id", conversationID))

	member, err := s.convDAO.GetMember(ctx, conversationID, userID)
	if err != nil || member == nil {
		return ErrNotMember
	}

	if member.UnreadCount > 0 {
		s.decrementTotalUnread(ctx, userID, member.UnreadCount)
	}
	return s.convDAO.SoftDeleteMember(ctx, conversationID, userID)
}

// ClearHistory 清空聊天记录（标记消息为已删除）
func (s *IMService) ClearHistory(ctx context.Context, userID int64, conversationID int64) error {
	funcName := "service.im_service.ClearHistory"
	logs.Info(ctx, funcName, "清空聊天记录",
		zap.Int64("user_id", userID), zap.Int64("conversation_id", conversationID))

	member, err := s.convDAO.GetMember(ctx, conversationID, userID)
	if err != nil || member == nil {
		return ErrNotMember
	}
	return s.msgDAO.DeleteByConversation(ctx, conversationID)
}

// SearchMessages 全局消息搜索
func (s *IMService) SearchMessages(ctx context.Context, userID int64, req *dto.SearchMessageRequest) (*dto.SearchMessageResponse, error) {
	funcName := "service.im_service.SearchMessages"
	logs.Debug(ctx, funcName, "全局消息搜索",
		zap.Int64("user_id", userID), zap.String("keyword", req.Keyword))

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}

	results, err := s.msgDAO.SearchMessages(ctx, userID, req.Keyword, limit)
	if err != nil {
		return nil, err
	}

	senderIDs := make([]int64, 0)
	senderSet := make(map[int64]bool)
	for _, r := range results {
		if !senderSet[r.SenderID] {
			senderIDs = append(senderIDs, r.SenderID)
			senderSet[r.SenderID] = true
		}
	}

	userMap := make(map[int64]*userBrief)
	if len(senderIDs) > 0 {
		users, err := s.userInfoGetter.GetUsersByIDs(ctx, senderIDs)
		if err != nil {
			logs.Error(ctx, funcName, "查询发送者信息失败", zap.Error(err))
		} else {
			for i := range users {
				u := users[i]
				userMap[u.ID] = &userBrief{Nickname: u.Nickname, Avatar: u.Avatar}
			}
		}
	}

	list := make([]dto.MessageSearchItem, 0, len(results))
	for _, r := range results {
		item := dto.MessageSearchItem{
			MessageDTO: dto.MessageDTO{
				ID:             r.ID,
				ConversationID: r.ConversationID,
				SenderID:       r.SenderID,
				Type:           constants.MessageTypeText,
				Content:        r.Content,
				Status:         constants.MessageStatusNormal,
				CreatedAt:      r.CreatedAt,
			},
		}
		if brief, ok := userMap[r.SenderID]; ok {
			item.SenderNickname = brief.Nickname
			item.SenderAvatar = brief.Avatar
		}
		list = append(list, item)
	}

	return &dto.SearchMessageResponse{List: list}, nil
}

// GetTotalUnread 获取用户的全局未读消息总数（从 Redis）
func (s *IMService) GetTotalUnread(ctx context.Context, userID int64) (int64, error) {
	key := fmt.Sprintf("%s%d", unreadKeyPrefix, userID)
	val, err := s.rdb.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// GetPeerUserID 获取单聊会话中对方的用户 ID（供 handler 层调用）
func (s *IMService) GetPeerUserID(ctx context.Context, conversationID, userID int64) (int64, error) {
	return s.convDAO.GetPeerUserID(ctx, conversationID, userID)
}

// ====== 内部辅助方法 ======

// getOrCreatePrivateConversation 查找或创建单聊会话
func (s *IMService) getOrCreatePrivateConversation(ctx context.Context, userID, targetUserID int64) (*model.Conversation, error) {
	funcName := "service.im_service.getOrCreatePrivateConversation"

	conv, err := s.convDAO.FindPrivateConversation(ctx, userID, targetUserID)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	logs.Info(ctx, funcName, "创建新的单聊会话",
		zap.Int64("user_id", userID), zap.Int64("target_user_id", targetUserID))

	newConv := &model.Conversation{
		Type:      constants.ConversationTypePrivate,
		CreatorID: userID,
	}
	if err := s.convDAO.CreateWithMembers(ctx, newConv, []int64{userID, targetUserID}); err != nil {
		return nil, err
	}
	return newConv, nil
}

// pushToUser 通过 PubSub 向指定用户推送消息
func (s *IMService) pushToUser(ctx context.Context, userID int64, event string, data interface{}) {
	push := ws.NewPushMessage(event, data)
	bytes, err := ws.MarshalPush(push)
	if err != nil {
		logs.Error(ctx, "service.im_service.pushToUser", "序列化推送消息失败",
			zap.String("event", event), zap.Error(err))
		return
	}
	if err := s.pubsub.Publish(ctx, userID, bytes); err != nil {
		logs.Error(ctx, "service.im_service.pushToUser", "PubSub 发布失败",
			zap.Int64("user_id", userID), zap.Error(err))
	}
}

// incrementTotalUnread Redis 全局未读 +1
func (s *IMService) incrementTotalUnread(ctx context.Context, userID int64) {
	key := fmt.Sprintf("%s%d", unreadKeyPrefix, userID)
	if err := s.rdb.Incr(ctx, key).Err(); err != nil {
		logs.Error(ctx, "service.im_service.incrementTotalUnread", "Redis INCR 失败",
			zap.Int64("user_id", userID), zap.Error(err))
	}
}

// decrementTotalUnread Redis 全局未读 -N（下限为 0）
func (s *IMService) decrementTotalUnread(ctx context.Context, userID int64, count int) {
	key := fmt.Sprintf("%s%d", unreadKeyPrefix, userID)
	if err := s.rdb.DecrBy(ctx, key, int64(count)).Err(); err != nil {
		logs.Error(ctx, "service.im_service.decrementTotalUnread", "Redis DECRBY 失败",
			zap.Int64("user_id", userID), zap.Error(err))
	}
}

// toMessageDTO 将 model.Message 转换为 dto.MessageDTO
func (s *IMService) toMessageDTO(m *model.Message) *dto.MessageDTO {
	return &dto.MessageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Type:           m.Type,
		Content:        m.Content,
		Status:         m.Status,
		ClientMsgID:    m.ClientMsgID,
		CreatedAt:      m.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// userBrief 用户简要信息（内部使用）
type userBrief struct {
	Nickname string
	Avatar   string
}

// truncateContent 截断消息内容用于预览
func truncateContent(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
