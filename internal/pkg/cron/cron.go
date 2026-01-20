package cron

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
)

// Cron 表达式
var (
	EveryMinuteSpec      = "0 * * * * *" // 每分钟
	EveryFiveMinutesSpec = "0 */5 * * * *"
	DailySpec            = "0 0 0 * * *" // 每天
	DailyAt              = func(hour, minute, second int) string {
		return fmt.Sprintf("%d %d %d * * *", second, minute, hour)
	} // 每天指定时间
)

type Server struct {
	cron *cron.Cron
	log  *log.Helper
}

func NewServer(logger log.Logger) *Server {
	return &Server{
		// WithSeconds 让表达式支持秒级 (可选)
		cron: cron.New(cron.WithSeconds()),
		log:  log.NewHelper(logger),
	}
}

// AddJob 接收实现了 CronJob 接口的对象
func (s *Server) AddJob(job Job) {
	// 包装原始 Job 以支持 Safe 机制和日志
	safeJob := s.makeSafe(job)

	_, err := s.cron.AddJob(job.Spec(), safeJob)
	if err != nil {
		s.log.Fatalf("[Cron] 注册任务 %s 失败: %v", job.Name(), err)
	}
	s.log.Infof("[Cron] 已注册任务: [%s] 频率: [%s] 描述: %s", job.Name(), job.Spec(), job.Description())
}

// makeSafe 核心逻辑：封装 Recovery 和日志记录
func (s *Server) makeSafe(j Job) cron.Job {
	return cron.FuncJob(func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Errorf("[Cron] 任务 %s 发生 Panic: %v\n%s", j.Name(), r, debug.Stack())
			}
		}()

		start := time.Now()
		// 执行原始的 Run 方法
		j.Run()

		s.log.Infof("[Cron] 任务 %s 执行完成, 耗时: %v", j.Name(), time.Since(start))
	})
}

func (s *Server) Start(ctx context.Context) error {
	s.cron.Start()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.cron.Stop()
	return nil
}
