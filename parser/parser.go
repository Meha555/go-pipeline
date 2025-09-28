package parser

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type PipelineConf struct {
	Name    string `yaml:"name" validate:"required"`
	Version string `yaml:"version" validate:"required"`
	Shell   string `yaml:"shell" validate:"required"`
	// NOTE 使用指针，这样可以判断是否存在该字段
	Notifiers *notifiersConf `yaml:"notifiers,omitempty"`
	Envs      []string       `yaml:"envs,omitempty"`
	Workdir   string         `yaml:"workdir,omitempty"`
	Stages    []string       `yaml:"stages" validate:"required"`
	Skips     []string       `yaml:"skips,omitempty"`
	// NOTE gopkg.in/yaml.v3 库中，结构体字段的声明顺序会影响解析优先级。如果 inline 字段（Jobs）在结构体中声明的位置早于其他关键字段（如 Stages/Skips），可能导致部分嵌套字段被意外忽略。
	Jobs map[string]jobConf `yaml:",inline" validate:"dive"`
}

// notifiersConf 通知器配置
type notifiersConf struct {
	Email *emailNotifierConf `yaml:"email,omitempty"`
	Bot   *botNotifierConf   `yaml:"bot,omitempty"`
	SMS   *smsNotifierConf   `yaml:"sms,omitempty"`
}

type jobConf struct {
	Stage        string    `yaml:"stage" validate:"required"`
	Actions      []string  `yaml:"actions" validate:"required"`
	Timeout      string    `yaml:"timeout,omitempty"`
	AllowFailure bool      `yaml:"allow_failure,omitempty"`
	Hooks        hooksConf `yaml:"hooks,omitempty"`
}

type hooksConf struct {
	Before []string `yaml:"before,omitempty"`
	After  []string `yaml:"after,omitempty"`
}

// emailNotifierConf 邮件通知器配置
type emailNotifierConf struct {
	Server string      `yaml:"server" validate:"required,hostname|ip"`
	Port   int         `yaml:"port" validate:"required,min=1,max=65535"`
	From   emailPoster `yaml:"from" validate:"required"`
	To     []string    `yaml:"to" validate:"required,min=1,dive,email"`
	Cc     []string    `yaml:"cc,omitempty" validate:"dive,email"`
}

type emailPoster struct {
	Address  string `yaml:"address" validate:"required,email"`
	Password string `yaml:"password" validate:"required"`
}

// botNotifierConf 机器人通知器配置
type botNotifierConf struct {
	Server string `yaml:"server" validate:"required,url"`
}

// smsNotifierConf 短信通知器配置
type smsNotifierConf struct {
	Server string `yaml:"server" validate:"required,url"`
	AppID  string `yaml:"appid" validate:"required"`
	AppKey string `yaml:"appkey" validate:"required"`
}

// ParseConfigFile 解析 YAML 配置文件
func ParseConfigFile(configPath string) (*PipelineConf, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); err != nil {
		return nil, err
	}

	// 读取配置文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	// 解析配置文件
	config := &PipelineConf{}
	if err := yaml.Unmarshal(content, config); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	// 校验配置信息
	if err := validate.Struct(config); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			for _, e := range validationErrors {
				// 这里只返回第一个校验错误
				return nil, fmt.Errorf("validate config failed: %w, (field: %s, tag: %s)", e, e.Field(), e.Tag())
			}
		}
	}

	return config, nil
}

var (
	ErrTimeoutIsEmpty = errors.New("timeout is empty")
	ErrTimeoutNoUnit  = errors.New("timeout has no unit")
	ErrTimeoutValue   = errors.New("invalid timeout value")
	ErrTimeoutUnit    = errors.New("invalid timeout unit, supported units are ms, s, m, h, d")
)

// ParseDuration 解析带单位的时间字符串为time.Duration
func ParseDuration(duration string) (time.Duration, error) {
	if duration == "" {
		return time.Duration(math.MaxInt64), ErrTimeoutIsEmpty
	}

	unitIndex := -1
	for i := range duration {
		if duration[i] < '0' || duration[i] > '9' {
			unitIndex = i
			break
		}
	}
	if unitIndex == -1 {
		return time.Duration(math.MaxInt64), ErrTimeoutNoUnit
	}

	// 分割数值和单位
	valueStr := duration[:unitIndex]
	unit := strings.ToLower(duration[unitIndex:])

	// 解析数值
	value, err := strconv.Atoi(valueStr)
	if err != nil || value < 0 {
		return time.Duration(math.MaxInt64), fmt.Errorf("%w: %v", ErrTimeoutValue, err)
	}

	// 根据单位转换为对应的Duration
	switch unit {
	case "ms":
		return time.Duration(value) * time.Millisecond, nil
	case "s":
		return time.Duration(value) * time.Second, nil
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrTimeoutUnit, unit)
	}
}

func ParseArgs(args []string, ctx context.Context) {
	if len(args) == 0 {
		return
	}
	// 这些额外参数会被解析为KV对或FLAG，存储到环境变量中，方便在命令中取用
	for _, arg := range args {
		kv := strings.SplitN(arg, "=", 2)
		if len(kv) == 2 { // KEY=VALUE
			// ctx = context.WithValue(ctx, internal.ContextKey(kv[0]), kv[1])
			os.Setenv(kv[0], kv[1])
		} else { // FLAG
			// ctx = context.WithValue(ctx, internal.ContextKey(arg), "true")
			os.Setenv(arg, "true")
		}
	}
}

var validate = validator.New()
