package util

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// GenerateEmailVerifyToken 生成邮箱验证令牌
func GenerateEmailVerifyToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ParseTokenExpiration 解析令牌过期时间
func ParseTokenExpiration(expireTime int64) time.Time {
	return time.Now().Add(time.Duration(expireTime) * time.Second)
}