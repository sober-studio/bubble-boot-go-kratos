package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// Message 聊天消息领域模型
type Message struct {
	ID        string
	FromUID   string
	ToUID     string
	Content   string
	Status    int // 0: 未读, 1: 已读
	CreatedAt time.Time
}

// ChatRepo 数据库操作接口（由 data 层实现）
type ChatRepo interface {
	SaveMessage(ctx context.Context, msg *Message) error
	UpdateUserStatus(ctx context.Context, uid string, online bool) error
}

// ChatUseCase 业务逻辑实现
type ChatUseCase struct {
	repo ChatRepo
	log  *log.Helper
}

func NewChatUseCase(repo ChatRepo, logger log.Logger) *ChatUseCase {
	return &ChatUseCase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "usecase/chat")),
	}
}

// ProcessMessage 处理并存储消息
func (uc *ChatUseCase) ProcessMessage(ctx context.Context, from, to, content string) (*Message, error) {
	uc.log.Infof("Processing message from %s to %s", from, to)

	// 1. 业务校验：不能为空
	if content == "" {
		return nil, fmt.Errorf("content is empty")
	}

	// 2. 业务校验：禁止给自己发消息
	if from == to {
		return nil, fmt.Errorf("cannot send message to yourself")
	}

	// 3. 敏感词过滤 (示例逻辑)
	filteredContent := uc.filterSensitiveWords(content)

	// 4. 构造消息实体
	msg := &Message{
		ID:        generateMsgID(), // 自行实现唯一 ID 生成
		FromUID:   from,
		ToUID:     to,
		Content:   filteredContent,
		Status:    0,
		CreatedAt: time.Now(),
	}

	// 5. 持久化到数据库 (调用 data 层)
	err := uc.repo.SaveMessage(ctx, msg)
	if err != nil {
		uc.log.Errorf("failed to save message: %v", err)
		return nil, err
	}

	return msg, nil
}

// 模拟敏感词过滤
func (uc *ChatUseCase) filterSensitiveWords(content string) string {
	// 实际开发中可以接入之前提到的 DFA 算法过滤工具
	return content
}

func generateMsgID() string {
	// 实际应使用 Snowflake 或 UUID
	return time.Now().Format("20060102150405")
}
