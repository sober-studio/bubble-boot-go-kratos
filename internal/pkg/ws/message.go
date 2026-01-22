package ws

import "encoding/json"

// Message 统一的消息格式
type Message struct {
	Action string          `json:"action"`        // 业务动作，如 "chat", "ping", "subscribe"
	Data   json.RawMessage `json:"data"`          // 业务数据
	Seq    string          `json:"seq,omitempty"` // 序列号，用于客户端匹配请求(可选)
}

// NewMessage 构造响应消息
func NewMessage(action string, data interface{}) []byte {
	// 简单起见，实际开发建议处理错误
	payload, _ := json.Marshal(data)

	finalMsg, _ := json.Marshal(Message{
		Action: action,
		Data:   payload,
	})
	return finalMsg
}
