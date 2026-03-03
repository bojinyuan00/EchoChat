// Package service 提供 IM 模块的业务逻辑
package service

import (
	"context"

	authModel "github.com/echochat/backend/app/auth/model"
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
