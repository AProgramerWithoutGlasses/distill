package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"goweb_staging/logger"
	"goweb_staging/pkg/settings"
	"goweb_staging/server"
	"log"
	"net/http"
	"os"
	"os/signal"

	"syscall"
	"time"
)

func main() {
	// 加载配置
	app, err := settings.Init("local")
	if err != nil {
		fmt.Printf("init setting failed,error: %v\n", err)
	}

	// 初始化日志
	err = logger.Init(app.LogConfig, app.Mode)
	if err != nil {
		fmt.Printf("init logger failed,error: %v\n", err)
	}

	// 开启服务
	router := server.Init(app)

	// 启动服务（优雅关机）
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.Port),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err) // Fatalf 相当于Printf()之后再调用os.Exit(1)。
		}
	}()

	quit := make(chan os.Signal, 1) // 创建一个接收信号的通道
	// kill 默认会发送 syscall.SIGTERM 信号
	// kill -2 发送 syscall.SIGINT 信号，Ctrl+C 就是触发系统SIGINT信号

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 此处不会阻塞
	<-quit                                               // 阻塞在此，当接收到上述两种信号时才会往下执行
	zap.L().Info("Shutdown Server ...")
	// 创建一个5秒超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server Shutdown", zap.Error(err))
	}

	zap.L().Info("Server exiting")
}
