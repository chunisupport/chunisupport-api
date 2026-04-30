package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// APITokenUsecase はAPIトークンに関するユースケースを提供します。
type APITokenUsecase interface {
	// Generate はユーザーに紐づくAPIトークンを新しく発行し、プレーントークンと保存情報を返します。
	Generate(ctx context.Context, userID int, name string) (string, *entity.APIToken, error)
	// List はユーザーに紐づくAPIトークン一覧を返します。
	List(ctx context.Context, userID int) ([]*entity.APIToken, error)
	// Validate はプレーントークンを検証し、紐づくユーザーとトークン情報を返します。
	Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error)
	// Delete はユーザーに紐づく指定APIトークンを削除します。
	Delete(ctx context.Context, userID int, tokenID int64) error
	// DeleteAll はユーザーに紐づくAPIトークンをすべて削除します。
	DeleteAll(ctx context.Context, userID int) error
}
