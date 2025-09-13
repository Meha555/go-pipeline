package info

import (
	"encoding/json"
	"fmt"
	check "go-pipeline/internal/validator"
)

// SourceInfo 代码源信息结构体
type SourceInfo struct {
	// 传入部分
	Repository string `json:"repository" validate:"required,url_or_path"` // 必须是URL或文件系统路径
	Branch     string `json:"branch" validate:"omitempty,max=100"`        // 可选，最长100字符
	// CommitId   string `json:"commit_id" validate:"omitempty,len=40"`      // 可选，Git commit ID通常是40位哈希
	Tag string `json:"tag" validate:"omitempty,max=100"` // 可选，（commitid和tag不能同时指定，这里使用tag号）
	// 传出部分，无需验证
	Datetime string `json:"datetime"`
	BuildId  string `json:"build_id"`
	PushId   string `json:"push_id"`
}

func (s SourceInfo) ToString() string {
	str, err := json.Marshal(s)
	if err != nil {
		return fmt.Sprintf("SourceInfo err: %v", err.Error())
	}
	return string(str)
}

func init() {
	// 注册自定义验证器
	if err := check.Validator.RegisterValidation("url_or_path", check.ValidateUrlOrPath); err != nil {
		panic(err)
	}
}
