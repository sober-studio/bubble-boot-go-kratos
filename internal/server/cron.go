package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/job"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/cron"
)

func NewCronServer(
	c *conf.Server,
	logger log.Logger,
	hello *job.HelloJob,
) *cron.Server {
	srv := cron.NewServer(logger)

	srv.AddJob(hello)

	return srv
}
