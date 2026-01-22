package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/ws"
)

type WebsocketService struct {
	hub          *ws.Hub
	chatService  *ChatService
	tokenService auth.TokenService
	upgrader     websocket.Upgrader
	log          *log.Helper
}

func NewWebsocketService(hub *ws.Hub, chatService *ChatService, tokenService auth.TokenService, logger log.Logger) *WebsocketService {
	return &WebsocketService{
		hub:          hub,
		chatService:  chatService,
		tokenService: tokenService,
		log:          log.NewHelper(logger),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// WSHandler 处理 HTTP 升级请求
func (s *WebsocketService) WSHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 生产级：身份验证 (JWT)
	// 因为浏览器 WebSocket API 不支持自定义 Header，通常从 Query 拿 token
	token := r.URL.Query().Get("token")
	uid := s.verifyToken(token) // 你的认证逻辑
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 2. 协议升级
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Errorf("upgrade failed: %v", err)
		return
	}

	// 3. 注册到管理中心
	// 我们传入一个处理函数给 Client，当 Client 收到消息时回调
	s.hub.Register(uid, conn, s.dispatch)
}

// dispatch 分发中心：根据 Action 调用不同的业务逻辑
func (s *WebsocketService) dispatch(uid string, payload []byte) {
	var msg ws.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		s.log.Errorf("unmarshal error: %v", err)
		return
	}

	ctx := context.Background()

	switch msg.Action {
	case "chat":
		s.chatService.HandleChat(ctx, uid, msg.Data)
	case "ping":
		// 直接响应一个 pong
		s.hub.SendToUser(uid, ws.NewMessage("pong", map[string]string{"reply": "alive"}))
	default:
		s.log.Warnf("unknown action: %s", msg.Action)
	}
}

func (s *WebsocketService) verifyToken(token string) string {
	if token == "" {
		return ""
	}
	// 这里调用你的 JWT 解析逻辑
	userID, err := s.tokenService.ParseTokenFromTokenString(context.Background(), token)
	if err != nil {
		return ""
	}
	return userID
}
