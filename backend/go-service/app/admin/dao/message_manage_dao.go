// Package dao 提供 admin 模块的数据库访问操作
package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/echochat/backend/app/dto"
	imModel "github.com/echochat/backend/app/im/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MessageManageDAO 管理端消息数据访问对象
type MessageManageDAO struct {
	db *gorm.DB
}

// NewMessageManageDAO 创建 MessageManageDAO 实例
func NewMessageManageDAO(db *gorm.DB) *MessageManageDAO {
	return &MessageManageDAO{db: db}
}

// ListMessages 分页查询消息列表（支持多条件筛选）
func (d *MessageManageDAO) ListMessages(ctx context.Context, req *dto.AdminMessageListRequest) ([]imModel.Message, int64, error) {
	funcName := "dao.message_manage_dao.ListMessages"
	logs.Debug(ctx, funcName, "查询消息列表")

	query := d.db.WithContext(ctx).Model(&imModel.Message{})

	if req.Keyword != "" {
		query = query.Where("content ILIKE ?", fmt.Sprintf("%%%s%%", req.Keyword))
	}
	if req.Type != nil {
		query = query.Where("type = ?", *req.Type)
	}
	if req.SenderID != nil {
		query = query.Where("sender_id = ?", *req.SenderID)
	}
	if req.ConversationID != nil {
		query = query.Where("conversation_id = ?", *req.ConversationID)
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	if req.StartTime != "" {
		if t, err := time.Parse("2006-01-02", req.StartTime); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse("2006-01-02", req.EndTime); err == nil {
			query = query.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		logs.Error(ctx, funcName, "统计消息总数失败", zap.Error(err))
		return nil, 0, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var messages []imModel.Message
	err := query.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&messages).Error
	if err != nil {
		logs.Error(ctx, funcName, "查询消息列表失败", zap.Error(err))
		return nil, 0, err
	}

	return messages, total, nil
}

// GetMessageByID 根据 ID 查询单条消息
func (d *MessageManageDAO) GetMessageByID(ctx context.Context, id int64) (*imModel.Message, error) {
	var msg imModel.Message
	err := d.db.WithContext(ctx).First(&msg, id).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// UpdateMessageStatus 更新消息状态
func (d *MessageManageDAO) UpdateMessageStatus(ctx context.Context, id int64, status int) error {
	return d.db.WithContext(ctx).
		Model(&imModel.Message{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// GetTotalCount 获取消息总数
func (d *MessageManageDAO) GetTotalCount(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&imModel.Message{}).Count(&count).Error
	return count, err
}

// GetTodayCount 获取今日消息数
func (d *MessageManageDAO) GetTodayCount(ctx context.Context) (int64, error) {
	var count int64
	today := time.Now().Truncate(24 * time.Hour)
	err := d.db.WithContext(ctx).Model(&imModel.Message{}).
		Where("created_at >= ?", today).Count(&count).Error
	return count, err
}

// GetTypeDistribution 获取消息类型分布
func (d *MessageManageDAO) GetTypeDistribution(ctx context.Context) ([]dto.TypeDistItem, error) {
	type row struct {
		Type  int   `gorm:"column:type"`
		Count int64 `gorm:"column:count"`
	}
	var rows []row
	err := d.db.WithContext(ctx).
		Model(&imModel.Message{}).
		Select("type, count(*) as count").
		Group("type").
		Order("count DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]dto.TypeDistItem, 0, len(rows))
	for _, r := range rows {
		result = append(result, dto.TypeDistItem{
			Type:  r.Type,
			Count: r.Count,
		})
	}
	return result, nil
}

// GetDailyTrend 获取每日消息趋势
func (d *MessageManageDAO) GetDailyTrend(ctx context.Context, days int) ([]dto.DailyTrendItem, error) {
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	type row struct {
		Date  string `gorm:"column:date"`
		Count int64  `gorm:"column:count"`
	}
	var rows []row
	err := d.db.WithContext(ctx).
		Model(&imModel.Message{}).
		Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date, count(*) as count").
		Where("created_at >= ?", startDate).
		Group("date").
		Order("date ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]dto.DailyTrendItem, 0, len(rows))
	for _, r := range rows {
		result = append(result, dto.DailyTrendItem{Date: r.Date, Count: r.Count})
	}
	return result, nil
}

// GetActiveUsers 获取活跃用户排行（Top 10）
func (d *MessageManageDAO) GetActiveUsers(ctx context.Context, days int) ([]dto.ActiveUserItem, error) {
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	type row struct {
		UserID   int64  `gorm:"column:sender_id"`
		Nickname string `gorm:"column:nickname"`
		Count    int64  `gorm:"column:count"`
	}
	var rows []row
	err := d.db.WithContext(ctx).
		Raw(`SELECT m.sender_id, u.nickname, count(*) as count
			 FROM im_messages m
			 JOIN auth_users u ON u.id = m.sender_id
			 WHERE m.created_at >= ? AND m.sender_id > 0
			 GROUP BY m.sender_id, u.nickname
			 ORDER BY count DESC
			 LIMIT 10`, startDate).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]dto.ActiveUserItem, 0, len(rows))
	for _, r := range rows {
		result = append(result, dto.ActiveUserItem{
			UserID:   r.UserID,
			Nickname: r.Nickname,
			Count:    r.Count,
		})
	}
	return result, nil
}

// GetActiveGroups 获取活跃群组排行（Top 10）
func (d *MessageManageDAO) GetActiveGroups(ctx context.Context, days int) ([]dto.ActiveGroupItem, error) {
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	type row struct {
		GroupID int64  `gorm:"column:id"`
		Name    string `gorm:"column:name"`
		Count   int64  `gorm:"column:count"`
	}
	var rows []row
	err := d.db.WithContext(ctx).
		Raw(`SELECT g.id, g.name, count(*) as count
			 FROM im_messages m
			 JOIN im_conversations c ON c.id = m.conversation_id
			 JOIN im_groups g ON g.conversation_id = c.id
			 WHERE m.created_at >= ? AND c.type = 2
			 GROUP BY g.id, g.name
			 ORDER BY count DESC
			 LIMIT 10`, startDate).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]dto.ActiveGroupItem, 0, len(rows))
	for _, r := range rows {
		result = append(result, dto.ActiveGroupItem{
			GroupID: r.GroupID,
			Name:    r.Name,
			Count:   r.Count,
		})
	}
	return result, nil
}
