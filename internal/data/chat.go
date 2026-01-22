package data

import (
	"context"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// chatRepo 实现了 biz.ChatRepo 接口
type chatRepo struct {
	data *Data
	log  *log.Helper
}

// NewChatRepo 构造函数，由 Wire 注入
func NewChatRepo(data *Data, logger log.Logger) biz.ChatRepo {
	return &chatRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/chat")),
	}
}

// SaveMessage 模拟保存消息到数据库
func (r *chatRepo) SaveMessage(ctx context.Context, msg *biz.Message) error {
	// 模拟数据库写入延迟
	r.log.WithContext(ctx).Infof("存储消息到数据库: [ID:%s] From:%s To:%s Content:%s",
		msg.ID, msg.FromUID, msg.ToUID, msg.Content)

	// 在真实的生产环境中，这里会使用 gorm 或 ent：
	// return r.data.db.WithContext(ctx).Create(msg).Error

	return nil
}

// UpdateUserStatus 模拟更新用户在线状态
func (r *chatRepo) UpdateUserStatus(ctx context.Context, uid string, online bool) error {
	status := "离线"
	if online {
		status = "上线"
	}

	r.log.WithContext(ctx).Infof("更新用户状态: 用户ID:%s 状态:%s", uid, status)

	// 在真实的生产环境中，这里通常操作 Redis：
	// return r.data.redis.Set(ctx, "user_online:"+uid, online, 0).Err()

	return nil
}
