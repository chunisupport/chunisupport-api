package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	domainservice "github.com/chunisupport/chunisupport-api/internal/domain/service"
)

const tokenByteLength = 32
const apiTokenMaxNameLength = 15

// APITokenMaxCountPerUser は1ユーザーが保持できるAPIトークン数の上限です。
const APITokenMaxCountPerUser = 10

var ErrInvalidAPIToken = errors.New("invalid API token")
var ErrInvalidAPITokenName = errors.New("invalid API token name")
var ErrAPITokenLimitExceeded = errors.New("api token limit exceeded")

// apiTokenUsecase は APITokenUsecase の実装です。
type apiTokenUsecase struct {
	db        repository.Executor
	tm        TransactionManager
	tokenRepo repository.APITokenRepository
	userRepo  repository.UserRepository
}

// NewAPITokenService はAPITokenUsecaseを生成します。
func NewAPITokenService(db repository.Executor, tm TransactionManager, tokenRepo repository.APITokenRepository, userRepo repository.UserRepository) APITokenUsecase {
	return &apiTokenUsecase{
		db:        db,
		tm:        tm,
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
	}
}

// Generate はユーザーに紐づくAPIトークンを新しく発行します。
func (us *apiTokenUsecase) Generate(ctx context.Context, userID int, name string) (string, *entity.APIToken, error) {
	name, err := normalizeAPITokenName(name)
	if err != nil {
		return "", nil, err
	}

	buf := make([]byte, tokenByteLength)
	if _, err = rand.Read(buf); err != nil {
		return "", nil, err
	}

	plain := base64.RawURLEncoding.EncodeToString(buf)
	hashed := hashToken(plain)

	token := &entity.APIToken{
		UserID:      userID,
		Name:        name,
		HashedToken: hashed,
		CreatedAt:   time.Now().UTC(),
	}

	err = us.tm.Transactional(ctx, func(tx repository.Executor) error {
		if _, err := us.userRepo.FindByIDForUpdate(ctx, tx, userID); err != nil {
			return err
		}
		count, err := us.tokenRepo.CountByUserID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if count >= APITokenMaxCountPerUser {
			return ErrAPITokenLimitExceeded
		}
		return us.tokenRepo.Create(ctx, tx, token)
	})
	if err != nil {
		return "", nil, err
	}

	return plain, token, nil
}

// List はユーザーに紐づくAPIトークン一覧を返します。
func (us *apiTokenUsecase) List(ctx context.Context, userID int) ([]*entity.APIToken, error) {
	return us.tokenRepo.FindByUserID(ctx, us.db, userID)
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

// Delete はユーザーに紐づく指定APIトークンを削除します。
func (us *apiTokenUsecase) Delete(ctx context.Context, userID int, tokenID int64) error {
	return us.tokenRepo.DeleteByID(ctx, us.db, userID, tokenID)
}

// DeleteAll はユーザーに紐づくAPIトークンをすべて削除します。
func (us *apiTokenUsecase) DeleteAll(ctx context.Context, userID int) error {
	return us.tokenRepo.DeleteByUserID(ctx, us.db, userID)
}

// hashToken は生のトークン文字列をSHA-256でハッシュ化します。
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func normalizeAPITokenName(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return domainservice.DefaultAPITokenName, nil
	}
	if utf8.RuneCountInString(normalized) > apiTokenMaxNameLength {
		return "", ErrInvalidAPITokenName
	}
	return normalized, nil
}
