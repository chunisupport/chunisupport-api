package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// APITokenRepository はAPIトークン集約の永続化を扱います。
type APITokenRepository interface {
	// Save はAPIトークンを新規保存または更新します。
	Save(ctx context.Context, exec Executor, token *entity.APIToken) error
	// ListByUserID はユーザーが所有するトークンを新しい順で返します。
	ListByUserID(ctx context.Context, exec Executor, userID int) ([]*entity.APIToken, error)
	// FindByIDAndUserID は所有者を限定してAPIトークンを検索します。
	FindByIDAndUserID(ctx context.Context, exec Executor, id uint64, userID int) (*entity.APIToken, error)
	// FindByIDAndUserIDForUpdate は所有者を限定してAPIトークンを検索し、更新用に行ロックします。
	FindByIDAndUserIDForUpdate(ctx context.Context, exec Executor, id uint64, userID int) (*entity.APIToken, error)
	// FindByHashedToken はハッシュ化トークンで検索します。
	FindByHashedToken(ctx context.Context, exec Executor, hashedToken string) (*entity.APIToken, error)
	// FindByHashedTokenForUpdate はハッシュ化トークンで検索し、更新用に行ロックします。
	FindByHashedTokenForUpdate(ctx context.Context, exec Executor, hashedToken string) (*entity.APIToken, error)
	// CountByUserID はユーザーが所有するトークン数を返します。
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
	// DeleteByIDAndUserID は所有者を限定してAPIトークンを削除します。
	DeleteByIDAndUserID(ctx context.Context, exec Executor, id uint64, userID int) error
}
