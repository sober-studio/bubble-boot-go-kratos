package snowflake

/*
ID 结构组成（64位）
	1位 符号位：固定为 0，确保 ID 永远为正数。
	41位 时间戳：毫秒级精度，可使用约 69 年。
	10位 工作机器 ID：支持最多 1024 个节点。
	12位 序列号：每毫秒内可生成 4096 个唯一 ID。
*/

import (
	"fmt"
	"sync"
	"time"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/idgen"
)

// 默认值常量
const (
	defaultEpoch        int64 = 1767225600000 // 2026-01-01
	defaultWorkerIDBits uint  = 10            // 1024 个节点
	defaultSequenceBits uint  = 12            // 每毫秒 4096 个 ID
	defaultMaxBackoffMS int64 = 5             // 5ms 回拨容忍
)

// Snowflake 结构体
type Snowflake struct {
	mu    sync.Mutex
	nowTS int64
	seq   int64

	// 配置项
	epoch        int64
	workerIDBits uint
	sequenceBits uint
	maxBackoffMS int64

	// 内部预计算值
	workerID       int64
	maxWorkerID    int64
	maxSequence    int64
	workerIDShift  uint
	timestampShift uint
}

// Option 定义配置函数签名
type Option func(*Snowflake)

// --- 以下是可选的配置函数 ---

func WithEpoch(epoch int64) Option {
	return func(s *Snowflake) {
		s.epoch = epoch
	}
}

func WithWorkerIDBits(bits uint) Option {
	return func(s *Snowflake) {
		s.workerIDBits = bits
	}
}

func WithSequenceBits(bits uint) Option {
	return func(s *Snowflake) {
		s.sequenceBits = bits
	}
}

func WithMaxBackoff(ms int64) Option {
	return func(s *Snowflake) {
		s.maxBackoffMS = ms
	}
}

// --- 初始化函数 ---

// NewSnowflake 第一个参数必填，后续可选
func NewSnowflake(workerID int64, opts ...Option) idgen.IDGenerator {
	// 1. 设置默认值
	s := &Snowflake{
		workerID:     workerID,
		epoch:        defaultEpoch,
		workerIDBits: defaultWorkerIDBits,
		sequenceBits: defaultSequenceBits,
		maxBackoffMS: defaultMaxBackoffMS,
	}

	// 2. 应用传入的可选配置（如果有）
	for _, opt := range opts {
		opt(s)
	}

	// 3. 校验位长度 (总和不建议超过 22 位: 64 - 1符号 - 41时间 = 22)
	if s.workerIDBits+s.sequenceBits > 22 {
		panic("workerIDBits 与 sequenceBits 之和不能超过 22")
	}

	// 4. 预计算掩码和位移
	s.maxWorkerID = -1 ^ (-1 << s.workerIDBits)
	s.maxSequence = -1 ^ (-1 << s.sequenceBits)
	s.workerIDShift = s.sequenceBits
	s.timestampShift = s.sequenceBits + s.workerIDBits

	if workerID < 0 || workerID > s.maxWorkerID {
		panic("workerID 超出范围 (0-1023)")
	}

	return s
}

// NextID 生成 ID 逻辑保持不变
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < s.nowTS {
		offset := s.nowTS - now
		if offset <= s.maxBackoffMS {
			time.Sleep(time.Duration(offset) * time.Millisecond)
			now = time.Now().UnixMilli()
		} else {
			return 0, fmt.Errorf("检测到时钟回拨，拒绝生成 ID")
		}
	}

	if now == s.nowTS {
		s.seq = (s.seq + 1) & s.maxSequence
		if s.seq == 0 {
			for now <= s.nowTS {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.seq = 0
	}

	s.nowTS = now

	id := ((now - s.epoch) << s.timestampShift) |
		(s.workerID << s.workerIDShift) |
		(s.seq)

	return id, nil
}

// --- 使用示例 ---

func main() {
	// 示例 1: 只传 workerID，全部使用默认值
	node1 := NewSnowflake(1)

	// 示例 2: 传 workerID，并修改 Epoch（其他默认）
	node2 := NewSnowflake(2, WithEpoch(1577836800000))

	// 示例 3: 传 workerID，修改机器位长度和回拨容忍度（其他默认）
	node3 := NewSnowflake(3,
		WithWorkerIDBits(8),
		WithMaxBackoff(10),
	)

	id1, _ := node1.NextID()
	id2, _ := node2.NextID()
	id3, _ := node3.NextID()

	fmt.Println("ID1:", id1)
	fmt.Println("ID2:", id2)
	fmt.Println("ID3:", id3)
}
