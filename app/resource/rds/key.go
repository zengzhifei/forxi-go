package rds

import "fmt"

const (
	// KeyPrefix 全局Redis键前缀
	Prefix = "forxi"
)

const (
	// 邮件验证码相关
	KeyEmailLimit       = "email:limit:%s"
	KeyEmailVerifyReg   = "email:verify:register:%s"
	KeyEmailVerifyReset = "email:verify:reset:%s"

	// 密码重置相关
	KeyPasswordResetToken = "password:reset:token:%s"
	KeyPasswordResetEmail = "password:reset:email:%s"

	// OAuth相关
	KeyOAuthBind = "oauth:bind:%s"

	// 文件预览缓存
	KeyFilePreviewCache = "filepreview:cache:%s"
)

// Key 生成Redis键名
func Key(format string, parts ...any) string {
	return fmt.Sprintf("%s:%s", Prefix, fmt.Sprintf(format, parts...))
}
