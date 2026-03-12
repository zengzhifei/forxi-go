package util

import (
	"errors"
	"fmt"
	"strings"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct 验证结构体
func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		return GetValidatorErrors(err)
	}
	return nil
}

// GetValidatorErrors 获取验证错误消息
func GetValidatorErrors(err error) error {
	var errorList []string

	for _, err := range err.(validator.ValidationErrors) {
		switch err.Tag() {
		case "required":
			errorList = append(errorList, fmt.Sprintf("%s 为必填项", err.Field()))
		case "email":
			errorList = append(errorList, fmt.Sprintf("%s 必须是有效的邮箱地址", err.Field()))
		case "min":
			errorList = append(errorList, fmt.Sprintf("%s 至少需要 %s 个字符", err.Field(), err.Param()))
		case "max":
			errorList = append(errorList, fmt.Sprintf("%s 最多 %s 个字符", err.Field(), err.Param()))
		case "url":
			errorList = append(errorList, fmt.Sprintf("%s 必须是有效的URL", err.Field()))
		case "len":
			errorList = append(errorList, fmt.Sprintf("%s 长度必须为 %s", err.Field(), err.Param()))
		default:
			errorList = append(errorList, fmt.Sprintf("%s 格式不正确", err.Field()))
		}
	}

	if len(errorList) > 0 {
		return errors.New("参数验证失败: " + strings.Join(errorList, ", "))
	}

	return nil
}