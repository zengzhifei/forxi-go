package service

import (
	"errors"

	"forxi.cn/forxi-go/app/model"
	"forxi.cn/forxi-go/app/repository"
	"forxi.cn/forxi-go/app/resource"
	"forxi.cn/forxi-go/app/resource/email"
	"forxi.cn/forxi-go/app/util"
)

// UserService 用户服务层
type UserService struct {
	userRepo     *repository.UserRepository
	emailService *email.EmailService
}

// NewUserService 创建用户服务实例
func NewUserService() *UserService {
	return &UserService{
		userRepo:     repository.NewUserRepository(),
		emailService: resource.EmailService,
	}
}

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Email            string `json:"email" validate:"required,email"`
	Password         string `json:"password" validate:"required,min=8"`
	Nickname         string `json:"nickname" validate:"required,min=2,max=50"`
	VerificationCode string `json:"verification_code" validate:"required,len=6"`
}

// UpdateProfileRequest 更新资料请求
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" validate:"omitempty,min=2,max=50"`
	Avatar   string `json:"avatar" validate:"omitempty,url"`
	Bio      string `json:"bio" validate:"omitempty,max=500"`
}

// Register 用户注册
func (s *UserService) Register(req *RegisterRequest) (*model.User, error) {
	// 验证密码强度
	if err := util.ValidatePasswordStrength(req.Password); err != nil {
		return nil, err
	}

	// 验证邮箱验证码
	valid, err := s.emailService.VerifyRegisterCode(req.Email, req.VerificationCode)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errors.New("invalid or expired verification code")
	}

	// 检查邮箱是否已存在
	exists, err := s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email already exists")
	}

	// 加密密码
	passwordHash, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 使用短雪花算法生成user_id（约11-13位数字）
	userID := resource.ShortSnowflake.NextID()

	// 创建用户（邮箱已通过验证码验证，设置email_verified为true）
	user := &model.User{
		UserID:        userID,
		Email:         req.Email,
		PasswordHash:  passwordHash,
		Nickname:      req.Nickname,
		Role:          "user",
		Status:        "active",
		EmailVerified: true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetProfile 获取用户资料
func (s *UserService) GetProfile(userID int64) (*model.User, error) {
	user, err := s.userRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateProfile 更新用户资料
func (s *UserService) UpdateProfile(userID int64, req *UpdateProfileRequest) (*model.User, error) {
	user, err := s.userRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Bio != "" {
		user.Bio = req.Bio
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

// VerifyEmail 验证邮箱
func (s *UserService) VerifyEmail(userID int64) error {
	return s.userRepo.UpdateFields(userID, map[string]interface{}{
		"email_verified": true,
	})
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(userID int64, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByUserID(userID)
	if err != nil {
		return err
	}

	if !util.CheckPassword(oldPassword, user.PasswordHash) {
		return errors.New("old password is incorrect")
	}

	newPasswordHash, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdateFields(userID, map[string]interface{}{
		"password_hash": newPasswordHash,
	})
}

// ListUsers 获取用户列表
func (s *UserService) ListUsers(page, pageSize int) ([]model.User, int64, error) {
	offset := (page - 1) * pageSize
	users, total, err := s.userRepo.List(offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UpdateUserStatus 更新用户状态
func (s *UserService) UpdateUserStatus(userID int64, status string) error {
	return s.userRepo.UpdateFields(userID, map[string]interface{}{
		"status": status,
	})
}

// UpdateUserRole 更新用户角色
func (s *UserService) UpdateUserRole(userID int64, role string) error {
	return s.userRepo.UpdateFields(userID, map[string]interface{}{
		"role": role,
	})
}
