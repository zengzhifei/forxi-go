package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"forxi.cn/forxi-go/api"
	"forxi.cn/forxi-go/app/config"
	"forxi.cn/forxi-go/app/resource"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 加载配置文件
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.dev.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置Gin运行模式
	gin.SetMode(cfg.Server.Mode)

	// 初始化资源
	if err := resource.Init(cfg); err != nil {
		log.Printf("Failed to initialize resource: %v", err)
	}

	resource.Logger.Info("Starting Forxi-Go Auth Server...")

	// 创建Gin引擎
	router := gin.New()

	// 恢复中间件
	router.Use(gin.Recovery())

	// 设置路由
	api.SetupRoutes(router)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 在goroutine中启动服务器
	go func() {
		resource.Logger.Info(fmt.Sprintf("Server is running on port %d", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			resource.Logger.Fatal("Server failed to start", zap.String("error", err.Error()))
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	resource.Logger.Info("Shutting down server...")

	// 创建一个5秒的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		resource.Logger.Fatal("Server forced to shutdown", zap.String("error", err.Error()))
	}

	resource.Logger.Info("Server exited")
}
