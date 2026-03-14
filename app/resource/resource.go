package resource

import (
	"encoding/json"
	"fmt"

	"forxi.cn/forxi-go/app/config"
	databaseClient "forxi.cn/forxi-go/app/resource/database"
	email "forxi.cn/forxi-go/app/resource/email"
	loggerClient "forxi.cn/forxi-go/app/resource/logger"
	redisClient "forxi.cn/forxi-go/app/resource/rds"
	snowflakeClient "forxi.cn/forxi-go/app/resource/snowflake"
	storageClient "forxi.cn/forxi-go/app/resource/storage"

	redislib "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 全局资源变量
var (
	Cfg            *config.Config
	Logger         *zap.Logger
	Snowflake      *snowflakeClient.Snowflake
	ShortSnowflake *snowflakeClient.ShortSnowflake
	DB             *gorm.DB
	Redis          *redislib.Client
	Storage        storageClient.Storage
	EmailService   *email.EmailService
)

// Init 初始化所有公共服务资源（数据库、Redis、存储等）
func Init(_cfg *config.Config) error {
	// 初始化配置
	Cfg = _cfg

	// 初始化日志
	if _logger, err := loggerClient.InitLogger(&Cfg.Log); err != nil {
		return err
	} else {
		Logger = _logger
		defer Logger.Sync()
	}

	// 初始化雪花算法（workerID和datacenterID可以通过配置设置）
	// 这里简单使用固定值1,0（机器ID=1, 数据中心ID=0）
	if _snowflake, err := snowflakeClient.InitSnowflake(1, 0); err != nil {
		return err
	} else {
		Snowflake = _snowflake
	}

	// 初始化短雪花算法（workerID和datacenterID可以通过配置设置）
	// 这里简单使用固定值1,0（机器ID=1, 数据中心ID=0）
	if _shortSnowflake, err := snowflakeClient.InitShortSnowflake(1, 0); err != nil {
		return err
	} else {
		ShortSnowflake = _shortSnowflake
	}

	// 初始化数据库
	if _db, err := databaseClient.InitDatabase(&Cfg.Database); err != nil {
		return err
	} else {
		DB = _db
	}

	// 初始化 Redis
	if _redis, err := redisClient.InitRedis(&Cfg.Redis); err != nil {
		return err
	} else {
		Redis = _redis
	}

	// 初始化存储
	switch Cfg.Storage.Active {
	case "qiniu":
		_storage := storageClient.NewQiniuStorage()
		if err := _storage.Init(&Cfg.Storage); err != nil {
			return err
		} else {
			Storage = _storage
			break
		}
	default:
		return fmt.Errorf("不支持的存储类型: %s", Cfg.Storage.Active)
	}

	// 初始化邮件服务
	EmailService = email.InitEmailService(&Cfg.Email, Redis)

	// 打印配置信息
	configJson, _ := json.Marshal(Cfg)
	Logger.Info("Config loaded", zap.String("config", string(configJson)))

	return nil
}
