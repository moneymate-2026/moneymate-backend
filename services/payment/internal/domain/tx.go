package domain

import "context"

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
