package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

// APITokenRepository はAPIトークン集約の永続化を扱います。
type APITokenRepository interface {
	// Save はAPIトークンを新規保存または更新します。
	Save(ctx context.Context, exec Executor, token *entity.APIToken) error
	// ListByUserID はユーザーが所有するトークンを新しい順で返します。
	ListByUserID(ctx context.Context, exec Executor, userID int) ([]*entity.APIToken, error)
	// FindByDisplayIDAndUserID は所有者を限定してAPIトークンを表示用IDで検索します。
	FindByDisplayIDAndUserID(ctx context.Context, exec Executor, displayID displayid.DisplayID, userID int) (*entity.APIToken, error)
	// FindByDisplayIDAndUserIDForUpdate は所有者を限定してAPIトークンを表示用IDで検索し、更新用に行ロックします。
	FindByDisplayIDAndUserIDForUpdate(ctx context.Context, exec Executor, displayID displayid.DisplayID, userID int) (*entity.APIToken, error)
	// FindByHashedToken はハッシュ化トークンで検索します。
	FindByHashedToken(ctx context.Context, exec Executor, hashedToken string) (*entity.APIToken, error)
	// FindByHashedTokenForUpdate はハッシュ化トークンで検索し、更新用に行ロックします。
	FindByHashedTokenForUpdate(ctx context.Context, exec Executor, hashedToken string) (*entity.APIToken, error)
	// CountByUserID はユーザーが所有するトークン数を返します。
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
	// DeleteByDisplayIDAndUserID は所有者を限定してAPIトークンを表示用IDで削除します。
	DeleteByDisplayIDAndUserID(ctx context.Context, exec Executor, displayID displayid.DisplayID, userID int) error
}
