package user

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"forxi.cn/forxi-go/app/config"
	"forxi.cn/forxi-go/app/service"
	"forxi.cn/forxi-go/app/util"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService         *service.AuthService
	oauthService        *service.OAuthService
	frontendCallbackURL string
}

func NewAuthController(authService *service.AuthService, oauthCfg *config.OAuthConfig, oauthService *service.OAuthService) *AuthController {
	return &AuthController{
		authService:         authService,
		oauthService:        oauthService,
		frontendCallbackURL: oauthCfg.FrontendCallbackURL,
	}
}

func (c *AuthController) Login(ctx *gin.Context) {
	var req service.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	ipAddress := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")
	deviceType := detectDeviceType(userAgent)

	loginResp, err := c.authService.Login(&req, ipAddress, userAgent, deviceType)
	if err != nil {
		if err.Error() == "invalid email or password" {
			util.BadRequest(ctx, "邮箱或密码错误")
		} else {
			util.InternalServerError(ctx, err.Error())
		}
		return
	}

	util.Success(ctx, loginResp)
}

func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	tokens, err := c.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		util.Unauthorized(ctx, "令牌无效或已过期")
		return
	}

	util.Success(ctx, tokens)
}

func (c *AuthController) Logout(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		util.Unauthorized(ctx, "User not authenticated")
		return
	}

	err := c.authService.Logout(userID.(int64))
	if err != nil {
		util.InternalServerError(ctx, err.Error())
		return
	}

	util.SuccessWithMessage(ctx, "退出登录成功", nil)
}

func (c *AuthController) RequestPasswordReset(ctx *gin.Context) {
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	err := c.authService.RequestPasswordReset(req.Email)
	if err != nil {
		util.InternalServerError(ctx, err.Error())
		return
	}

	util.SuccessWithMessage(ctx, "邮件已发送", nil)
}

func (c *AuthController) ResetPassword(ctx *gin.Context) {
	var req struct {
		Token       string `json:"token" validate:"required"`
		NewPassword string `json:"new_password" validate:"required,min=8"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	err := c.authService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "invalid or expired token" {
			util.BadRequest(ctx, "令牌无效或已过期")
		} else if errMsg == "user not found" {
			util.BadRequest(ctx, "用户不存在")
		} else {
			util.InternalServerError(ctx, errMsg)
		}
		return
	}

	util.SuccessWithMessage(ctx, "密码重置成功", nil)
}

func (c *AuthController) GetLoginLogs(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		util.Unauthorized(ctx, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	logs, err := c.authService.GetLoginLogs(userID.(int64), page, pageSize)
	if err != nil {
		util.InternalServerError(ctx, err.Error())
		return
	}

	util.Success(ctx, logs)
}

func detectDeviceType(userAgent string) string {
	userAgent = strings.ToLower(userAgent)
	if strings.Contains(userAgent, "mobile") || strings.Contains(userAgent, "android") || strings.Contains(userAgent, "iphone") || strings.Contains(userAgent, "ipad") {
		return "mobile"
	}
	if strings.Contains(userAgent, "tablet") || strings.Contains(userAgent, "ipad") {
		return "tablet"
	}
	return "desktop"
}

func (c *AuthController) GetGitHubAuthURL(ctx *gin.Context) {
	state := generateState()

	userID, exists := ctx.Get("user_id")
	if exists {
		oauthState, _ := c.oauthService.GenerateOAuthState(userID.(int64))
		state = oauthState
	}

	authURL := c.oauthService.GetGitHubAuthURL(state)

	util.Success(ctx, map[string]string{
		"auth_url": authURL,
	})
}

func (c *AuthController) GitHubLogin(ctx *gin.Context) {
	code := ctx.Query("code")
	state := ctx.Query("state")
	if code == "" {
		redirectURL := c.frontendCallbackURL + "?error=授权码缺失"
		ctx.Redirect(302, redirectURL)
		return
	}

	currentUserID, _ := c.oauthService.ParseOAuthState(state)

	user, tokens, isBind, err := c.oauthService.HandleGitHubCallback(code, currentUserID)
	if err != nil {
		errMsg := err.Error()
		if strings.HasPrefix(errMsg, "needs_email_bind:") {
			bindToken := strings.TrimPrefix(errMsg, "needs_email_bind:")
			redirectURL := c.frontendCallbackURL + "?needs_email_bind=true&bind_token=" + bindToken
			ctx.Redirect(302, redirectURL)
			return
		}
		redirectURL := c.frontendCallbackURL + "?error=" + errMsg
		ctx.Redirect(302, redirectURL)
		return
	}

	redirectURL := c.frontendCallbackURL + "?access_token=" + tokens.AccessToken +
		"&refresh_token=" + tokens.RefreshToken +
		"&bind_type=" + map[bool]string{true: "bind", false: "login"}[isBind]

	ipAddress := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")
	c.authService.LogLoginAttempt(user.UserID, ipAddress, userAgent, detectDeviceType(userAgent), "GitHub", "success")

	ctx.Redirect(302, redirectURL)
}

func (c *AuthController) BindGitHubEmail(ctx *gin.Context) {
	var req struct {
		BindToken       string `json:"bind_token" form:"bind_token" validate:"required"`
		Email           string `json:"email" form:"email" validate:"required,email"`
		EmailCode       string `json:"email_code" form:"email_code" validate:"required"`
		Password        string `json:"password" form:"password" validate:"required,min=8"`
		ConfirmPassword string `json:"confirm_password" form:"confirm_password" validate:"required"`
	}

	if err := ctx.ShouldBind(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if req.Password != req.ConfirmPassword {
		util.BadRequest(ctx, "两次输入的密码不一致")
		return
	}

	user, tokens, err := c.oauthService.BindEmailWithGitHub(req.BindToken, req.Email, req.EmailCode, req.Password)
	if err != nil {
		errMsg := err.Error()
		switch errMsg {
		case "bind token expired or not found":
			util.BadRequest(ctx, "绑定令牌已过期，请重新授权")
		case "invalid bind token":
			util.BadRequest(ctx, "无效的绑定令牌")
		case "email already bound to another account":
			util.BadRequest(ctx, "邮箱已被其他账号绑定")
		case "invalid email code":
			util.BadRequest(ctx, "邮箱验证码错误")
		case "email code expired":
			util.BadRequest(ctx, "邮箱验证码已过期")
		case "need password verification":
			util.BadRequest(ctx, "need_password_verification")
		default:
			util.InternalServerError(ctx, errMsg)
		}
		return
	}

	ipAddress := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")
	c.authService.LogLoginAttempt(user.UserID, ipAddress, userAgent, detectDeviceType(userAgent), "GitHub", "success")

	loginResp := &service.LoginResponse{
		User:   user,
		Tokens: tokens,
	}

	util.Success(ctx, loginResp)
}

func (c *AuthController) UnbindOAuthAccount(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		util.Unauthorized(ctx, "User not authenticated")
		return
	}

	var req struct {
		Provider string `json:"provider" validate:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	err := c.oauthService.UnbindOAuthAccount(userID.(int64), req.Provider)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "OAuth account not found" {
			util.BadRequest(ctx, "未找到绑定的账号")
		} else {
			util.InternalServerError(ctx, errMsg)
		}
		return
	}

	util.SuccessWithMessage(ctx, "解绑成功", nil)
}

func (c *AuthController) GetOAuthAccounts(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		util.Unauthorized(ctx, "User not authenticated")
		return
	}

	accounts, err := c.oauthService.GetOAuthAccounts(userID.(int64))
	if err != nil {
		util.InternalServerError(ctx, err.Error())
		return
	}

	type OAuthAccountResponse struct {
		Provider string `json:"provider"`
		Name     string `json:"name"`
		HTMLURL  string `json:"html_url"`
	}

	var response []OAuthAccountResponse
	for _, account := range accounts {
		resp := OAuthAccountResponse{
			Provider: account.Provider,
		}
		if account.RawData != "" {
			var rawData map[string]interface{}
			if err := json.Unmarshal([]byte(account.RawData), &rawData); err == nil {
				if name, ok := rawData["name"].(string); ok {
					resp.Name = name
				}
				if htmlURL, ok := rawData["html_url"].(string); ok {
					resp.HTMLURL = htmlURL
				}
			}
		}
		response = append(response, resp)
	}

	util.Success(ctx, response)
}

func generateState() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
