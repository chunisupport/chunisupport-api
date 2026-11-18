package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// APITokenOutput は管理APIへ返すAPIトークン情報です。
type APITokenOutput struct {
	ID          string
	Name        string
	Permission  string
	PublicOnly  bool
	TokenPrefix *string
	LastUsedAt  *time.Time
	CreatedAt   time.Time
}

// GeneratedAPITokenOutput は発行時に一度だけ返す平文トークンと管理情報です。
type GeneratedAPITokenOutput struct {
	Token    string
	Metadata *APITokenOutput
}

// APITokenUsecase はAPIトークンに関するユースケースを提供します。
type APITokenUsecase interface {
	// Generate は権限付きの名前付きAPIトークンを追加発行します。read_writeはEDITOR以上に限ります。
	// publicOnly を true にすると、本人やフレンドであっても非公開ユーザーを参照できないトークンになります。
	Generate(ctx context.Context, userID int, name string, permission string, publicOnly bool) (*GeneratedAPITokenOutput, error)
	// List はユーザーが所有するAPIトークンを返します。
	List(ctx context.Context, userID int) ([]*APITokenOutput, error)
	// Rename は所有するAPIトークンの名前を変更します。
	Rename(ctx context.Context, userID int, id string, name string) (*APITokenOutput, error)
	// Validate はプレーントークンを検証し、紐づくユーザーとトークン情報を返します。
	Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error)
	// Delete は所有するAPIトークンをID指定で削除します。
	Delete(ctx context.Context, userID int, id string) error
}
