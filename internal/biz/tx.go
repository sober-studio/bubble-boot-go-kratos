package biz

import "context"

// Transaction 事务接口
type Transaction interface {
	InTx(context.Context, func(ctx context.Context) error) error
}
