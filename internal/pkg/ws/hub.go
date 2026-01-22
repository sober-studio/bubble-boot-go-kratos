package ws

import (
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second    // 写入超时
	pongWait       = 60 * time.Second    // 等待 pong 超时
	pingPeriod     = (pongWait * 9) / 10 // 发送 ping 周期
	maxMessageSize = 512                 // 最大消息大小
)

type HandlerFunc func(uid string, payload []byte)

// Client 封装单个连接
type Client struct {
	Hub     *Hub
	Conn    *websocket.Conn
	UID     string
	Send    chan []byte // 缓冲发送通道，防止并发写 panic
	handler HandlerFunc // 处理器回调
}

// Hub 维护所有活跃连接
type Hub struct {
	clients sync.Map // key: UID, value: *Client
	log     *log.Helper
}

func NewHub(logger log.Logger) *Hub {
	return &Hub{
		log: log.NewHelper(log.With(logger, "module", "ws/hub")),
	}
}

// Register 注册并启动客户端监听
func (h *Hub) Register(uid string, conn *websocket.Conn, handler HandlerFunc) {
	client := &Client{
		Hub:     h,
		Conn:    conn,
		UID:     uid,
		Send:    make(chan []byte, 256),
		handler: handler, // 注入处理器
	}
	h.clients.Store(uid, client)

	// 启动读写协程
	go client.writePump()
	go client.readPump()
}

func (h *Hub) Unregister(uid string) {
	if client, ok := h.clients.LoadAndDelete(uid); ok {
		close(client.(*Client).Send)
		client.(*Client).Conn.Close()
	}
}

func (h *Hub) SendToUser(uid string, msg []byte) {
	if client, ok := h.clients.Load(uid); ok {
		client.(*Client).Send <- msg
	}
}

// readPump 从连接读取消息并处理（心跳处理核心）
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister(c.UID)
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, payload, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		// 调用分发器
		if c.handler != nil {
			c.handler(c.UID, payload)
		}
	}
}

// writePump 将消息从通道写回连接（解决并发写问题）
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
