package model

import (
	"time"
)

// OAuthAccount 第三方账号模型
type OAuthAccount struct {
	ID             int64      `gorm:"primaryKey;autoIncrement;comment:OAuth账号ID" json:"-"`
	UserID         int64      `gorm:"not null;index:idx_user_id;comment:关联的用户ID（对应users表的user_id）" json:"user_id"`
	Provider       string     `gorm:"type:varchar(50);not null;index:idx_provider;comment:OAuth提供商: wechat/github" json:"provider"`
	ProviderUserID string     `gorm:"type:varchar(255);not null;comment:第三方平台用户ID" json:"provider_user_id"`
	AccessToken    string     `gorm:"type:text;comment:访问令牌" json:"-"`
	RefreshToken   string     `gorm:"type:text;comment:刷新令牌" json:"-"`
	ExpiresAt      *time.Time `gorm:"index:idx_expires_at;comment:令牌过期时间" json:"expires_at"`
	RawData        string     `gorm:"type:json;comment:原始数据" json:"raw_data"`
	CreatedAt      time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (OAuthAccount) TableName() string {
	return "oauth_accounts"
}