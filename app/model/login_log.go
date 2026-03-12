package model

import (
	"time"
)

// LoginLog 登录日志模型
type LoginLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;comment:登录日志ID" json:"-"`
	UserID      int64     `gorm:"not null;default:0;index:idx_user_id;comment:用户ID" json:"user_id"`
	IPAddress   string    `gorm:"type:varchar(45);not null;default:'';index:idx_ip_address;comment:IP地址" json:"ip_address"`
	UserAgent   string    `gorm:"type:text;comment:用户代理信息" json:"user_agent"`
	DeviceType  string    `gorm:"type:varchar(50);not null;default:'';comment:设备类型: mobile/tablet/desktop" json:"device_type"`
	LoginAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_login_at;comment:登录时间" json:"login_at"`
	LoginMethod string    `gorm:"type:varchar(20);not null;default:'';index:idx_login_method;comment:登录方式: email/wechat/github" json:"login_method"`
	Status      string    `gorm:"type:varchar(20);not null;default:'success';index:idx_status;comment:登录状态: success/failed" json:"status"`
}

// TableName 指定表名
func (LoginLog) TableName() string {
	return "login_logs"
}