package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"forxi.cn/forxi-go/app/config"
	"forxi.cn/forxi-go/app/model"
	"forxi.cn/forxi-go/app/repository"
	"forxi.cn/forxi-go/app/resource"
	"forxi.cn/forxi-go/app/resource/rds"
	"forxi.cn/forxi-go/app/util"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// OAuthService OAuth服务层
type OAuthService struct {
	userRepo    *repository.UserRepository
	oauthRepo   *repository.OAuthRepository
	redisClient *redis.Client
	config      *config.OAuthConfig
	jwtConfig   *config.JWTConfig
}

// NewOAuthService 创建OAuth服务实例
func NewOAuthService() *OAuthService {
	return &OAuthService{
		userRepo:    repository.NewUserRepository(),
		oauthRepo:   repository.NewOAuthRepository(),
		redisClient: resource.Redis,
		config:      &resource.Cfg.OAuth,
		jwtConfig:   &resource.Cfg.JWT,
	}
}

// GenerateOAuthState 生成 OAuth 状态（使用 JWT，包含 user_id）
func (s *OAuthService) GenerateOAuthState(userID int64) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ID:        strconv.FormatInt(userID, 10),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtConfig.Secret))
}

// ParseOAuthState 解析 OAuth 状态
func (s *OAuthService) ParseOAuthState(state string) (*int64, error) {
	token, err := jwt.ParseWithClaims(state, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtConfig.Secret), nil
	})
	if err != nil {
		return nil, nil
	}
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		if claims.ID != "" {
			userID, err := strconv.ParseInt(claims.ID, 10, 64)
			if err != nil {
				return nil, nil
			}
			return &userID, nil
		}
	}
	return nil, nil
}

// OAuthResult OAuth处理结果
type OAuthResult struct {
	IsNewUser      bool   // 是否新用户（未绑定邮箱）
	NeedsEmailBind bool   // 是否需要绑定邮箱
	Provider       string // OAuth提供商
	Nickname       string // 用户昵称
	Avatar         string // 用户头像
	Bio            string // 用户简介
	BindToken      string // 绑定令牌（用于绑定邮箱）
}

// GitHubUserInfo GitHub用户信息
type GitHubUserInfo struct {
	ID       int64  `json:"id"`
	Login    string `json:"login"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar_url"`
	Bio      string `json:"bio"`
	Location string `json:"location"`
}

// GitHubBindInfo GitHub绑定信息（存储在Redis）
type GitHubBindInfo struct {
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	Nickname       string `json:"nickname"`
	Avatar         string `json:"avatar"`
	Bio            string `json:"bio"`
	RawData        string `json:"raw_data"`
}

// GetGitHubOAuthConfig 获取GitHub OAuth配置
func (s *OAuthService) GetGitHubOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.config.GitHub.ClientID,
		ClientSecret: s.config.GitHub.ClientSecret,
		RedirectURL:  s.config.GitHub.RedirectURL,
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
}

// GetGitHubAuthURL 获取GitHub授权URL
func (s *OAuthService) GetGitHubAuthURL(state string) string {
	oauthConfig := s.GetGitHubOAuthConfig()
	return oauthConfig.AuthCodeURL(state)
}

// HandleGitHubCallback 处理GitHub回调
// 返回值：user, tokens, isBind(是否绑定操作), error
func (s *OAuthService) HandleGitHubCallback(code string, currentUserID *int64) (*model.User, *util.TokenPair, bool, error) {
	oauthConfig := s.GetGitHubOAuthConfig()

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, nil, false, errors.New("failed to exchange token: " + err.Error())
	}

	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get(s.config.GitHub.UserInfoURL)
	if err != nil {
		return nil, nil, false, errors.New("failed to get user info: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, false, errors.New("failed to get user info")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, false, errors.New("failed to read response body")
	}

	var userInfo GitHubUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, nil, false, errors.New("failed to parse user info")
	}

	providerUserID := strconv.FormatInt(userInfo.ID, 10)

	oauthAccount, err := s.oauthRepo.FindByProviderUserID("github", providerUserID)
	if err == nil {
		user, err := s.userRepo.FindByUserID(oauthAccount.UserID)
		if err != nil {
			return nil, nil, false, err
		}

		var expiresAt *time.Time
		if !token.Expiry.IsZero() {
			expiresAt = &token.Expiry
		}
		s.oauthRepo.UpdateToken(oauthAccount.ID, token.AccessToken, token.RefreshToken, expiresAt)

		tokens, err := util.GenerateToken(
			user.UserID,
			user.Email,
			user.Role,
			s.jwtConfig.Secret,
			s.jwtConfig.AccessExpire,
			s.jwtConfig.RefreshExpire,
		)
		if err != nil {
			return nil, nil, false, err
		}

		return user, tokens, false, nil
	}

	nickname := userInfo.Name
	if nickname == "" {
		nickname = userInfo.Login
	}

	if currentUserID != nil {
		user, err := s.userRepo.FindByUserID(*currentUserID)
		if err != nil {
			return nil, nil, false, errors.New("user not found")
		}

		now := time.Now()
		var expiresAt *time.Time
		if !token.Expiry.IsZero() {
			expiresAt = &token.Expiry
		}
		newOAuthAccount := &model.OAuthAccount{
			UserID:         user.UserID,
			Provider:       "github",
			ProviderUserID: providerUserID,
			AccessToken:    token.AccessToken,
			RefreshToken:   token.RefreshToken,
			ExpiresAt:      expiresAt,
			RawData:        string(body),
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if err := s.oauthRepo.Create(newOAuthAccount); err != nil {
			return nil, nil, false, err
		}

		tokens, err := util.GenerateToken(
			user.UserID,
			user.Email,
			user.Role,
			s.jwtConfig.Secret,
			s.jwtConfig.AccessExpire,
			s.jwtConfig.RefreshExpire,
		)
		if err != nil {
			return nil, nil, false, err
		}

		return user, tokens, true, nil
	}

	bindToken := util.GenerateEmailVerifyToken()
	bindInfo := GitHubBindInfo{
		Provider:       "github",
		ProviderUserID: providerUserID,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		Nickname:       nickname,
		Avatar:         userInfo.Avatar,
		Bio:            userInfo.Bio,
		RawData:        string(body),
	}

	ctx := context.Background()
	bindKey := rds.Key(rds.KeyOAuthBind, bindToken)
	bindData, err := json.Marshal(bindInfo)
	if err != nil {
		return nil, nil, false, errors.New("failed to marshal bind info")
	}
	if err := s.redisClient.Set(ctx, bindKey, bindData, 10*time.Minute).Err(); err != nil {
		return nil, nil, false, errors.New("failed to store bind info: " + err.Error())
	}

	result := &OAuthResult{
		BindToken: bindToken,
	}
	return nil, nil, false, errors.New("needs_email_bind:" + result.BindToken)
}

// BindEmailWithGitHub 通过GitHub绑定邮箱（用于首次登录时绑定邮箱）
func (s *OAuthService) BindEmailWithGitHub(bindToken, email, emailCode, password string) (*model.User, *util.TokenPair, error) {
	ctx := context.Background()

	bindKey := rds.Key(rds.KeyOAuthBind, bindToken)
	bindData, err := s.redisClient.Get(ctx, bindKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil, errors.New("bind token expired or not found")
		}
		return nil, nil, errors.New("failed to get bind info")
	}

	var bindInfo GitHubBindInfo
	if err := json.Unmarshal([]byte(bindData), &bindInfo); err != nil {
		return nil, nil, errors.New("failed to parse bind info")
	}

	if bindInfo.Provider != "github" {
		return nil, nil, errors.New("invalid bind token")
	}

	codeKey := rds.Key(rds.KeyEmailVerifyReg, email)
	codeValue, err := s.redisClient.Get(ctx, codeKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil, errors.New("email code expired")
		}
		return nil, nil, errors.New("failed to verify email code")
	}

	if codeValue != emailCode {
		return nil, nil, errors.New("invalid email code")
	}

	s.redisClient.Del(ctx, codeKey)

	exists, err := s.userRepo.ExistsByEmail(email)
	if err != nil {
		return nil, nil, err
	}

	if exists {
		existingUser, err := s.userRepo.FindByEmail(email)
		if err != nil {
			return nil, nil, err
		}

		if !util.CheckPassword(password, existingUser.PasswordHash) {
			return nil, nil, errors.New("password is incorrect")
		}

		now := time.Now()
		newOAuthAccount := &model.OAuthAccount{
			UserID:         existingUser.UserID,
			Provider:       "github",
			ProviderUserID: bindInfo.ProviderUserID,
			AccessToken:    bindInfo.AccessToken,
			RefreshToken:   bindInfo.RefreshToken,
			RawData:        bindInfo.RawData,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if err := s.oauthRepo.Create(newOAuthAccount); err != nil {
			return nil, nil, err
		}

		s.redisClient.Del(ctx, bindKey)

		tokens, err := util.GenerateToken(
			existingUser.UserID,
			existingUser.Email,
			existingUser.Role,
			s.jwtConfig.Secret,
			s.jwtConfig.AccessExpire,
			s.jwtConfig.RefreshExpire,
		)
		if err != nil {
			return nil, nil, err
		}

		return existingUser, tokens, nil
	}

	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		return nil, nil, err
	}

	userID := resource.ShortSnowflake.NextID()
	user := &model.User{
		UserID:        userID,
		Email:         email,
		PasswordHash:  hashedPassword,
		Nickname:      bindInfo.Nickname,
		Avatar:        bindInfo.Avatar,
		Bio:           bindInfo.Bio,
		Role:          "user",
		Status:        "active",
		EmailVerified: true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, nil, err
	}

	now := time.Now()
	newOAuthAccount := &model.OAuthAccount{
		UserID:         userID,
		Provider:       "github",
		ProviderUserID: bindInfo.ProviderUserID,
		AccessToken:    bindInfo.AccessToken,
		RefreshToken:   bindInfo.RefreshToken,
		RawData:        bindInfo.RawData,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.oauthRepo.Create(newOAuthAccount); err != nil {
		return nil, nil, err
	}

	s.redisClient.Del(ctx, bindKey)

	tokens, err := util.GenerateToken(
		user.UserID,
		user.Email,
		user.Role,
		s.jwtConfig.Secret,
		s.jwtConfig.AccessExpire,
		s.jwtConfig.RefreshExpire,
	)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// UnbindOAuthAccount 解绑第三方账号
func (s *OAuthService) UnbindOAuthAccount(userID int64, provider string) error {
	oauthAccount, err := s.oauthRepo.FindByUserIDAndProvider(userID, provider)
	if err != nil {
		return errors.New("OAuth account not found")
	}

	if err := s.oauthRepo.Delete(oauthAccount.ID); err != nil {
		return err
	}

	return nil
}

// GetOAuthAccounts 获取用户的OAuth账号列表
func (s *OAuthService) GetOAuthAccounts(userID int64) ([]model.OAuthAccount, error) {
	return s.oauthRepo.FindByUserID(userID)
}
