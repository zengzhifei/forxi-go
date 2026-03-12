package util

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 密码加密
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 密码验证
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePasswordStrength 验证密码强度
// 要求：8-20位，包含大写、小写、数字、特殊符号中至少3种
func ValidatePasswordStrength(password string) error {
	length := len(password)
	
	// 检查长度
	if length < 8 || length > 20 {
		return errors.New("password must be between 8 and 20 characters")
	}
	
	// 统计包含的字符类型
	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	// 计算满足的类型数量
	typeCount := 0
	if hasUpper {
		typeCount++
	}
	if hasLower {
		typeCount++
	}
	if hasDigit {
		typeCount++
	}
	if hasSpecial {
		typeCount++
	}
	
	// 至少包含3种类型
	if typeCount < 3 {
		return errors.New("password must contain at least 3 types of characters (uppercase, lowercase, digits, special characters)")
	}
	
	return nil
}