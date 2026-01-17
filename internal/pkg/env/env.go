package env

import "sync"

type Type string

const (
	Dev  Type = "dev"
	Test Type = "test"
	Prod Type = "prod"
)

var (
	current = Dev // 默认设为开发环境
	once    sync.Once
)

// Init 在 main.go 中被调用一次
func Init(e string) {
	once.Do(func() {
		current = Type(e)
	})
}

func IsDev() bool  { return current == Dev }
func IsTest() bool { return current == Test }
func IsProd() bool { return current == Prod }
func Get() Type    { return current }
