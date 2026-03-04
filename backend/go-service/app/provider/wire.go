//go:build wireinject

// Package provider 提供 Wire 依赖注入入口
package provider

import (
	"github.com/echochat/backend/app/admin"
	"github.com/echochat/backend/app/auth"
	"github.com/echochat/backend/app/contact"
	authService "github.com/echochat/backend/app/auth/service"
	contactDAO "github.com/echochat/backend/app/contact/dao"
	contactService "github.com/echochat/backend/app/contact/service"
	imApp "github.com/echochat/backend/app/im"
	imService "github.com/echochat/backend/app/im/service"
	wsApp "github.com/echochat/backend/app/ws"
	"github.com/echochat/backend/config"
	"github.com/google/wire"
)

// InitializeApp 初始化整个应用（Wire 自动生成实现）
func InitializeApp(cfg *config.Config) (*App, error) {
	wire.Build(
		InfraSet,
		auth.AuthSet,
		admin.AdminSet,
		wsApp.WSSet,
		contact.ContactSet,
		imApp.IMSet,
		wire.Bind(new(wsApp.FriendIDsGetter), new(*contactDAO.FriendshipDAO)),
		wire.Bind(new(wsApp.TokenValidator), new(*authService.AuthService)),
		wire.Bind(new(imService.FriendChecker), new(*contactDAO.FriendshipDAO)),
		wire.Bind(new(imService.UserInfoGetter), new(*contactDAO.FriendshipDAO)),
		wire.Bind(new(contactService.OnlineChecker), new(*wsApp.OnlineService)),
	)
	return nil, nil
}
