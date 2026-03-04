// Package dao 提供 IM 模块的数据库访问操作
package dao

import (
	"context"
	"time"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/im/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ConversationDAO 会话数据访问对象
type ConversationDAO struct {
	db *gorm.DB
}

// NewConversationDAO 创建 ConversationDAO 实例
func NewConversationDAO(db *gorm.DB) *ConversationDAO {
	return &ConversationDAO{db: db}
}

// FindPrivateConversation 查找两人之间的单聊会话
// 通过 JOIN im_conversation_members 判断是否已存在，避免重复创建
func (d *ConversationDAO) FindPrivateConversation(ctx context.Context, userID, targetUserID int64) (*model.Conversation, error) {
	funcName := "dao.conversation_dao.FindPrivateConversation"
	logs.Debug(ctx, funcName, "查找单聊会话",
		zap.Int64("user_id", userID), zap.Int64("target_user_id", targetUserID))

	var conv model.Conversation
	err := d.db.WithContext(ctx).
		Raw(`SELECT c.* FROM im_conversations c
			 JOIN im_conversation_members cm1 ON cm1.conversation_id = c.id
			 JOIN im_conversation_members cm2 ON cm2.conversation_id = c.id
			 WHERE cm1.user_id = ? AND cm2.user_id = ? AND c.type = ?
			 LIMIT 1`,
			userID, targetUserID, constants.ConversationTypePrivate).
		Scan(&conv).Error

	if err != nil {
		logs.Error(ctx, funcName, "查找单聊会话失败", zap.Error(err))
		return nil, err
	}
	if conv.ID == 0 {
		return nil, nil
	}
	return &conv, nil
}

// CreateWithMembers 在事务中创建会话及其成员记录
func (d *ConversationDAO) CreateWithMembers(ctx context.Context, conv *model.Conversation, memberIDs []int64) error {
	funcName := "dao.conversation_dao.CreateWithMembers"
	logs.Info(ctx, funcName, "创建会话及成员",
		zap.Int("type", conv.Type), zap.Int("member_count", len(memberIDs)))

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conv).Error; err != nil {
			logs.Error(ctx, funcName, "创建会话失败", zap.Error(err))
			return err
		}

		for _, uid := range memberIDs {
			member := &model.ConversationMember{
				ConversationID: conv.ID,
				UserID:         uid,
			}
			if err := tx.Create(member).Error; err != nil {
				logs.Error(ctx, funcName, "创建会话成员失败",
					zap.Int64("conversation_id", conv.ID), zap.Int64("user_id", uid), zap.Error(err))
				return err
			}
		}
		return nil
	})
}

// GetUserConversations 获取用户的会话列表（排除已软删除的）
// 返回会话基本信息、该用户的成员视图（未读数、置顶等）和单聊对方用户 ID
func (d *ConversationDAO) GetUserConversations(ctx context.Context, userID int64) ([]ConversationWithMember, error) {
	funcName := "dao.conversation_dao.GetUserConversations"
	logs.Debug(ctx, funcName, "查询用户会话列表", zap.Int64("user_id", userID))

	var results []ConversationWithMember
	err := d.db.WithContext(ctx).
		Raw(`SELECT c.id, c.type, c.last_message_id, c.last_msg_content, c.last_msg_time, c.last_msg_sender_id,
			        cm.is_pinned, cm.unread_count, cm.clear_before_msg_id,
			        cm.is_do_not_disturb, cm.at_me_count,
			        COALESCE(peer.user_id, 0) AS peer_user_id
			 FROM im_conversations c
			 JOIN im_conversation_members cm ON cm.conversation_id = c.id AND cm.user_id = ?
			 LEFT JOIN im_conversation_members peer ON peer.conversation_id = c.id AND peer.user_id != ? AND c.type = ?
			 WHERE cm.is_deleted = false
			 ORDER BY cm.is_pinned DESC, c.last_msg_time DESC NULLS LAST`,
			userID, userID, constants.ConversationTypePrivate).
		Scan(&results).Error

	if err != nil {
		logs.Error(ctx, funcName, "查询用户会话列表失败", zap.Error(err))
	}
	return results, err
}

// ConversationWithMember 会话列表查询结果（JOIN 成员表 + LEFT JOIN 对方成员）
type ConversationWithMember struct {
	ID               int64      `json:"id"`
	Type             int        `json:"type"`
	LastMessageID    *int64     `json:"last_message_id"`
	LastMsgContent   string     `json:"last_msg_content"`
	LastMsgTime      *time.Time `json:"last_msg_time"`
	LastMsgSenderID  *int64     `json:"last_msg_sender_id"`
	IsPinned         bool       `json:"is_pinned"`
	UnreadCount      int        `json:"unread_count"`
	ClearBeforeMsgID int64      `json:"clear_before_msg_id"`
	IsDoNotDisturb   bool       `json:"is_do_not_disturb"`
	AtMeCount        int        `json:"at_me_count"`
	PeerUserID       int64      `json:"peer_user_id"`
}

// GetMember 获取指定会话中指定用户的成员记录
func (d *ConversationDAO) GetMember(ctx context.Context, conversationID, userID int64) (*model.ConversationMember, error) {
	funcName := "dao.conversation_dao.GetMember"
	logs.Debug(ctx, funcName, "查询会话成员",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID))

	var member model.ConversationMember
	err := d.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// GetPeerUserID 获取单聊会话中对方的用户 ID
func (d *ConversationDAO) GetPeerUserID(ctx context.Context, conversationID, userID int64) (int64, error) {
	funcName := "dao.conversation_dao.GetPeerUserID"
	logs.Debug(ctx, funcName, "查询对方用户 ID",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID))

	var peerID int64
	err := d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Select("user_id").
		Where("conversation_id = ? AND user_id != ?", conversationID, userID).
		Scan(&peerID).Error
	return peerID, err
}

// GetConversationMemberIDs 获取会话的所有成员 ID
func (d *ConversationDAO) GetConversationMemberIDs(ctx context.Context, conversationID int64) ([]int64, error) {
	var ids []int64
	err := d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ?", conversationID).
		Pluck("user_id", &ids).Error
	return ids, err
}

// UpdateMemberPinned 更新会话成员的置顶状态
func (d *ConversationDAO) UpdateMemberPinned(ctx context.Context, conversationID, userID int64, isPinned bool) error {
	funcName := "dao.conversation_dao.UpdateMemberPinned"
	logs.Info(ctx, funcName, "更新置顶状态",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID), zap.Bool("is_pinned", isPinned))

	return d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("is_pinned", isPinned).Error
}

// SoftDeleteMember 软删除会话（仅影响当前用户视图，不影响对方）
func (d *ConversationDAO) SoftDeleteMember(ctx context.Context, conversationID, userID int64) error {
	funcName := "dao.conversation_dao.SoftDeleteMember"
	logs.Info(ctx, funcName, "软删除会话",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID))

	return d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"is_deleted":   true,
			"unread_count": 0,
		}).Error
}

// RestoreMember 恢复已软删除的会话成员（收到新消息时自动恢复）
func (d *ConversationDAO) RestoreMember(ctx context.Context, conversationID, userID int64) error {
	return d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("is_deleted", false).Error
}

// UpdateLastMessage 更新会话的最后消息信息（冗余字段，避免列表查询 JOIN）
func (d *ConversationDAO) UpdateLastMessage(ctx context.Context, conversationID int64, msgID int64, content string, senderID int64, msgTime time.Time) error {
	funcName := "dao.conversation_dao.UpdateLastMessage"
	logs.Debug(ctx, funcName, "更新最后消息",
		zap.Int64("conversation_id", conversationID), zap.Int64("message_id", msgID))

	return d.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"last_message_id":    msgID,
			"last_msg_content":   content,
			"last_msg_time":      msgTime,
			"last_msg_sender_id": senderID,
		}).Error
}

// IncrementUnread 将指定成员的未读消息数 +1
func (d *ConversationDAO) IncrementUnread(ctx context.Context, conversationID, userID int64) error {
	return d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1")).Error
}

// IncrementAtMeCount 将指定成员的 @提醒计数 +1
func (d *ConversationDAO) IncrementAtMeCount(ctx context.Context, conversationID, userID int64) error {
	return d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		UpdateColumn("at_me_count", gorm.Expr("at_me_count + 1")).Error
}

// ClearAtMeCount 清零指定成员的 @提醒计数
func (d *ConversationDAO) ClearAtMeCount(ctx context.Context, conversationID, userID int64) error {
	return d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("at_me_count", 0).Error
}

// GetMemberDNDMap 批量获取会话成员的免打扰状态（返回 userID → isDoNotDisturb 的映射）
func (d *ConversationDAO) GetMemberDNDMap(ctx context.Context, conversationID int64) (map[int64]bool, error) {
	type memberDND struct {
		UserID         int64 `json:"user_id"`
		IsDoNotDisturb bool  `json:"is_do_not_disturb"`
	}
	var members []memberDND
	err := d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Select("user_id, is_do_not_disturb").
		Where("conversation_id = ?", conversationID).
		Scan(&members).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]bool, len(members))
	for _, m := range members {
		result[m.UserID] = m.IsDoNotDisturb
	}
	return result, nil
}

// ClearUnread 清零指定成员的未读消息数
func (d *ConversationDAO) ClearUnread(ctx context.Context, conversationID, userID int64, lastMsgID int64) error {
	return d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"unread_count":     0,
			"last_read_msg_id": lastMsgID,
		}).Error
}

// GetUnreadConversations 获取有未读消息的会话列表（用于离线消息推送）
func (d *ConversationDAO) GetUnreadConversations(ctx context.Context, userID int64) ([]ConversationWithMember, error) {
	funcName := "dao.conversation_dao.GetUnreadConversations"
	logs.Debug(ctx, funcName, "查询未读会话", zap.Int64("user_id", userID))

	var results []ConversationWithMember
	err := d.db.WithContext(ctx).
		Raw(`SELECT c.id, c.type, c.last_msg_content, c.last_msg_time, c.last_msg_sender_id,
			        cm.is_pinned, cm.unread_count
			 FROM im_conversations c
			 JOIN im_conversation_members cm ON cm.conversation_id = c.id
			 WHERE cm.user_id = ? AND cm.is_deleted = false AND cm.unread_count > 0
			 ORDER BY c.last_msg_time DESC`,
			userID).
		Scan(&results).Error

	if err != nil {
		logs.Error(ctx, funcName, "查询未读会话失败", zap.Error(err))
	}
	return results, err
}

// UpdateClearBefore 更新用户的清空记录截止消息 ID（个人视图操作，不影响对方）
func (d *ConversationDAO) UpdateClearBefore(ctx context.Context, conversationID, userID, lastMsgID int64) error {
	funcName := "dao.conversation_dao.UpdateClearBefore"
	logs.Info(ctx, funcName, "更新清空记录截止 ID",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID),
		zap.Int64("last_msg_id", lastMsgID))

	return d.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]interface{}{
			"clear_before_msg_id": lastMsgID,
			"unread_count":        0,
		}).Error
}

// GetByID 根据 ID 获取会话
func (d *ConversationDAO) GetByID(ctx context.Context, id int64) (*model.Conversation, error) {
	var conv model.Conversation
	err := d.db.WithContext(ctx).First(&conv, id).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}
