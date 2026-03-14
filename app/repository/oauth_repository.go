package repository

import (
	"forxi.cn/forxi-go/app/model"
	"forxi.cn/forxi-go/app/resource"
	"gorm.io/gorm"
)

// OAuthRepository OAuth账号仓储
type OAuthRepository struct {
	db *gorm.DB
}

// NewOAuthRepository 创建OAuth仓储实例
func NewOAuthRepository() *OAuthRepository {
	return &OAuthRepository{
		db: resource.DB,
	}
}

// Create 创建OAuth账号
func (r *OAuthRepository) Create(oauthAccount *model.OAuthAccount) error {
	return r.db.Create(oauthAccount).Error
}

// FindByUserID 根据用户ID查找OAuth账号列表
func (r *OAuthRepository) FindByUserID(userID int64) ([]model.OAuthAccount, error) {
	var oauthAccounts []model.OAuthAccount
	err := r.db.Where("user_id = ?", userID).Find(&oauthAccounts).Error
	return oauthAccounts, err
}

// FindByUserIDAndProvider 根据用户ID和提供商查找OAuth账号
func (r *OAuthRepository) FindByUserIDAndProvider(userID int64, provider string) (*model.OAuthAccount, error) {
	var oauthAccount model.OAuthAccount
	err := r.db.Where("user_id = ? AND provider = ?", userID, provider).First(&oauthAccount).Error
	return &oauthAccount, err
}

// FindByProviderUserID 根据提供商和第三方用户ID查找OAuth账号
func (r *OAuthRepository) FindByProviderUserID(provider, providerUserID string) (*model.OAuthAccount, error) {
	var oauthAccount model.OAuthAccount
	err := r.db.Where("provider = ? AND provider_user_id = ?", provider, providerUserID).First(&oauthAccount).Error
	return &oauthAccount, err
}

// UpdateToken 更新OAuth账号的令牌
func (r *OAuthRepository) UpdateToken(id int64, accessToken, refreshToken string, expiresAt interface{}) error {
	return r.db.Model(&model.OAuthAccount{}).Where("id = ?", id).Updates(map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_at":    expiresAt,
	}).Error
}

// Delete 删除OAuth账号
func (r *OAuthRepository) Delete(id int64) error {
	return r.db.Delete(&model.OAuthAccount{}, id).Error
}
