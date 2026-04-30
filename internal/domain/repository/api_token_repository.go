package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// APITokenRepository はAPIトークンの永続化を扱います。
type APITokenRepository interface {
	// Create はユーザーに紐づくトークンを保存します。
	Create(ctx context.Context, exec Executor, token *entity.APIToken) error
	// FindByUserID はユーザーIDに紐づくトークン一覧を検索します。
	FindByUserID(ctx context.Context, exec Executor, userID int) ([]*entity.APIToken, error)
	// FindByHashedToken はハッシュ化トークンで検索します。対象が存在しない場合は ErrAPITokenNotFound を返します。
	FindByHashedToken(ctx context.Context, exec Executor, hashedToken string) (*entity.APIToken, error)
	// CountByUserID はユーザーIDに紐づくAPIトークン数を返します。
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
	// DeleteByID はユーザーIDとトークンIDに紐づくAPIトークンを削除します。
	DeleteByID(ctx context.Context, exec Executor, userID int, tokenID int64) error
	// DeleteByUserID はユーザーIDに紐づくAPIトークンをすべて削除します。
	DeleteByUserID(ctx context.Context, exec Executor, userID int) error
}
