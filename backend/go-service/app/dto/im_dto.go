package dto

// ====== 消息相关 DTO ======

// SendMessageRequest 发送消息请求（WS 事件 im.message.send 的 data 字段）
type SendMessageRequest struct {
	ConversationID int64   `json:"conversation_id"`  // 会话 ID（与 TargetUserID 二选一）
	TargetUserID   int64   `json:"target_user_id"`   // 对方用户 ID（首次发消息时使用，自动创建会话）
	Type           int     `json:"type"`              // 消息类型：1=文本, 2=图片, 3=语音, 5=文件
	Content        string  `json:"content"`           // 消息内容（富媒体消息时为空字符串）
	Extra          string  `json:"extra"`             // 扩展数据 JSON 字符串（图片/语音/文件元信息）
	ClientMsgID    string  `json:"client_msg_id"`     // 客户端消息唯一 ID，用于幂等去重
	AtUserIDs      []int64 `json:"at_user_ids"`       // @提醒用户 ID 列表（含 0 表示 @所有人）
}

// MessageDTO 消息传输对象（返回给前端）
type MessageDTO struct {
	ID             int64   `json:"id"`                        // 消息 ID
	ConversationID int64   `json:"conversation_id"`           // 所属会话 ID
	SenderID       int64   `json:"sender_id"`                 // 发送者用户 ID
	Type           int     `json:"type"`                      // 消息类型
	Content        string  `json:"content"`                   // 消息内容
	Extra          *string `json:"extra,omitempty"`           // 扩展数据 JSON（图片/语音/文件元信息）
	Status         int     `json:"status"`                    // 消息状态：1=正常，2=已撤回
	ClientMsgID    string  `json:"client_msg_id"`             // 客户端消息 ID
	AtUserIDs      []int64 `json:"at_user_ids"`               // @提醒用户 ID 列表
	CreatedAt      string  `json:"created_at"`                // 发送时间
}

// RecallMessageRequest 撤回消息请求（WS 事件 im.message.recall 的 data 字段）
type RecallMessageRequest struct {
	MessageID int64 `json:"message_id"` // 要撤回的消息 ID
}

// ====== 会话相关 DTO ======

// ConversationDTO 会话列表条目（返回给前端）
type ConversationDTO struct {
	ID              int64  `json:"id"`                 // 会话 ID
	Type            int    `json:"type"`               // 会话类型：1=单聊，2=群聊
	PeerUserID      int64  `json:"peer_user_id"`       // 单聊对方用户 ID（群聊为 0）
	PeerNickname    string `json:"peer_nickname"`      // 对方昵称（群聊时为群名称）
	PeerAvatar      string `json:"peer_avatar"`        // 对方头像（群聊时为群头像）
	LastMsgContent  string `json:"last_msg_content"`   // 最后消息预览
	LastMsgTime     string `json:"last_msg_time"`      // 最后消息时间
	LastMsgSenderID *int64 `json:"last_msg_sender_id"` // 最后消息发送者 ID
	IsPinned        bool   `json:"is_pinned"`          // 是否置顶
	UnreadCount     int    `json:"unread_count"`       // 未读消息数
	GroupID         int64  `json:"group_id,omitempty"` // 群聊 ID（仅 type=2 有值）
	IsDoNotDisturb  bool   `json:"is_do_not_disturb"`  // 是否免打扰
	AtMeCount       int    `json:"at_me_count"`        // 被@提醒未读计数
}

// ConversationListResponse 会话列表响应
type ConversationListResponse struct {
	List []ConversationDTO `json:"list"` // 会话列表
}

// PinConversationRequest 置顶/取消置顶请求
type PinConversationRequest struct {
	IsPinned bool `json:"is_pinned"` // true=置顶, false=取消
}

// ClearHistoryRequest 清空聊天记录请求（REST API）
type ClearHistoryRequest struct {
	ConversationID int64 `json:"conversation_id" binding:"required"` // 会话 ID
}

// ====== 消息搜索 DTO ======

// SearchMessageRequest 消息搜索请求
type SearchMessageRequest struct {
	Keyword string `form:"keyword" binding:"required"` // 搜索关键词
	Limit   int    `form:"limit"`                      // 返回条数（默认 50）
}

// SearchMessageResponse 消息搜索结果
type SearchMessageResponse struct {
	List []MessageSearchItem `json:"list"` // 搜索结果列表
}

// MessageSearchItem 搜索结果单条
type MessageSearchItem struct {
	MessageDTO
	SenderNickname string `json:"sender_nickname"` // 发送者昵称
	SenderAvatar   string `json:"sender_avatar"`   // 发送者头像
}

// ====== 历史消息 DTO ======

// HistoryMessageRequest 历史消息查询请求
type HistoryMessageRequest struct {
	ConversationID int64 `form:"conversation_id" binding:"required"` // 会话 ID
	BeforeID       int64 `form:"before_id"`                          // 游标：查询 ID 小于此值的消息，0 表示最新
	Limit          int   `form:"limit"`                              // 每次拉取条数（默认 30）
}

// HistoryMessageResponse 历史消息响应
//
// 已读状态回填设计（2026-04-20）：
//   - PeerLastReadMsgID：单聊时填"对方 last_read_msg_id"，用于前端在页面刷新后恢复 readStatusMap，
//     避免已读标签变未读的体验退化；群聊时为 0
//   - ReadCountMap：群聊时批量返回"自己发送消息的 read_count"，前端合并到 groupReadCountMap；
//     单聊时为 nil（JSON omitempty 省略，节省带宽）
type HistoryMessageResponse struct {
	List              []MessageDTO  `json:"list"`                         // 消息列表（ID 降序）
	HasMore           bool          `json:"has_more"`                     // 是否还有更多历史消息
	PeerLastReadMsgID int64         `json:"peer_last_read_msg_id"`        // 单聊：对方已读位置（群聊=0）
	ReadCountMap      map[int64]int `json:"read_count_map,omitempty"`     // 群聊：{messageID: readCount}（仅自己发的消息）
}

// ====== 已读标记 DTO ======

// MarkReadRequest 标记已读请求（WS 事件 im.conversation.read 的 data 字段）
type MarkReadRequest struct {
	ConversationID int64 `json:"conversation_id"` // 要标记已读的会话 ID
}

// ====== 输入状态 DTO ======

// TypingRequest 正在输入通知请求（WS 事件 im.typing 的 data 字段）
type TypingRequest struct {
	ConversationID int64 `json:"conversation_id"` // 会话 ID
}
