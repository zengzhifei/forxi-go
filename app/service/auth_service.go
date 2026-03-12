package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"forxi.cn/forxi-go/app/config"
	"forxi.cn/forxi-go/app/database"
	"forxi.cn/forxi-go/app/middleware"
	"forxi.cn/forxi-go/app/model"
	"forxi.cn/forxi-go/app/repository"
	"forxi.cn/forxi-go/app/util"
	"go.uber.org/zap"

	"github.com/redis/go-redis/v9"
)

// AuthService 认证服务层
type AuthService struct {
	userRepo     *repository.UserRepository
	loginLogRepo *repository.LoginLogRepository
	redisClient  *redis.Client
	redisPrefix  string // Redis key前缀
	rateLimiter  *middleware.IPRateLimiter
	config       *config.JWTConfig
	emailService *EmailService
	oauthConfig  *config.OAuthConfig
}

// NewAuthService 创建认证服务实例
func NewAuthService(cfg *config.JWTConfig, redisConfig *config.RedisConfig, rateLimiter *middleware.IPRateLimiter, emailService *EmailService, oauthConfig *config.OAuthConfig) *AuthService {
	return &AuthService{
		userRepo:     repository.NewUserRepository(),
		loginLogRepo: repository.NewLoginLogRepository(),
		redisClient:  database.GetRedis(),
		redisPrefix:  redisConfig.Prefix,
		rateLimiter:  rateLimiter,
		config:       cfg,
		emailService: emailService,
		oauthConfig:  oauthConfig,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User   *model.User     `json:"user"`
	Tokens *util.TokenPair `json:"tokens"`
}

// Login 用户登录
func (s *AuthService) Login(req *LoginRequest, ipAddress, userAgent, deviceType string) (*LoginResponse, error) {
	// 查找用户
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		s.LogLoginAttempt(0, ipAddress, userAgent, deviceType, "email", "failed")
		return nil, errors.New("invalid email or password")
	}

	// 检查账号状态
	if user.Status != "active" {
		s.LogLoginAttempt(user.UserID, ipAddress, userAgent, deviceType, "email", "failed")
		return nil, errors.New("account is inactive or banned")
	}

	// 验证密码
	if !util.CheckPassword(req.Password, user.PasswordHash) {
		s.LogLoginAttempt(user.UserID, ipAddress, userAgent, deviceType, "email", "failed")
		return nil, errors.New("invalid email or password")
	}

	// 生成令牌
	tokens, err := util.GenerateToken(user.UserID, user.Email, user.Role, s.config.Secret, s.config.AccessExpire, s.config.RefreshExpire)
	if err != nil {
		return nil, err
	}

	// 记录登录日志
	s.LogLoginAttempt(user.UserID, ipAddress, userAgent, deviceType, "email", "success")

	return &LoginResponse{
		User:   user,
		Tokens: tokens,
	}, nil
}

// LogLoginAttempt 记录登录尝试
func (s *AuthService) LogLoginAttempt(userID int64, ipAddress, userAgent, deviceType, loginMethod, status string) {
	loginLog := &model.LoginLog{
		UserID:      userID,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		DeviceType:  deviceType,
		LoginMethod: loginMethod,
		Status:      status,
	}
	s.loginLogRepo.Create(loginLog)
}

// RefreshToken 刷新令牌
func (s *AuthService) RefreshToken(refreshToken string) (*util.TokenPair, error) {
	// 解析刷新令牌
	claims, err := util.ParseToken(refreshToken, s.config.Secret)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// 查找用户
	user, err := s.userRepo.FindByUserID(claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 检查账号状态
	if user.Status != "active" {
		return nil, errors.New("account is inactive or banned")
	}

	// 生成新令牌
	tokens, err := util.GenerateToken(user.UserID, user.Email, user.Role, s.config.Secret, s.config.AccessExpire, s.config.RefreshExpire)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// RequestPasswordReset 请求密码重置（使用Redis存储临时token）
func (s *AuthService) RequestPasswordReset(email string) error {
	ctx := context.Background()

	// 查找用户
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// 为了安全，即使邮箱不存在也返回成功
		return nil
	}

	// 生成重置令牌
	token := util.GenerateEmailVerifyToken()

	// 存储到Redis：token -> user_id，15分钟过期
	tokenKey := fmt.Sprintf("%s:password:reset:token:%s", s.redisPrefix, token)
	err = s.redisClient.Set(ctx, tokenKey, user.UserID, 15*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	// 同时存储：email -> token（用于防止重复请求）
	emailKey := fmt.Sprintf("%s:password:reset:email:%s", s.redisPrefix, email)
	err = s.redisClient.Set(ctx, emailKey, token, 15*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to store email mapping: %w", err)
	}

	// 生成重置链接
	resetLink := fmt.Sprintf("%s?token=%s&email=%s", s.oauthConfig.PasswordResetURL, token, email)

	// 发送重置邮件
	err = s.emailService.SendResetPasswordLink(email, resetLink)
	if err != nil {
		middleware.Logger.Error("Failed to send password reset email", zap.String("email", email), zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// ResetPassword 重置密码（从Redis验证token）
func (s *AuthService) ResetPassword(token, newPassword string) error {
	ctx := context.Background()

	// 从Redis获取user_id
	tokenKey := fmt.Sprintf("%s:password:reset:token:%s", s.redisPrefix, token)
	userIDStr, err := s.redisClient.Get(ctx, tokenKey).Result()
	if err != nil {
		if err == redis.Nil {
			return errors.New("invalid or expired token")
		}
		return fmt.Errorf("failed to get reset token: %w", err)
	}

	// 解析user_id
	var userID int64
	_, err = fmt.Sscanf(userIDStr, "%d", &userID)
	if err != nil {
		return errors.New("invalid token format")
	}

	// 查找用户
	user, err := s.userRepo.FindByUserID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// 检查账号状态
	if user.Status != "active" {
		return errors.New("account is inactive or banned")
	}

	// 加密新密码
	passwordHash, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// 更新用户密码
	if err := s.userRepo.UpdateFields(userID, map[string]interface{}{
		"password_hash": passwordHash,
	}); err != nil {
		return err
	}

	// 删除已使用的token（防止重复使用）
	s.redisClient.Del(ctx, tokenKey)

	// 删除email映射
	emailKey := fmt.Sprintf("%s:password:reset:email:%s", s.redisPrefix, user.Email)
	s.redisClient.Del(ctx, emailKey)

	return nil
}

// Logout 登出（客户端删除令牌即可，如需服务端黑名单可实现）
func (s *AuthService) Logout(userID int64) error {
	// 当前实现为无状态JWT，登出由客户端删除令牌
	// 如需服务端登出，可实现令牌黑名单机制
	return nil
}

// CheckLoginAttempts 检查登录尝试次数
func (s *AuthService) CheckLoginAttempts(ipAddress string) (bool, error) {
	count, err := s.loginLogRepo.CountFailedByIP(ipAddress)
	if err != nil {
		return false, err
	}
	return count < 5, nil
}

// GetLoginLogs 获取用户登录日志
func (s *AuthService) GetLoginLogs(userID int64, page, pageSize int) ([]model.LoginLog, error) {
	offset := (page - 1) * pageSize
	return s.loginLogRepo.FindByUserID(userID, offset, pageSize)
}
