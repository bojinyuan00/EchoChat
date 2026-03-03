package dao

import (
	"context"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/im/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MessageDAO 消息数据访问对象
type MessageDAO struct {
	db *gorm.DB
}

// NewMessageDAO 创建 MessageDAO 实例
func NewMessageDAO(db *gorm.DB) *MessageDAO {
	return &MessageDAO{db: db}
}

// Create 写入一条消息
func (d *MessageDAO) Create(ctx context.Context, msg *model.Message) error {
	funcName := "dao.message_dao.Create"
	logs.Debug(ctx, funcName, "写入消息",
		zap.Int64("conversation_id", msg.ConversationID), zap.Int64("sender_id", msg.SenderID))

	err := d.db.WithContext(ctx).Create(msg).Error
	if err != nil {
		logs.Error(ctx, funcName, "写入消息失败", zap.Error(err))
	}
	return err
}

// GetByID 根据 ID 查询单条消息
func (d *MessageDAO) GetByID(ctx context.Context, id int64) (*model.Message, error) {
	var msg model.Message
	err := d.db.WithContext(ctx).First(&msg, id).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetByConversation 获取会话的历史消息（游标分页，按 ID 降序）
// beforeID > 0 时作为游标，查询 ID 小于 beforeID 的消息
// beforeID = 0 时查询最新的 limit 条消息
func (d *MessageDAO) GetByConversation(ctx context.Context, conversationID int64, beforeID int64, limit int) ([]model.Message, error) {
	funcName := "dao.message_dao.GetByConversation"
	logs.Debug(ctx, funcName, "查询历史消息",
		zap.Int64("conversation_id", conversationID),
		zap.Int64("before_id", beforeID), zap.Int("limit", limit))

	query := d.db.WithContext(ctx).
		Where("conversation_id = ? AND status != ?", conversationID, constants.MessageStatusDeleted)

	if beforeID > 0 {
		query = query.Where("id < ?", beforeID)
	}

	var messages []model.Message
	err := query.Order("id DESC").Limit(limit).Find(&messages).Error
	if err != nil {
		logs.Error(ctx, funcName, "查询历史消息失败", zap.Error(err))
	}
	return messages, err
}

// UpdateStatus 更新消息状态（撤回/删除）
func (d *MessageDAO) UpdateStatus(ctx context.Context, id int64, status int) error {
	funcName := "dao.message_dao.UpdateStatus"
	logs.Info(ctx, funcName, "更新消息状态",
		zap.Int64("message_id", id), zap.Int("status", status))

	return d.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// DeleteByConversation 软删除会话中所有消息（标记 status=3）
func (d *MessageDAO) DeleteByConversation(ctx context.Context, conversationID int64) error {
	funcName := "dao.message_dao.DeleteByConversation"
	logs.Info(ctx, funcName, "清空会话消息",
		zap.Int64("conversation_id", conversationID))

	return d.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND status = ?", conversationID, constants.MessageStatusNormal).
		Update("status", constants.MessageStatusDeleted).Error
}

// SearchMessages 全局消息搜索（按关键词匹配，仅搜索用户所在会话的消息）
// 返回匹配的消息列表（已 JOIN 成员表确保权限）
func (d *MessageDAO) SearchMessages(ctx context.Context, userID int64, keyword string, limit int) ([]MessageSearchResult, error) {
	funcName := "dao.message_dao.SearchMessages"
	logs.Debug(ctx, funcName, "全局消息搜索",
		zap.Int64("user_id", userID), zap.String("keyword", keyword))

	var results []MessageSearchResult
	err := d.db.WithContext(ctx).
		Raw(`SELECT m.id, m.conversation_id, m.sender_id, m.content, m.created_at
			 FROM im_messages m
			 JOIN im_conversation_members cm ON cm.conversation_id = m.conversation_id
			 WHERE cm.user_id = ? AND cm.is_deleted = false
			   AND m.status = ? AND m.content LIKE ?
			 ORDER BY m.created_at DESC
			 LIMIT ?`,
			userID, constants.MessageStatusNormal, "%"+keyword+"%", limit).
		Scan(&results).Error

	if err != nil {
		logs.Error(ctx, funcName, "全局消息搜索失败", zap.Error(err))
	}
	return results, err
}

// MessageSearchResult 搜索结果单条记录
type MessageSearchResult struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversation_id"`
	SenderID       int64  `json:"sender_id"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
}

// FindByClientMsgID 根据客户端消息 ID 查询（幂等去重用）
func (d *MessageDAO) FindByClientMsgID(ctx context.Context, conversationID int64, clientMsgID string) (*model.Message, error) {
	if clientMsgID == "" {
		return nil, nil
	}

	var msg model.Message
	err := d.db.WithContext(ctx).
		Where("conversation_id = ? AND client_msg_id = ?", conversationID, clientMsgID).
		First(&msg).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetLatestMessageID 获取会话中最新一条消息的 ID（用于标记已读）
func (d *MessageDAO) GetLatestMessageID(ctx context.Context, conversationID int64) (int64, error) {
	var id int64
	err := d.db.WithContext(ctx).
		Model(&model.Message{}).
		Select("COALESCE(MAX(id), 0)").
		Where("conversation_id = ?", conversationID).
		Scan(&id).Error
	return id, err
}
