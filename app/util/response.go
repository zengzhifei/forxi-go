package util

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse 分页响应格式
type PaginatedResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    interface{}     `json:"data"`
	Meta    *PaginationMeta `json:"meta,omitempty"`
}

// PaginationMeta 分页元数据
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// Success 返回成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 返回带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// SuccessWithPagination 返回分页成功响应
func SuccessWithPagination(c *gin.Context, data interface{}, meta *PaginationMeta) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Meta:    meta,
	})
}

// Error 返回错误响应
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest 返回400错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, 400, message)
}

// Unauthorized 返回401错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, 401, message)
}

// Forbidden 返回403错误
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, 403, message)
}

// NotFound 返回404错误
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, 404, message)
}

// TooManyRequests 返回429错误
func TooManyRequests(c *gin.Context, message string) {
	Error(c, http.StatusTooManyRequests, 429, message)
}

// InternalServerError 返回500错误
func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, 500, message)
}

// Download 返回文件下载响应
func Download(c *gin.Context, filePath, filename, contentType string) {
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline; filename="+filepath.Base(filename))
	c.File(filePath)
}
