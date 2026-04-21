// Package service 提供 notify 模块的业务逻辑
package service

import (
	"context"
	"encoding/json"

	authModel "github.com/echochat/backend/app/auth/model"
)

// PushPayload 跨模块推送通知的统一入参
// 由上游业务模块（contact/group/admin/meeting）构造并传递给 Pusher
type PushPayload struct {
	UserID     int64       // 接收者用户 ID（必填）
	Type       string      // 通知类型常量，见 constants.NotifyType*
	Title      string      // 通知标题（可选，为空时使用类型默认文案）
	Content    string      // 通知副文案
	ActorID    *int64      // 触发主体用户 ID（申请人/邀请人等），系统广播为 nil
	TargetType string      // 业务对象类型：user / group / meeting / system
	TargetID   *int64      // 业务对象 ID
	Extra      interface{} // 扩展数据，内部会序列化为 JSON 字符串存入 extra 列
}

// Pusher 通知推送接口
// 由 notify 模块实现，供 contact / group / admin / meeting 等上游模块注入调用
// 实现原则：
//  1. 入库 + WS 推送 双动作，WS 推送失败不回滚入库（降级策略）
//  2. 同步调用可能阻塞业务，实现内部使用 goroutine 异步处理
//  3. 一个 PushPayload 对应单个接收者；批量场景请多次调用
type Pusher interface {
	Push(ctx context.Context, payload *PushPayload)
	PushBatch(ctx context.Context, payloads []*PushPayload)
}

// UserInfoResolver 查询用户昵称 / 头像的接口，用于补全通知中的 actor 信息
// 由 contact.FriendshipDAO 隐式实现（已有 GetUsersByIDs 方法）
type UserInfoResolver interface {
	GetUsersByIDs(ctx context.Context, userIDs []int64) ([]authModel.User, error)
}

// marshalExtra 将 Extra 字段序列化为 JSON 字符串，供 DAO 写入 jsonb 列
// 空值返回 nil；若调用方已传入合法 JSON 字符串（string 或 *string），直接透传；其余类型走 json.Marshal
func marshalExtra(extra interface{}) *string {
	if extra == nil {
		return nil
	}
	switch v := extra.(type) {
	case string:
		if v == "" {
			return nil
		}
		return &v
	case *string:
		if v == nil || *v == "" {
			return nil
		}
		cp := *v
		return &cp
	}
	bytes, err := json.Marshal(extra)
	if err != nil {
		return nil
	}
	s := string(bytes)
	return &s
}
