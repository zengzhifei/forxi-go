package rds

import (
	"context"
	"fmt"

	"forxi.cn/forxi-go/app/config"

	"github.com/redis/go-redis/v9"
)

var redisPrefix string

// InitRedis 初始化Redis连接
func InitRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// 全局Redis前缀
	redisPrefix = cfg.Prefix

	return redisClient, nil
}
