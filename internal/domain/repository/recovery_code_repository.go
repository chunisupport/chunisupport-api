package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// RecoveryCodeRepository はリカバリーコードの永続化を扱います。
type RecoveryCodeRepository interface {
	// CreateBatch はリカバリーコードをまとめて保存します。
	CreateBatch(ctx context.Context, exec Executor, codes []*entity.RecoveryCode) error
	// DeleteByUserID は指定ユーザーのリカバリーコードを削除します。
	DeleteByUserID(ctx context.Context, exec Executor, userID int) error
	// DeleteByID はリカバリーコードを削除します。
	DeleteByID(ctx context.Context, exec Executor, id uint32) error
	// FindByHash はハッシュでリカバリーコードを検索します。対象が存在しない場合は ErrRecoveryCodeNotFound を返します。
	FindByHash(ctx context.Context, exec Executor, codeHash []byte) (*entity.RecoveryCode, error)
	// FindByHashForUpdate はハッシュでリカバリーコードを検索し、ロックします。対象が存在しない場合は ErrRecoveryCodeNotFound を返します。
	FindByHashForUpdate(ctx context.Context, exec Executor, codeHash []byte) (*entity.RecoveryCode, error)
}
