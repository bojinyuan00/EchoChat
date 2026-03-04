// Package file 文件上传模块 Wire 依赖注入配置
package file

import (
	"github.com/echochat/backend/app/file/controller"
	"github.com/echochat/backend/app/file/service"
	"github.com/google/wire"
)

// FileSet 文件上传模块 Wire Provider Set
var FileSet = wire.NewSet(
	service.NewFileService,
	controller.NewFileController,
)
