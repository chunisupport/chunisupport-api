package repository

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// RecoveryCodeRepository はリカバリーコードの永続化を扱います。
type RecoveryCodeRepository interface {
	// CreateBatch はリカバリーコードをまとめて保存します。
	CreateBatch(ctx context.Context, exec Executor, codes []*entity.RecoveryCode) error
	// DeleteByUserID は指定ユーザーのリカバリーコードを削除します。
	DeleteByUserID(ctx context.Context, exec Executor, userID int) error
	// DeleteByID はリカバリーコードを削除します。
	DeleteByID(ctx context.Context, exec Executor, id uint32) error
	// FindByHash はハッシュでリカバリーコードを検索します。
	FindByHash(ctx context.Context, exec Executor, codeHash []byte) (*entity.RecoveryCode, error)
	// FindByHashForUpdate はハッシュでリカバリーコードを検索し、ロックします。
	FindByHashForUpdate(ctx context.Context, exec Executor, codeHash []byte) (*entity.RecoveryCode, error)
}
