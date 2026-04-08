package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// TemporaryPlayerDataRepository は未ログイン一時データの保存・参照・削除を扱います。
type TemporaryPlayerDataRepository interface {
	Create(ctx context.Context, data *entity.TemporaryPlayerData) error
	FindByToken(ctx context.Context, token string) (*entity.TemporaryPlayerData, error)
	Delete(ctx context.Context, token string) error
}
