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
	ErrNotFriend        = errors.New("对方不是你的好友")
	ErrEmptyContent     = errors.New("消息内容不能为空")
	ErrConvNotFound     = errors.New("会话不存在")
	ErrMsgNotFound      = errors.New("消息不存在")
	ErrNotSender        = errors.New("只能撤回自己发送的消息")
	ErrRecallTimeout    = errors.New("超过撤回时限")
	ErrNotMember        = errors.New("你不是该会话的成员")
	ErrDuplicateMsg     = errors.New("重复消息")
	ErrInvalidMsgType   = errors.New("不支持的消息类型")
	ErrGroupDissolved   = errors.New("群聊已解散")
	ErrGroupAllMuted    = errors.New("当前群已开启全体禁言")
	ErrUserMuted        = errors.New("你已被禁言，无法发送消息")
)

// IMService 即时通讯核心业务服务
type IMService struct {
	convDAO        *dao.ConversationDAO
	msgDAO         *dao.MessageDAO
	pubsub         *ws.PubSub
	rdb            *redis.Client
	friendChecker  FriendChecker
	userInfoGetter UserInfoGetter
	groupInfo      GroupInfoGetter
	readRecorder   MessageReadRecorder
}

// NewIMService 创建 IMService 实例
func NewIMService(
	convDAO *dao.ConversationDAO,
	msgDAO *dao.MessageDAO,
	pubsub *ws.PubSub,
	rdb *redis.Client,
	friendChecker FriendChecker,
	userInfoGetter UserInfoGetter,
	groupInfo GroupInfoGetter,
	readRecorder MessageReadRecorder,
) *IMService {
	return &IMService{
		convDAO:        convDAO,
		msgDAO:         msgDAO,
		pubsub:         pubsub,
		rdb:            rdb,
		friendChecker:  friendChecker,
		userInfoGetter: userInfoGetter,
		groupInfo:      groupInfo,
		readRecorder:   readRecorder,
	}
}

// SendMessage 发送消息（核心流程，同时支持单聊和群聊）
// 单聊：校验好友关系 → 查找/创建会话 → 写入消息 → 推送给对方
// 群聊：校验群成员+禁言 → 写入消息（含 @信息）→ 推送给所有群成员
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

	if convID == 0 && req.TargetUserID > 0 {
		return s.sendPrivateMessage(ctx, senderID, req)
	} else if convID > 0 {
		conv, err := s.convDAO.GetByID(ctx, convID)
		if err != nil {
			return nil, ErrConvNotFound
		}
		if conv.Type == constants.ConversationTypeGroup {
			return s.sendGroupMessage(ctx, senderID, req)
		}
		return s.sendPrivateMessageByConvID(ctx, senderID, convID, req)
	}
	return nil, ErrConvNotFound
}

// sendPrivateMessage 发送单聊消息（首次发送，通过 TargetUserID）
func (s *IMService) sendPrivateMessage(ctx context.Context, senderID int64, req *dto.SendMessageRequest) (*dto.MessageDTO, error) {
	funcName := "service.im_service.sendPrivateMessage"

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

	return s.writeAndPushPrivateMessage(ctx, senderID, conv.ID, req.TargetUserID, req)
}

// sendPrivateMessageByConvID 发送单聊消息（已有会话 ID）
func (s *IMService) sendPrivateMessageByConvID(ctx context.Context, senderID, convID int64, req *dto.SendMessageRequest) (*dto.MessageDTO, error) {
	funcName := "service.im_service.sendPrivateMessageByConvID"

	member, err := s.convDAO.GetMember(ctx, convID, senderID)
	if err != nil || member == nil {
		return nil, ErrNotMember
	}

	peerID, err := s.convDAO.GetPeerUserID(ctx, convID, senderID)
	if err != nil {
		logs.Error(ctx, funcName, "查询对方用户 ID 失败", zap.Error(err))
		return nil, err
	}

	return s.writeAndPushPrivateMessage(ctx, senderID, convID, peerID, req)
}

// writeAndPushPrivateMessage 单聊：幂等去重 + 写消息 + 推送
func (s *IMService) writeAndPushPrivateMessage(ctx context.Context, senderID, convID, peerID int64, req *dto.SendMessageRequest) (*dto.MessageDTO, error) {
	funcName := "service.im_service.writeAndPushPrivateMessage"

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

	pushData := s.buildMessagePushData(ctx, msg, senderID)
	pushData["conv_type"] = constants.ConversationTypePrivate
	s.pushToUser(ctx, peerID, "im.message.new", pushData)

	return s.toMessageDTO(msg), nil
}

// sendGroupMessage 发送群聊消息
func (s *IMService) sendGroupMessage(ctx context.Context, senderID int64, req *dto.SendMessageRequest) (*dto.MessageDTO, error) {
	funcName := "service.im_service.sendGroupMessage"
	convID := req.ConversationID

	member, err := s.convDAO.GetMember(ctx, convID, senderID)
	if err != nil || member == nil {
		return nil, ErrNotMember
	}

	if s.groupInfo != nil {
		groupBrief, gErr := s.groupInfo.GetGroupBrief(ctx, convID)
		if gErr != nil {
			logs.Error(ctx, funcName, "获取群信息失败", zap.Error(gErr))
			return nil, ErrConvNotFound
		}
		if groupBrief.Status == constants.GroupStatusDissolved {
			return nil, ErrGroupDissolved
		}
		if groupBrief.IsAllMuted && member.Role < constants.GroupRoleAdmin {
			return nil, ErrGroupAllMuted
		}
	}

	if member.IsMuted {
		return nil, ErrUserMuted
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
	if len(req.AtUserIDs) > 0 {
		msg.AtUserIDs = req.AtUserIDs
	}
	if err := s.msgDAO.Create(ctx, msg); err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.convDAO.UpdateLastMessage(ctx, convID, msg.ID, truncateContent(req.Content, 100), senderID, now); err != nil {
		logs.Error(ctx, funcName, "更新最后消息失败", zap.Error(err))
	}

	memberIDs, err := s.convDAO.GetConversationMemberIDs(ctx, convID)
	if err != nil {
		logs.Error(ctx, funcName, "获取群成员列表失败", zap.Error(err))
		return s.toMessageDTO(msg), nil
	}

	dndMap, dndErr := s.convDAO.GetMemberDNDMap(ctx, convID)
	if dndErr != nil {
		logs.Error(ctx, funcName, "获取免打扰状态失败", zap.Error(dndErr))
		dndMap = make(map[int64]bool)
	}

	pushData := s.buildMessagePushData(ctx, msg, senderID)
	pushData["conv_type"] = constants.ConversationTypeGroup
	if len(req.AtUserIDs) > 0 {
		pushData["at_user_ids"] = req.AtUserIDs
	}

	for _, uid := range memberIDs {
		if uid == senderID {
			continue
		}
		if err := s.convDAO.IncrementUnread(ctx, convID, uid); err != nil {
			logs.Error(ctx, funcName, "递增未读计数失败", zap.Int64("user_id", uid), zap.Error(err))
		}

		if !dndMap[uid] {
			s.incrementTotalUnread(ctx, uid)
		}

		if len(req.AtUserIDs) > 0 {
			isAtMe := false
			for _, atID := range req.AtUserIDs {
				if atID == uid || atID == 0 {
					isAtMe = true
					break
				}
			}
			if isAtMe {
				if err := s.convDAO.IncrementAtMeCount(ctx, convID, uid); err != nil {
					logs.Error(ctx, funcName, "递增@计数失败", zap.Int64("user_id", uid), zap.Error(err))
				}
			}
		}

		s.pushToUser(ctx, uid, "im.message.new", pushData)
	}

	return s.toMessageDTO(msg), nil
}

// RecallMessage 撤回消息
// 单聊：只能撤回自己的消息，2 分钟内
// 群聊：自己的消息 2 分钟内撤回；群主/管理员可无时限撤回任何消息
func (s *IMService) RecallMessage(ctx context.Context, operatorID int64, messageID int64) error {
	funcName := "service.im_service.RecallMessage"
	logs.Info(ctx, funcName, "撤回消息",
		zap.Int64("operator_id", operatorID), zap.Int64("message_id", messageID))

	msg, err := s.msgDAO.GetByID(ctx, messageID)
	if err != nil {
		return ErrMsgNotFound
	}

	conv, err := s.convDAO.GetByID(ctx, msg.ConversationID)
	if err != nil {
		logs.Error(ctx, funcName, "获取会话信息失败", zap.Error(err))
		return ErrConvNotFound
	}

	isAdmin := false
	if conv.Type == constants.ConversationTypeGroup {
		member, mErr := s.convDAO.GetMember(ctx, msg.ConversationID, operatorID)
		if mErr != nil || member == nil {
			return ErrNotMember
		}
		isAdmin = member.Role >= constants.GroupRoleAdmin
	}

	if msg.SenderID == operatorID {
		if time.Since(msg.CreatedAt).Seconds() > float64(constants.MessageRecallTimeLimit) {
			if !isAdmin {
				return ErrRecallTimeout
			}
		}
	} else {
		if !isAdmin {
			return ErrNotSender
		}
	}

	if err := s.msgDAO.UpdateStatus(ctx, messageID, constants.MessageStatusRecalled); err != nil {
		logs.Error(ctx, funcName, "更新消息状态失败", zap.Error(err))
		return err
	}

	recallText := "撤回了一条消息"
	if msg.SenderID != operatorID && isAdmin {
		operatorInfo, infoErr := s.userInfoGetter.GetUsersByIDs(ctx, []int64{operatorID, msg.SenderID})
		operatorName := "管理员"
		senderName := "成员"
		if infoErr == nil {
			for _, u := range operatorInfo {
				if u.ID == operatorID {
					operatorName = u.Nickname
				}
				if u.ID == msg.SenderID {
					senderName = u.Nickname
				}
			}
		}
		recallText = fmt.Sprintf("管理员 %s 撤回了 %s 的一条消息", operatorName, senderName)
	} else {
		senderInfo, infoErr := s.userInfoGetter.GetUsersByIDs(ctx, []int64{operatorID})
		if infoErr == nil && len(senderInfo) > 0 {
			recallText = senderInfo[0].Nickname + " 撤回了一条消息"
		}
	}

	if conv.LastMessageID != nil && *conv.LastMessageID == msg.ID {
		if updateErr := s.convDAO.UpdateLastMessage(ctx, msg.ConversationID, msg.ID, recallText, operatorID, msg.CreatedAt); updateErr != nil {
			logs.Error(ctx, funcName, "更新会话预览失败", zap.Error(updateErr))
		}
	}

	memberIDs, err := s.convDAO.GetConversationMemberIDs(ctx, msg.ConversationID)
	if err != nil {
		logs.Error(ctx, funcName, "获取会话成员失败", zap.Error(err))
		return nil
	}
	for _, uid := range memberIDs {
		if uid == operatorID {
			continue
		}
		s.pushToUser(ctx, uid, "im.message.recalled", map[string]interface{}{
			"message_id":      messageID,
			"conversation_id": msg.ConversationID,
			"operator_id":     operatorID,
			"sender_id":       msg.SenderID,
			"recall_text":     recallText,
		})
	}

	return nil
}

// GetConversationList 获取会话列表（含对方用户信息 / 群聊信息）
func (s *IMService) GetConversationList(ctx context.Context, userID int64) (*dto.ConversationListResponse, error) {
	funcName := "service.im_service.GetConversationList"
	logs.Debug(ctx, funcName, "获取会话列表", zap.Int64("user_id", userID))

	convs, err := s.convDAO.GetUserConversations(ctx, userID)
	if err != nil {
		return nil, err
	}

	peerIDs := make([]int64, 0, len(convs))
	groupConvIDs := make([]int64, 0)
	for _, c := range convs {
		if c.Type == constants.ConversationTypePrivate && c.PeerUserID > 0 {
			peerIDs = append(peerIDs, c.PeerUserID)
		} else if c.Type == constants.ConversationTypeGroup {
			groupConvIDs = append(groupConvIDs, c.ID)
		}
	}

	userMap := make(map[int64]*userBrief)
	if len(peerIDs) > 0 {
		users, uErr := s.userInfoGetter.GetUsersByIDs(ctx, peerIDs)
		if uErr != nil {
			logs.Error(ctx, funcName, "批量查询用户信息失败", zap.Error(uErr))
		} else {
			for i := range users {
				u := users[i]
				userMap[u.ID] = &userBrief{Nickname: u.Nickname, Avatar: u.Avatar}
			}
		}
	}

	groupMap := make(map[int64]*dto.GroupBrief)
	if s.groupInfo != nil && len(groupConvIDs) > 0 {
		for _, convID := range groupConvIDs {
			brief, gErr := s.groupInfo.GetGroupBrief(ctx, convID)
			if gErr != nil {
				logs.Error(ctx, funcName, "获取群信息失败",
					zap.Int64("conversation_id", convID), zap.Error(gErr))
				continue
			}
			groupMap[convID] = brief
		}
	}

	list := make([]dto.ConversationDTO, 0, len(convs))
	for _, c := range convs {
		item := dto.ConversationDTO{
			ID:              c.ID,
			Type:            c.Type,
			PeerUserID:      c.PeerUserID,
			LastMsgContent:  c.LastMsgContent,
			LastMsgSenderID: c.LastMsgSenderID,
			IsPinned:        c.IsPinned,
			UnreadCount:     c.UnreadCount,
			IsDoNotDisturb:  c.IsDoNotDisturb,
			AtMeCount:       c.AtMeCount,
		}
		if c.ClearBeforeMsgID > 0 && c.LastMessageID != nil && *c.LastMessageID <= c.ClearBeforeMsgID {
			item.LastMsgContent = ""
			item.LastMsgSenderID = nil
		}
		if c.LastMsgTime != nil {
			item.LastMsgTime = c.LastMsgTime.Format("2006-01-02 15:04:05")
		}

		if c.Type == constants.ConversationTypePrivate {
			if brief, ok := userMap[c.PeerUserID]; ok {
				item.PeerNickname = brief.Nickname
				item.PeerAvatar = brief.Avatar
			}
		} else if c.Type == constants.ConversationTypeGroup {
			if gBrief, ok := groupMap[c.ID]; ok {
				item.PeerNickname = gBrief.Name
				item.PeerAvatar = gBrief.Avatar
				item.GroupID = gBrief.ID
			}
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

	messages, err := s.msgDAO.GetByConversation(ctx, req.ConversationID, req.BeforeID, member.ClearBeforeMsgID, limit+1)
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

// MarkGroupMessagesRead 标记群聊消息已读（消息级别）
// 将指定消息标记为已读 + 推送已读计数变化给消息发送者
func (s *IMService) MarkGroupMessagesRead(ctx context.Context, userID int64, req *dto.MarkGroupReadRequest) error {
	funcName := "service.im_service.MarkGroupMessagesRead"
	logs.Info(ctx, funcName, "群消息标记已读",
		zap.Int64("user_id", userID), zap.Int64("conversation_id", req.ConversationID),
		zap.Int("msg_count", len(req.MessageIDs)))

	member, err := s.convDAO.GetMember(ctx, req.ConversationID, userID)
	if err != nil || member == nil {
		return ErrNotMember
	}

	if s.readRecorder == nil || len(req.MessageIDs) == 0 {
		return nil
	}

	if err := s.readRecorder.BatchCreateReads(ctx, req.MessageIDs, userID); err != nil {
		logs.Error(ctx, funcName, "批量创建已读记录失败", zap.Error(err))
		return err
	}

	if member.AtMeCount > 0 {
		if err := s.convDAO.ClearAtMeCount(ctx, req.ConversationID, userID); err != nil {
			logs.Error(ctx, funcName, "清零@计数失败", zap.Error(err))
		}
	}

	readCounts, err := s.readRecorder.GetReadCountBatch(ctx, req.MessageIDs)
	if err != nil {
		logs.Error(ctx, funcName, "获取已读计数失败", zap.Error(err))
		return nil
	}

	for _, msgID := range req.MessageIDs {
		msg, mErr := s.msgDAO.GetByID(ctx, msgID)
		if mErr != nil || msg.SenderID == userID {
			continue
		}
		count := readCounts[msgID]
		s.pushToUser(ctx, msg.SenderID, "im.message.read.count", map[string]interface{}{
			"message_id":      msgID,
			"conversation_id": req.ConversationID,
			"read_count":      count,
		})
	}

	return nil
}

// GetMessageReadDetail 获取消息已读详情（已读/未读用户列表）
func (s *IMService) GetMessageReadDetail(ctx context.Context, userID int64, messageID int64) (*dto.GetReadDetailResponse, error) {
	funcName := "service.im_service.GetMessageReadDetail"
	logs.Debug(ctx, funcName, "获取已读详情",
		zap.Int64("user_id", userID), zap.Int64("message_id", messageID))

	msg, err := s.msgDAO.GetByID(ctx, messageID)
	if err != nil {
		return nil, ErrMsgNotFound
	}

	member, err := s.convDAO.GetMember(ctx, msg.ConversationID, userID)
	if err != nil || member == nil {
		return nil, ErrNotMember
	}

	memberIDs, err := s.convDAO.GetConversationMemberIDs(ctx, msg.ConversationID)
	if err != nil {
		return nil, err
	}

	readUserIDs := make(map[int64]bool)
	if s.readRecorder != nil {
		ids, rErr := s.readRecorder.GetReadUserIDs(ctx, messageID)
		if rErr != nil {
			logs.Error(ctx, funcName, "获取已读用户列表失败", zap.Error(rErr))
		} else {
			for _, id := range ids {
				readUserIDs[id] = true
			}
		}
	}

	allUserIDs := make([]int64, 0, len(memberIDs))
	for _, id := range memberIDs {
		if id != msg.SenderID {
			allUserIDs = append(allUserIDs, id)
		}
	}

	userMap := make(map[int64]*userBrief)
	if len(allUserIDs) > 0 {
		users, uErr := s.userInfoGetter.GetUsersByIDs(ctx, allUserIDs)
		if uErr != nil {
			logs.Error(ctx, funcName, "批量查询用户信息失败", zap.Error(uErr))
		} else {
			for i := range users {
				u := users[i]
				userMap[u.ID] = &userBrief{Nickname: u.Nickname, Avatar: u.Avatar}
			}
		}
	}

	var readList, unreadList []dto.MessageReadDetailDTO
	for _, uid := range allUserIDs {
		item := dto.MessageReadDetailDTO{
			UserID: uid,
		}
		if brief, ok := userMap[uid]; ok {
			item.UserNickname = brief.Nickname
			item.UserAvatar = brief.Avatar
		}
		if readUserIDs[uid] {
			readList = append(readList, item)
		} else {
			unreadList = append(unreadList, item)
		}
	}

	return &dto.GetReadDetailResponse{
		ReadList:   readList,
		UnreadList: unreadList,
		ReadCount:  len(readList),
		TotalCount: len(allUserIDs),
	}, nil
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

// ClearHistory 清空聊天记录（个人视图操作，仅影响当前用户，不影响对方）
// 通过记录清空截止消息 ID 实现，而非真正删除消息
func (s *IMService) ClearHistory(ctx context.Context, userID int64, conversationID int64) error {
	funcName := "service.im_service.ClearHistory"
	logs.Info(ctx, funcName, "清空聊天记录",
		zap.Int64("user_id", userID), zap.Int64("conversation_id", conversationID))

	member, err := s.convDAO.GetMember(ctx, conversationID, userID)
	if err != nil || member == nil {
		return ErrNotMember
	}

	latestMsgID, err := s.msgDAO.GetLatestMessageID(ctx, conversationID)
	if err != nil {
		logs.Error(ctx, funcName, "获取最新消息 ID 失败", zap.Error(err))
		return err
	}

	if err := s.convDAO.UpdateClearBefore(ctx, conversationID, userID, latestMsgID); err != nil {
		logs.Error(ctx, funcName, "更新清空截止 ID 失败", zap.Error(err))
		return err
	}

	if member.UnreadCount > 0 {
		s.decrementTotalUnread(ctx, userID, member.UnreadCount)
	}

	return nil
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

// PushTypingNotification 向对方推送正在输入通知（通过 PubSub 支持跨实例）
func (s *IMService) PushTypingNotification(ctx context.Context, conversationID, senderID int64) {
	peerID, err := s.convDAO.GetPeerUserID(ctx, conversationID, senderID)
	if err != nil {
		logs.Warn(ctx, "service.im_service.PushTypingNotification", "查询对方用户 ID 失败",
			zap.Int64("conversation_id", conversationID), zap.Error(err))
		return
	}
	s.pushToUser(ctx, peerID, "im.typing", map[string]interface{}{
		"conversation_id": conversationID,
		"user_id":         senderID,
	})
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

// decrementTotalUnread Redis 全局未读 -N（使用 Lua 脚本保证原子性，下限为 0）
func (s *IMService) decrementTotalUnread(ctx context.Context, userID int64, count int) {
	key := fmt.Sprintf("%s%d", unreadKeyPrefix, userID)
	script := redis.NewScript(`
		local current = tonumber(redis.call('GET', KEYS[1]) or '0')
		local decr = tonumber(ARGV[1])
		local newVal = current - decr
		if newVal < 0 then newVal = 0 end
		redis.call('SET', KEYS[1], newVal)
		return newVal
	`)
	if err := script.Run(ctx, s.rdb, []string{key}, count).Err(); err != nil {
		logs.Error(ctx, "service.im_service.decrementTotalUnread", "Redis Lua 脚本执行失败",
			zap.Int64("user_id", userID), zap.Error(err))
	}
}

// toMessageDTO 将 model.Message 转换为 dto.MessageDTO
func (s *IMService) toMessageDTO(m *model.Message) *dto.MessageDTO {
	d := &dto.MessageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Type:           m.Type,
		Content:        m.Content,
		Status:         m.Status,
		ClientMsgID:    m.ClientMsgID,
		CreatedAt:      m.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if len(m.AtUserIDs) > 0 {
		d.AtUserIDs = []int64(m.AtUserIDs)
	}
	return d
}

// buildMessagePushData 构建消息推送数据（单聊/群聊通用）
func (s *IMService) buildMessagePushData(ctx context.Context, msg *model.Message, senderID int64) map[string]interface{} {
	pushData := map[string]interface{}{
		"id":              msg.ID,
		"conversation_id": msg.ConversationID,
		"sender_id":       senderID,
		"type":            msg.Type,
		"content":         msg.Content,
		"client_msg_id":   msg.ClientMsgID,
		"created_at":      msg.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if senderUsers, sErr := s.userInfoGetter.GetUsersByIDs(ctx, []int64{senderID}); sErr == nil && len(senderUsers) > 0 {
		pushData["sender_name"] = senderUsers[0].Nickname
		pushData["sender_avatar"] = senderUsers[0].Avatar
	}
	return pushData
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
