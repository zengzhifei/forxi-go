package email

import (
	"context"
	"errors"
	"fmt"
	"time"

	"forxi.cn/forxi-go/app/config"
	"forxi.cn/forxi-go/app/resource/rds"
	"forxi.cn/forxi-go/app/util"

	"github.com/redis/go-redis/v9"
)

// EmailService 邮件服务
type EmailService struct {
	emailConfig *config.EmailConfig
	redisClient *redis.Client
}

// NewEmailService 创建邮件服务实例
func InitEmailService(emailConfig *config.EmailConfig, redisClient *redis.Client) *EmailService {
	return &EmailService{
		emailConfig: emailConfig,
		redisClient: redisClient,
	}
}

// SendResetPasswordLink 发送密码重置邮件（带链接）
func (s *EmailService) SendResetPasswordLink(email, resetLink string) error {
	return util.SendVerificationCode(s.emailConfig, email, "", "reset_password", resetLink)
}

// SendRegisterCode 发送注册验证码
func (s *EmailService) SendRegisterCode(email string) error {
	ctx := context.Background()

	// 检查发送频率限制（1分钟内只能发送1次）
	limitKey := rds.Key(rds.KeyEmailLimit, email)
	exists, err := s.redisClient.Exists(ctx, limitKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	if exists > 0 {
		return errors.New("verification code sent too frequently, please try again later")
	}

	// 生成6位随机验证码
	code := util.GenerateVerificationCode()

	// 存储到Redis，10分钟过期
	verifyKey := rds.Key(rds.KeyEmailVerifyReg, email)
	err = s.redisClient.Set(ctx, verifyKey, code, 10*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to store verification code: %w", err)
	}

	// 发送邮件
	err = util.SendVerificationCode(s.emailConfig, email, code, "register")
	if err != nil {
		// 发送失败，删除Redis中的验证码
		s.redisClient.Del(ctx, verifyKey)
		return fmt.Errorf("邮件发送失败: %w", err)
	}

	// 设置频率限制，1分钟过期
	err = s.redisClient.Set(ctx, limitKey, "1", 1*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to set rate limit: %w", err)
	}

	return nil
}

// VerifyRegisterCode 验证注册验证码
func (s *EmailService) VerifyRegisterCode(email, code string) (bool, error) {
	ctx := context.Background()

	// 从Redis获取验证码
	verifyKey := rds.Key(rds.KeyEmailVerifyReg, email)
	storedCode, err := s.redisClient.Get(ctx, verifyKey).Result()
	if err != nil {
		if err == redis.Nil {
			return false, errors.New("verification code not found or expired")
		}
		return false, fmt.Errorf("failed to get verification code: %w", err)
	}

	// 比对验证码
	if storedCode != code {
		return false, errors.New("invalid verification code")
	}

	// 验证通过，删除验证码
	err = s.redisClient.Del(ctx, verifyKey).Err()
	if err != nil {
		return false, fmt.Errorf("failed to delete verification code: %w", err)
	}

	return true, nil
}

// SendResetPasswordCode 发送密码重置验证码
func (s *EmailService) SendResetPasswordCode(email string) error {
	ctx := context.Background()

	// 检查发送频率限制（1分钟内只能发送1次）
	limitKey := rds.Key(rds.KeyEmailLimit, email)
	exists, err := s.redisClient.Exists(ctx, limitKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	if exists > 0 {
		return errors.New("verification code sent too frequently, please try again later")
	}

	// 生成6位随机验证码
	code := util.GenerateVerificationCode()

	// 存储到Redis，10分钟过期
	verifyKey := rds.Key(rds.KeyEmailVerifyReset, email)
	err = s.redisClient.Set(ctx, verifyKey, code, 10*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to store verification code: %w", err)
	}

	// 发送邮件
	err = util.SendVerificationCode(s.emailConfig, email, code, "reset_password")
	if err != nil {
		// 发送失败，删除Redis中的验证码
		s.redisClient.Del(ctx, verifyKey)
		return fmt.Errorf("failed to send email: %w", err)
	}

	// 设置频率限制，1分钟过期
	err = s.redisClient.Set(ctx, limitKey, "1", 1*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to set rate limit: %w", err)
	}

	return nil
}

// VerifyResetPasswordCode 验证密码重置验证码
func (s *EmailService) VerifyResetPasswordCode(email, code string) (bool, error) {
	ctx := context.Background()

	// 从Redis获取验证码
	verifyKey := rds.Key(rds.KeyEmailVerifyReset, email)
	storedCode, err := s.redisClient.Get(ctx, verifyKey).Result()
	if err != nil {
		if err == redis.Nil {
			return false, errors.New("verification code not found or expired")
		}
		return false, fmt.Errorf("failed to get verification code: %w", err)
	}

	// 比对验证码
	if storedCode != code {
		return false, errors.New("invalid verification code")
	}

	// 验证通过，删除验证码
	err = s.redisClient.Del(ctx, verifyKey).Err()
	if err != nil {
		return false, fmt.Errorf("failed to delete verification code: %w", err)
	}

	return true, nil
}
