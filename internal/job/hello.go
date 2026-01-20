package job

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/cron"
)

var _ cron.Job = (*HelloJob)(nil)

type HelloJob struct {
	cron.BaseJob
	log *log.Helper
}

func NewHelloJob(logger log.Logger) *HelloJob {
	return &HelloJob{
		BaseJob: cron.BaseJob{
			JobName: "HelloJob",
			JobSpec: cron.EveryMinuteSpec,
			JobDesc: "Hello Job",
		},
		log: log.NewHelper(logger),
	}
}

func (h HelloJob) Run() {
	h.log.Infof("Hello Job Run")
}
