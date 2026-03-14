package user

import (
	"forxi.cn/forxi-go/app/service"
	"forxi.cn/forxi-go/app/util"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService  *service.UserService
	emailService *service.EmailService
}

func NewUserController(emailService *service.EmailService) *UserController {
	return &UserController{
		userService:  service.NewUserService(emailService),
		emailService: emailService,
	}
}

func (c *UserController) Register(ctx *gin.Context) {
	var req service.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	user, err := c.userService.Register(&req)
	if err != nil {
		if err.Error() == "email already exists" {
			util.BadRequest(ctx, "邮箱已存在")
			return
		}
		util.InternalServerError(ctx, err.Error())
		return
	}

	util.Success(ctx, user)
}

func (c *UserController) GetProfile(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		util.Unauthorized(ctx, "用户未登录")
		return
	}

	user, err := c.userService.GetProfile(userID.(int64))
	if err != nil {
		util.NotFound(ctx, "用户不存在")
		return
	}

	util.Success(ctx, user)
}

func (c *UserController) UpdateProfile(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		util.Unauthorized(ctx, "用户未登录")
		return
	}

	var req service.UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	user, err := c.userService.UpdateProfile(userID.(int64), &req)
	if err != nil {
		util.InternalServerError(ctx, err.Error())
		return
	}

	util.Success(ctx, user)
}

func (c *UserController) ChangePassword(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		util.Unauthorized(ctx, "用户未登录")
		return
	}

	var req struct {
		OldPassword string `json:"old_password" validate:"required"`
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

	err := c.userService.ChangePassword(userID.(int64), req.OldPassword, req.NewPassword)
	if err != nil {
		if err.Error() == "old password is incorrect" {
			util.BadRequest(ctx, "旧密码不正确")
			return
		}
		util.InternalServerError(ctx, err.Error())
		return
	}

	util.SuccessWithMessage(ctx, "密码修改成功", nil)
}

func (c *UserController) SendRegisterCode(ctx *gin.Context) {
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

	err := c.emailService.SendRegisterCode(req.Email)
	if err != nil {
		if err.Error() == "verification code sent too frequently, please try again later" {
			util.TooManyRequests(ctx, "验证码发送过于频繁，请稍后再试")
			return
		}
		util.InternalServerError(ctx, err.Error())
		return
	}

	util.SuccessWithMessage(ctx, "验证码发送成功", nil)
}
