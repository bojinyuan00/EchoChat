// Package service 提供 IM 模块的业务逻辑
package service

import (
	"context"

	authModel "github.com/echochat/backend/app/auth/model"
	"github.com/echochat/backend/app/dto"
)

// FriendChecker 好友关系校验接口
// 由 contact.FriendshipDAO 隐式实现，通过 Wire 接口注入避免直接依赖 contact 包
type FriendChecker interface {
	IsFriend(ctx context.Context, userID, targetID int64) (bool, error)
}

// UserInfoGetter 用户信息批量查询接口
// 由 contact.FriendshipDAO 隐式实现，返回 id/username/nickname/avatar
type UserInfoGetter interface {
	GetUsersByIDs(ctx context.Context, userIDs []int64) ([]authModel.User, error)
}

// GroupInfoGetter 群信息查询接口（Phase 2c）
// 由 group.GroupDAO 隐式实现，通过 Wire 接口注入
// IM 模块通过此接口获取群表信息（判断全体禁言、已解散等）
// 成员信息（角色/禁言/列表）通过 convDAO 直接查询 im_conversation_members 表
type GroupInfoGetter interface {
	GetGroupBrief(ctx context.Context, conversationID int64) (*dto.GroupBrief, error)
}

// MessageReadRecorder 群消息已读记录接口（Phase 2c）
// 由 group.MessageReadDAO 隐式实现
type MessageReadRecorder interface {
	BatchCreateReads(ctx context.Context, messageIDs []int64, userID int64) error
	GetReadCountBatch(ctx context.Context, messageIDs []int64) (map[int64]int, error)
	GetReadUserIDs(ctx context.Context, messageID int64) ([]int64, error)
}
