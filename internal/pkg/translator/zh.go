package translator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/debug"
)

// 1. 定义一个通用的校验错误接口
// 只要是 PGV 生成的错误结构体（不论是哪个版本），通常都实现了这些方法
type validationError interface {
	Field() string
	Reason() string
	ErrorName() string
}

// 2. 字段映射表（可选）
// 将 Proto 中的英文名映射为中文名。如果字段太多，不写这个表则直接显示原文字段名
var fieldMap = map[string]string{
	"Username": "用户名",
	"Password": "密码",
	"Email":    "邮箱",
	"Age":      "年龄",
	"Mobile":   "手机号",
	"IdCard":   "身份证号",
}

// Translate 翻译核心函数
func Translate(err error) string {
	if err == nil {
		return "success"
	}

	// 3. 使用接口断言。只要 err 实现了 validationError 接口的方法，就能匹配成功
	// 这样就避免了去引用 github.com/envoyproxy/... 或 github.com/bufbuild/...
	var vErr validationError
	if errors.As(err, &vErr) {
		field := vErr.Field()
		// 查表翻译字段名
		if zh, ok := fieldMap[field]; ok {
			field = zh
		}

		reason := vErr.Reason()

		// 4. 翻译常见的 PGV 错误原因
		// 旧版 PGV 的 Reason 字符串通常是固定的
		switch {
		case strings.Contains(reason, "value length must be at least"):
			return fmt.Sprintf("%s长度不够", field)
		case strings.Contains(reason, "value length must be between"):
			return fmt.Sprintf("%s长度不符合要求", field)
		case strings.Contains(reason, "is required"):
			return fmt.Sprintf("%s不能为空", field)
		case strings.Contains(reason, "must be a valid email"):
			return fmt.Sprintf("%s格式不正确", field)
		case strings.Contains(reason, "value must be greater than"):
			return fmt.Sprintf("%s太小了", field)
		case strings.Contains(reason, "value must be less than"):
			return fmt.Sprintf("%s太大了", field)
		case strings.Contains(reason, "value must be inside range"):
			return fmt.Sprintf("%s不在合法范围内", field)
		case strings.Contains(reason, "value does not match regex pattern"):
			return fmt.Sprintf("%s格式错误", field)
		default:
			// 如果没有匹配到预设的规则，返回原字段名 + 英文原由
			if debug.IsDebug() {
				return fmt.Sprintf("%s校验失败: %s", field, reason)
			}
			return fmt.Sprintf("%s校验失败: %s", field)
		}
	}

	// 如果不是校验错误，返回原始错误信息
	return err.Error()
}
