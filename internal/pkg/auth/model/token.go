package model

import "time"

// UserToken 用于持久化
type UserToken struct {
	JTI       string    // JWT ID
	UserID    string    // 用户 ID
	IssuedAt  time.Time // 签发时间
	ExpiresAt time.Time // 过期时间
	TokenStr  string    // JWT 原文
}
