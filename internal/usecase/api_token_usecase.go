package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// APITokenUsecase はAPIトークンに関するユースケースを提供します。
type APITokenUsecase interface {
	// Generate はユーザーに紐づくAPIトークンを新しく発行し、プレーントークンを返します。
	Generate(ctx context.Context, userID int) (string, error)
	// Validate はプレーントークンを検証し、紐づくユーザーとトークン情報を返します。
	Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error)
	// Delete はユーザーに紐づくAPIトークンを削除します。
	Delete(ctx context.Context, userID int) error
}
