package data

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

type GormLogger struct {
	logger log.Logger // 存储原始 logger 而不是 Helper
}

func NewGormLogger(l log.Logger) glogger.Interface {
	return &GormLogger{
		logger: l,
	}
}

func (l *GormLogger) LogMode(level glogger.LogLevel) glogger.Interface { return l }

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	log.WithContext(ctx, l.logger).Log(log.LevelInfo, "msg", msg, "data", data)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	log.WithContext(ctx, l.logger).Log(log.LevelWarn, "msg", msg, "data", data)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	log.WithContext(ctx, l.logger).Log(log.LevelError, "msg", msg, "data", data)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	// 使用 WithContext 确保 TraceID 能被提取并打印
	logger := log.WithContext(ctx, l.logger)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Log(log.LevelError,
			"kind", "sql",
			"elapsed", float64(elapsed.Nanoseconds())/1e6,
			"rows", rows,
			"sql", sql,
			"err", err,
		)
		return
	}

	logger.Log(log.LevelDebug,
		"kind", "sql",
		"elapsed", float64(elapsed.Nanoseconds())/1e6,
		"rows", rows,
		"sql", sql,
	)
}
