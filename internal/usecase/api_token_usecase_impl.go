package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

const tokenByteLength = 32

var ErrInvalidAPIToken = errors.New("invalid API token")

// apiTokenUsecase は APITokenUsecase の実装です。
type apiTokenUsecase struct {
	db        repository.Executor
	tokenRepo repository.APITokenRepository
	userRepo  repository.UserRepository
}

// NewAPITokenUsecase はAPITokenUsecaseを生成します。
func NewAPITokenUsecase(db repository.Executor, tokenRepo repository.APITokenRepository, userRepo repository.UserRepository) APITokenUsecase {
	return &apiTokenUsecase{
		db:        db,
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
	}
}

// Generate はユーザーに紐づくAPIトークンを新しく発行します。既存のトークンは置き換えられます。
func (us *apiTokenUsecase) Generate(ctx context.Context, userID int) (string, error) {
	buf := make([]byte, tokenByteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	plain := base64.RawURLEncoding.EncodeToString(buf)
	hashed := hashToken(plain)

	token := &entity.APIToken{
		UserID:      userID,
		HashedToken: hashed,
	}

	if err := us.tokenRepo.CreateOrReplace(ctx, us.db, token); err != nil {
		return "", err
	}

	return plain, nil
}

// GetStatus はユーザーに紐づくAPIトークンの状態を返します。未発行の場合は nil を返します。
func (us *apiTokenUsecase) GetStatus(ctx context.Context, userID int) (*entity.APIToken, error) {
	token, err := us.tokenRepo.FindByUserID(ctx, us.db, userID)
	if err != nil {
		if errors.Is(err, repository.ErrAPITokenNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return token, nil
}

// Validate はAPIトークンを検証し、有効な場合はユーザーとトークン情報を返します。
func (us *apiTokenUsecase) Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error) {
	if rawToken == "" {
		return nil, nil, ErrInvalidAPIToken
	}

	hashed := hashToken(rawToken)
	token, err := us.tokenRepo.FindByHashedToken(ctx, us.db, hashed)
	if err != nil {
		if errors.Is(err, repository.ErrAPITokenNotFound) {
			return nil, nil, ErrInvalidAPIToken
		}
		return nil, nil, err
	}

	user, err := us.userRepo.FindByID(ctx, us.db, token.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrInvalidAPIToken
		}
		return nil, nil, err
	}

	return user, token, nil
}

// Delete はユーザーに紐づくAPIトークンを削除します。
func (us *apiTokenUsecase) Delete(ctx context.Context, userID int) error {
	return us.tokenRepo.DeleteByUserID(ctx, us.db, userID)
}

// hashToken は生のトークン文字列をSHA-256でハッシュ化します。
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
