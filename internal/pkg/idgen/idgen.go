package idgen

// IDGenerator 定义了通用的 ID 生成接口
type IDGenerator interface {
	NextID() (int64, error)
}
