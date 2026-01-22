package service

import (
	"context"
	"encoding/json"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/biz"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/ws"
)

type ChatService struct {
	hub *ws.Hub
	uc  *biz.ChatUseCase
}

func NewChatService(hub *ws.Hub, uc *biz.ChatUseCase) *ChatService {
	return &ChatService{hub: hub, uc: uc}
}

// HandleChat 处理客户端发来的聊天消息
func (s *ChatService) HandleChat(ctx context.Context, uid string, data []byte) {
	// 1. 解析业务数据
	var req struct {
		ToUID   string `json:"to_uid"`
		Content string `json:"content"`
	}
	json.Unmarshal(data, &req)

	// 2. 调用业务逻辑层（存入数据库、敏感词过滤等）
	msg, err := s.uc.ProcessMessage(ctx, uid, req.ToUID, req.Content)
	if err != nil {
		// 向发送者响应失败
		s.hub.SendToUser(uid, ws.NewMessage("error", map[string]string{"msg": "发送失败"}))
		return
	}

	// 3. 响应发送者 (确认发送成功)
	s.hub.SendToUser(uid, ws.NewMessage("chat_ack", map[string]string{"msg_id": msg.ID}))

	// 4. 推送给接收者 (如果接收者在线)
	s.hub.SendToUser(req.ToUID, ws.NewMessage("new_chat", map[string]string{
		"from_uid": uid,
		"content":  req.Content,
	}))
}
