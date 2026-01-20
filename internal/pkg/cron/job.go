package cron

// Job 是我们自定义的、带有元数据的接口
type Job interface {
	Name() string
	Description() string
	Spec() string
	Run() // 实现 robfig/cron 的 Job 接口
}

// BaseJob 提供基础字段封装
type BaseJob struct {
	JobName string
	JobSpec string
	JobDesc string
}

func (b *BaseJob) Name() string        { return b.JobName }
func (b *BaseJob) Spec() string        { return b.JobSpec }
func (b *BaseJob) Description() string { return b.JobDesc }
