package oss

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
)

var ProviderSet = wire.NewSet()

func NewOSS(c *conf.Data, logger log.Logger) Storage {
	// 1. 如果是开发环境，默认返回 Local Storage，除非配置指定了其他
	// 这里可以根据实际需求调整策略，例如完全依赖配置文件
	if c.Oss == nil {
		// 如果没有配置 OSS，返回默认 Local Storage 或报错
		// 这里为了稳健性，构造一个默认配置
		c.Oss = &conf.Data_Oss{
			Bucket: "uploads",
		}
		return NewLocalStorage(c.Oss, logger)
	}

	// 2. 根据配置文件决定使用哪个供应商
	switch c.Oss.Provider {
	case "aliyun":
		return NewAliyunStorage(c.Oss, logger)
	case "qiniu":
		return NewQiniuStorage(c.Oss, logger)
	case "minio":
		return NewMinioStorage(c.Oss, logger)
	case "local":
		return NewLocalStorage(c.Oss, logger)
	}

	// 默认 fallback
	return NewLocalStorage(c.Oss, logger)
}
