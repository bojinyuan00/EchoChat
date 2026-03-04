//go:build wireinject

// Package provider 提供 Wire 依赖注入入口
package provider

import (
	"github.com/echochat/backend/app/admin"
	"github.com/echochat/backend/app/auth"
	authService "github.com/echochat/backend/app/auth/service"
	"github.com/echochat/backend/app/contact"
	contactDAO "github.com/echochat/backend/app/contact/dao"
	contactService "github.com/echochat/backend/app/contact/service"
	fileApp "github.com/echochat/backend/app/file"
	groupApp "github.com/echochat/backend/app/group"
	groupDAO "github.com/echochat/backend/app/group/dao"
	groupService "github.com/echochat/backend/app/group/service"
	imApp "github.com/echochat/backend/app/im"
	imDAO "github.com/echochat/backend/app/im/dao"
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
		fileApp.FileSet,
		groupApp.GroupSet,
		wire.Bind(new(wsApp.FriendIDsGetter), new(*contactDAO.FriendshipDAO)),
		wire.Bind(new(groupService.UserInfoProvider), new(*contactDAO.FriendshipDAO)),
		wire.Bind(new(imService.GroupInfoGetter), new(*groupDAO.GroupDAO)),
		wire.Bind(new(imService.MessageReadRecorder), new(*groupDAO.MessageReadDAO)),
		wire.Bind(new(wsApp.TokenValidator), new(*authService.AuthService)),
		wire.Bind(new(imService.FriendChecker), new(*contactDAO.FriendshipDAO)),
		wire.Bind(new(imService.UserInfoGetter), new(*contactDAO.FriendshipDAO)),
		wire.Bind(new(contactService.OnlineChecker), new(*wsApp.OnlineService)),
		wire.Bind(new(groupService.MessageWriter), new(*imDAO.MessageDAO)),
	)
	return nil, nil
}
