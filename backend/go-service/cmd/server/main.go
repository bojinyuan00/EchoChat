// Package main 是 EchoChat 后端服务的入口
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/app/provider"
	"github.com/echochat/backend/router"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("config", "config.dev")
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志系统
	if err := logs.Init(&cfg.Log); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logs.Sync()

	ctx := context.Background()
	logs.Info(ctx, "main", "EchoChat 服务启动中",
		zap.String("mode", cfg.Server.Mode),
		zap.Int("port", cfg.Server.Port),
	)

	// 3. 通过 Wire 初始化所有组件
	app, err := provider.InitializeApp(cfg)
	if err != nil {
		logs.Fatal(ctx, "main", "初始化应用失败", zap.Error(err))
	}

	// 4. 创建 Gin Engine
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()

	// 5. 注册中间件（顺序：Trace → Logger → CORS → Recovery）
	engine.Use(
		middleware.Trace(),
		middleware.Logger(),
		middleware.CORS(),
		middleware.Recovery(),
	)

	// 6. 注册路由（由 router.Setup 统一汇总各模块路由）
	router.Setup(engine, app)

	// 7. 启动 HTTP 服务（优雅关闭）
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: engine,
	}

	go func() {
		logs.Info(ctx, "main", "HTTP 服务启动", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.Fatal(ctx, "main", "HTTP 服务启动失败", zap.Error(err))
		}
	}()

	// 等待中断信号，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logs.Info(ctx, "main", "正在关闭服务...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logs.Error(ctx, "main", "服务关闭失败", zap.Error(err))
	}

	logs.Info(ctx, "main", "服务已停止")
}
