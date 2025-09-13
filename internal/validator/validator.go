package validator

import (
	"net/url"
	"path/filepath"

	"github.com/go-playground/validator/v10"
)

// 检查是否为有效的URL或文件路径
func ValidateUrlOrPath(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	_, err := url.ParseRequestURI(value)
	if err == nil {
		return true
	}

	return filepath.IsAbs(value)
}

var Validator *validator.Validate

func init() {
	Validator = validator.New()
}
