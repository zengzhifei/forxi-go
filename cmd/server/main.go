package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"forxi.cn/forxi-go/api"
	"forxi.cn/forxi-go/app/config"
	"forxi.cn/forxi-go/app/database"
	"forxi.cn/forxi-go/app/middleware"
	"forxi.cn/forxi-go/app/util"

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

	// 初始化短雪花算法（workerID和datacenterID可以通过配置设置）
	// 这里简单使用固定值1,0（机器ID=1, 数据中心ID=0）
	if err := util.InitShortSnowflake(1, 0); err != nil {
		log.Fatalf("Failed to initialize short snowflake: %v", err)
	}

	// 设置Gin运行模式
	gin.SetMode(cfg.Server.Mode)

	// 初始化日志
	if err := middleware.InitLogger(&cfg.Log); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer middleware.Logger.Sync()

	middleware.Logger.Info("Starting Forxi-Go Auth Server...")
	configJSON, _ := json.Marshal(cfg)
	middleware.Logger.Info("Config loaded", zap.String("config", string(configJSON)))

	// 初始化数据库连接
	if err := database.InitDatabase(&cfg.Database); err != nil {
		middleware.Logger.Fatal("Failed to initialize database", zap.String("error", err.Error()))
	}

	// 初始化Redis连接
	if err := database.InitRedis(&cfg.Redis); err != nil {
		middleware.Logger.Fatal("Failed to initialize redis", zap.String("error", err.Error()))
	}
	middleware.Logger.Info("Redis connected successfully")

	// 创建Gin引擎
	router := gin.New()

	// 恢复中间件
	router.Use(gin.Recovery())

	// 设置路由
	api.SetupRoutes(router, cfg)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 在goroutine中启动服务器
	go func() {
		middleware.Logger.Info(fmt.Sprintf("Server is running on port %d", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			middleware.Logger.Fatal("Server failed to start", zap.String("error", err.Error()))
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	middleware.Logger.Info("Shutting down server...")

	// 创建一个5秒的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		middleware.Logger.Fatal("Server forced to shutdown", zap.String("error", err.Error()))
	}

	middleware.Logger.Info("Server exited")
}
