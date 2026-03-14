package config

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config 应用配置结构体
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Redis       RedisConfig       `mapstructure:"redis"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	OAuth       OAuthConfig       `mapstructure:"oauth"`
	Email       EmailConfig       `mapstructure:"email"`
	Log         LogConfig         `mapstructure:"log"`
	RateLimit   RateLimitConfig   `mapstructure:"rate_limit"`
	FilePreview FilePreviewConfig `mapstructure:"file_preview"`
	Storage     StorageConfig     `mapstructure:"storage"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Mode         string `mapstructure:"mode"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	Charset         string `mapstructure:"charset"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret        string `mapstructure:"secret"`
	AccessExpire  int    `mapstructure:"access_expire"`
	RefreshExpire int    `mapstructure:"refresh_expire"`
	Issuer        string `mapstructure:"issuer"`
}

// OAuthConfig OAuth配置
type OAuthConfig struct {
	FrontendCallbackURL string        `mapstructure:"frontend_callback_url"`
	PasswordResetURL    string        `mapstructure:"password_reset_url"`
	GitHub              OAuthProvider `mapstructure:"github"`
}

// OAuthProvider OAuth提供商配置
type OAuthProvider struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	UserInfoURL  string `mapstructure:"user_info_url"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Prefix   string `mapstructure:"prefix"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	SMTPHost  string `mapstructure:"smtp_host"`
	SMTPPort  int    `mapstructure:"smtp_port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	FromEmail string `mapstructure:"from_email"`
	FromName  string `mapstructure:"from_name"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	QPS   int `mapstructure:"qps"`
	Burst int `mapstructure:"burst"`
}

// FilePreviewConfig 文件预览配置
type FilePreviewConfig struct {
	MaxFileSize int64 `mapstructure:"max_file_size"` // 文件大小限制（字节）
	CacheExpiry int   `mapstructure:"cache_expiry"`  // 缓存过期时间（秒）
}

// StorageConfig 存储配置
type StorageConfig struct {
	Active string            `mapstructure:"active"`
	Qiniu  StorageConfigItem `mapstructure:"qiniu"`
}

// QiniuConfig 七牛云配置
type StorageConfigItem struct {
	AccessKey string `mapstructure:"access_key"` // AccessKey
	SecretKey string `mapstructure:"secret_key"` // SecretKey
	Bucket    string `mapstructure:"bucket"`     // 存储空间名称
	Domain    string `mapstructure:"domain"`     // CDN域名
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	_ = godotenv.Load()

	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
