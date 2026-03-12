package model

import (
	"time"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID            int64          `gorm:"primaryKey;autoIncrement;comment:自增主键" json:"-"`
	UserID        int64          `gorm:"type:bigint;not null;uniqueIndex:uk_user_id;comment:用户全局唯一ID" json:"user_id"`
	Email         string         `gorm:"type:varchar(255);not null;uniqueIndex:uk_email;comment:邮箱地址" json:"email"`
	PasswordHash  string         `gorm:"type:varchar(255);not null;default:'';comment:密码哈希" json:"-"`
	Nickname      string         `gorm:"type:varchar(100);not null;default:'';comment:用户昵称" json:"nickname"`
	Avatar        string         `gorm:"type:varchar(500);not null;default:'';comment:用户头像URL" json:"avatar"`
	Bio           string         `gorm:"type:text;comment:用户简介" json:"bio"`
	Role          string         `gorm:"type:varchar(20);not null;default:'user';index:idx_role;comment:用户角色: user/admin/super_admin" json:"role"`
	EmailVerified bool           `gorm:"not null;default:false;index:idx_email_verified;comment:邮箱是否已验证: 0-未验证 1-已验证" json:"email_verified"`
	Status        string         `gorm:"type:varchar(20);not null;default:'active';index:idx_status;comment:账号状态: active/inactive/banned" json:"status"`
	CreatedAt     time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_created_at;comment:创建时间" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index:idx_deleted_at;comment:删除时间" json:"-"`
	
	// 关联
	OAuthAccounts []OAuthAccount `gorm:"foreignKey:UserID;references:UserID;comment:第三方账号" json:"-"`
	LoginLogs     []LoginLog     `gorm:"foreignKey:UserID;references:UserID;comment:登录日志" json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}