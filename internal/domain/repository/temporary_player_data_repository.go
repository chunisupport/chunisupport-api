package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// TemporaryPlayerDataRepository は未ログイン一時データの保存・参照・削除を扱います。
type TemporaryPlayerDataRepository interface {
	Create(ctx context.Context, exec Executor, data *entity.TemporaryPlayerData) error
	// Claim は確定処理中として予約し、同一トークンの並行確定を防ぎます。
	// 処理中のトークンには ErrTemporaryPlayerDataInUse を返します。
	Claim(ctx context.Context, exec Executor, token string) (*entity.TemporaryPlayerData, error)
	// Release は確定処理に失敗したトークンの予約を解除し、同じトークンで再試行できるようにします。
	Release(ctx context.Context, exec Executor, token string) error
	Delete(ctx context.Context, exec Executor, token string) error
}
