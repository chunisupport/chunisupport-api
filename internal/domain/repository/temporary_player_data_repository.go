package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// TemporaryPlayerDataRepository は未ログイン一時データの保存・参照・削除を扱います。
type TemporaryPlayerDataRepository interface {
	Create(ctx context.Context, exec Executor, data *entity.TemporaryPlayerData) error
	FindByToken(ctx context.Context, exec Executor, token string) (*entity.TemporaryPlayerData, error)
	ConsumeByToken(ctx context.Context, exec Executor, token string) (*entity.TemporaryPlayerData, error)
	Delete(ctx context.Context, exec Executor, token string) error
}
