package repository

import (
	"forxi.cn/forxi-go/app/model"
	"forxi.cn/forxi-go/app/resource"
	"gorm.io/gorm"
)

// LoginLogRepository 登录日志数据访问层
type LoginLogRepository struct {
	db *gorm.DB
}

// NewLoginLogRepository 创建登录日志仓库实例
func NewLoginLogRepository() *LoginLogRepository {
	return &LoginLogRepository{
		db: resource.DB,
	}
}

// Create 创建登录日志
func (r *LoginLogRepository) Create(log *model.LoginLog) error {
	return r.db.Create(log).Error
}

// FindByUserID 根据用户ID查找登录日志
func (r *LoginLogRepository) FindByUserID(userID int64, offset, limit int) ([]model.LoginLog, error) {
	var logs []model.LoginLog
	err := r.db.Where("user_id = ?", userID).
		Order("login_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// CountByUserID 统计用户登录次数
func (r *LoginLogRepository) CountByUserID(userID int64) (int64, error) {
	var count int64
	err := r.db.Model(&model.LoginLog{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountFailedByIP 统计IP失败登录次数（最近5分钟）
func (r *LoginLogRepository) CountFailedByIP(ipAddress string) (int64, error) {
	var count int64
	err := r.db.Model(&model.LoginLog{}).
		Where("ip_address = ? AND status = ? AND login_at >= DATE_SUB(NOW(), INTERVAL 5 MINUTE)", ipAddress, "failed").
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
