package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// APITokenRepository はAPIトークンの永続化を扱います。
type APITokenRepository interface {
	// CreateOrReplace はユーザーに紐づくトークンを保存し、既存のトークンがあれば置き換えます。
	CreateOrReplace(ctx context.Context, exec Executor, token *entity.APIToken) error
	// FindByHashedToken はハッシュ化トークンで検索します。対象が存在しない場合は ErrAPITokenNotFound を返します。
	FindByHashedToken(ctx context.Context, exec Executor, hashedToken string) (*entity.APIToken, error)
	// DeleteByUserID はユーザーIDに紐づくAPIトークンを削除します。
	DeleteByUserID(ctx context.Context, exec Executor, userID int) error
}
