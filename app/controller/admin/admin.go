package admin

import (
	"strconv"

	"forxi.cn/forxi-go/app/service"
	"forxi.cn/forxi-go/app/util"

	"github.com/gin-gonic/gin"
)

type AdminController struct {
	userService *service.UserService
}

func NewAdminController() *AdminController {
	return &AdminController{
		userService: service.NewUserService(),
	}
}

func (c *AdminController) ListUsers(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	users, total, err := c.userService.ListUsers(page, pageSize)
	if err != nil {
		util.InternalServerError(ctx, err.Error())
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	meta := &util.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	util.SuccessWithPagination(ctx, users, meta)
}

func (c *AdminController) UpdateUserStatus(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		util.BadRequest(ctx, "无效的用户ID")
		return
	}

	var req struct {
		Status string `json:"status" validate:"required,oneof=active inactive banned"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.userService.UpdateUserStatus(id, req.Status); err != nil {
		util.InternalServerError(ctx, err.Error())
		return
	}

	util.SuccessWithMessage(ctx, "用户状态更新成功", nil)
}

func (c *AdminController) UpdateUserRole(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		util.BadRequest(ctx, "无效的用户ID")
		return
	}

	var req struct {
		Role string `json:"role" validate:"required,oneof=user admin super_admin"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.BadRequest(ctx, "请求参数无效")
		return
	}

	if err := util.ValidateStruct(&req); err != nil {
		util.BadRequest(ctx, err.Error())
		return
	}

	if err := c.userService.UpdateUserRole(id, req.Role); err != nil {
		util.InternalServerError(ctx, err.Error())
		return
	}

	util.SuccessWithMessage(ctx, "用户角色更新成功", nil)
}
