package debug

import (
	"context"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/env"
)

type debugKey struct{}

// Info 存储调试数据的 map
type Info map[string]interface{}

// FromContext 从 Context 中获取调试信息
func FromContext(ctx context.Context) (Info, bool) {
	info, ok := ctx.Value(debugKey{}).(Info)
	return info, ok
}

// NewContext 向 Context 中注入调试信息
func NewContext(ctx context.Context, key string, value interface{}) context.Context {
	info, ok := FromContext(ctx)
	if !ok {
		info = make(Info)
	}
	info[key] = value
	return context.WithValue(ctx, debugKey{}, info)
}

func IsDebug() bool { return !env.IsProd() }
