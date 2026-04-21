package dto

// ====== 通知基础 DTO ======

// NotificationDTO 通知传输对象
// 前端列表渲染使用，字段顺序与 notify_notifications 一一对应
type NotificationDTO struct {
	ID         int64   `json:"id"`                      // 通知 ID
	Type       string  `json:"type"`                    // 通知类型常量
	Category   string  `json:"category"`                // 前端分类（friend / group / meeting / system），由 type 计算得出
	Title      string  `json:"title"`                   // 标题
	Content    string  `json:"content"`                 // 副文案
	Extra      *string `json:"extra,omitempty"`         // 扩展数据 JSON 字符串（前端按需解析）
	ActorID    *int64  `json:"actor_id,omitempty"`      // 触发用户 ID
	ActorName  string  `json:"actor_name,omitempty"`    // 触发用户昵称（后端按需补全）
	ActorAvatar string `json:"actor_avatar,omitempty"`  // 触发用户头像
	TargetType string  `json:"target_type"`             // 业务对象类型
	TargetID   *int64  `json:"target_id,omitempty"`     // 业务对象 ID
	IsRead     bool    `json:"is_read"`                 // 是否已读
	ReadAt     string  `json:"read_at,omitempty"`       // 已读时间（未读为空）
	CreatedAt  string  `json:"created_at"`              // 创建时间
}

// ====== REST API 请求/响应 DTO ======

// GetNotificationsRequest 通知列表查询参数
// GET /api/v1/notifications
type GetNotificationsRequest struct {
	Category string `form:"category"`                   // 分类过滤：all/friend/group/meeting/system，空视为 all
	IsRead   *bool  `form:"is_read"`                    // 已读状态过滤：nil=全部，true=已读，false=未读
	BeforeID int64  `form:"before_id"`                  // 游标分页：小于此 ID 的记录
	Limit    int    `form:"limit"`                      // 页大小（默认 20，最大 100）
}

// NotificationListResponse 通知列表响应
type NotificationListResponse struct {
	List    []NotificationDTO `json:"list"`    // 通知列表（按 created_at DESC 返回）
	HasMore bool              `json:"has_more"` // 是否还有更多数据
}

// UnreadCountResponse 未读数响应
type UnreadCountResponse struct {
	Total    int            `json:"total"`    // 未读总数
	ByCategory map[string]int `json:"by_category"` // 按分类未读数（friend/group/meeting/system）
}

// MarkReadResponse 标记已读响应
type MarkReadResponse struct {
	Affected int `json:"affected"` // 实际被标记为已读的记录数
}

// ====== 管理端广播 DTO ======

// BroadcastNotificationRequest 管理员发起系统广播请求
// POST /api/v1/admin/notifications/broadcast
type BroadcastNotificationRequest struct {
	Title      string  `json:"title" binding:"required,max=100"`   // 广播标题
	Content    string  `json:"content" binding:"required,max=500"` // 广播正文
	TargetType string  `json:"target_type"`                         // 业务对象类型，通常为 system
	TargetID   *int64  `json:"target_id"`                           // 业务对象 ID（可选）
	Extra      *string `json:"extra"`                               // 扩展 JSON
}

// BroadcastNotificationResponse 广播结果响应
type BroadcastNotificationResponse struct {
	Affected int `json:"affected"` // 成功入库的用户数
}
