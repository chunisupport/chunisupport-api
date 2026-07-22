package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/apitokenname"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

var (
	ErrInvalidAPIToken       = errors.New("invalid API token")
	ErrInvalidAPITokenName   = errors.New("invalid API token name")
	ErrInvalidAPITokenID     = errors.New("invalid API token id")
	ErrAPITokenNotFound      = errors.New("API token not found")
	ErrAPITokenLimitExceeded = errors.New("API token limit exceeded")
	ErrAPITokenNameConflict  = errors.New("API token name conflict")
)

// apiTokenUsecase は APITokenUsecase の実装です。
type apiTokenUsecase struct {
	db        repository.Executor
	tm        TransactionManager
	tokenRepo repository.APITokenRepository
	userRepo  repository.UserRepository
	now       func() time.Time
}

// NewAPITokenUsecase はAPITokenUsecaseを生成します。
func NewAPITokenUsecase(db repository.Executor, tm TransactionManager, tokenRepo repository.APITokenRepository, userRepo repository.UserRepository) APITokenUsecase {
	return newAPITokenUsecaseWithClock(db, tm, tokenRepo, userRepo, time.Now)
}

func newAPITokenUsecaseWithClock(db repository.Executor, tm TransactionManager, tokenRepo repository.APITokenRepository, userRepo repository.UserRepository, now func() time.Time) APITokenUsecase {
	return &apiTokenUsecase{
		db:        db,
		tm:        tm,
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
		now:       now,
	}
}

// Generate は既存トークンを維持したまま、名前付きAPIトークンを追加発行します。
func (u *apiTokenUsecase) Generate(ctx context.Context, userID int, name string) (*GeneratedAPITokenOutput, error) {
	validatedName, err := apitokenname.NewAPITokenName(name)
	if err != nil {
		return nil, ErrInvalidAPITokenName
	}

	plain, err := generateAPIToken()
	if err != nil {
		return nil, err
	}
	token, err := entity.NewAPIToken(userID, validatedName.String(), hashToken(plain), plain[:info.APITokenPrefixLength])
	if err != nil {
		return nil, err
	}

	err = u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if _, err := u.userRepo.FindByIDForUpdate(ctx, tx, userID); err != nil {
			return err
		}
		count, err := u.tokenRepo.CountByUserID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if count >= info.APITokenMaxPerUser {
			return ErrAPITokenLimitExceeded
		}
		if err := u.tokenRepo.Save(ctx, tx, token); err != nil {
			if errors.Is(err, repository.ErrAPITokenConflict) {
				return ErrAPITokenNameConflict
			}
			return err
		}
		created, err := u.tokenRepo.FindByIDAndUserID(ctx, tx, token.ID, userID)
		if err != nil {
			return err
		}
		token = created
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &GeneratedAPITokenOutput{Token: plain, Metadata: toAPITokenOutput(token)}, nil
}

// List はユーザーが所有するAPIトークンを返します。
func (u *apiTokenUsecase) List(ctx context.Context, userID int) ([]*APITokenOutput, error) {
	tokens, err := u.tokenRepo.ListByUserID(ctx, u.db, userID)
	if err != nil {
		return nil, err
	}
	outputs := make([]*APITokenOutput, 0, len(tokens))
	for _, token := range tokens {
		outputs = append(outputs, toAPITokenOutput(token))
	}
	return outputs, nil
}

// Rename は所有するAPIトークンの名前を変更します。
func (u *apiTokenUsecase) Rename(ctx context.Context, userID int, id string, name string) (*APITokenOutput, error) {
	tokenID, err := parseAPITokenID(id)
	if err != nil {
		return nil, err
	}
	var token *entity.APIToken
	err = u.tm.Transactional(ctx, func(tx repository.Executor) error {
		found, err := u.tokenRepo.FindByIDAndUserIDForUpdate(ctx, tx, tokenID, userID)
		if errors.Is(err, repository.ErrAPITokenNotFound) {
			return ErrAPITokenNotFound
		}
		if err != nil {
			return err
		}
		if err := found.Rename(name); err != nil {
			return ErrInvalidAPITokenName
		}
		if err := u.tokenRepo.Save(ctx, tx, found); err != nil {
			if errors.Is(err, repository.ErrAPITokenConflict) {
				return ErrAPITokenNameConflict
			}
			if errors.Is(err, repository.ErrAPITokenNotFound) {
				return ErrAPITokenNotFound
			}
			return err
		}
		token = found
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toAPITokenOutput(token), nil
}

// Validate はAPIトークンを検証し、有効な場合はユーザーとトークン情報を返します。
func (u *apiTokenUsecase) Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error) {
	if rawToken == "" {
		return nil, nil, ErrInvalidAPIToken
	}

	hashedToken := hashToken(rawToken)
	token, err := u.tokenRepo.FindByHashedToken(ctx, u.db, hashedToken)
	if err != nil {
		if errors.Is(err, repository.ErrAPITokenNotFound) {
			return nil, nil, ErrInvalidAPIToken
		}
		return nil, nil, err
	}

	user, err := u.userRepo.FindByID(ctx, u.db, token.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrInvalidAPIToken
		}
		return nil, nil, err
	}

	now := u.now().UTC()
	if token.ShouldRecordUsage(now, info.APITokenLastUsedUpdateInterval) {
		err := u.tm.Transactional(ctx, func(tx repository.Executor) error {
			lockedToken, err := u.tokenRepo.FindByHashedTokenForUpdate(ctx, tx, hashedToken)
			if errors.Is(err, repository.ErrAPITokenNotFound) {
				return ErrInvalidAPIToken
			}
			if err != nil {
				return err
			}
			if lockedToken.ShouldRecordUsage(now, info.APITokenLastUsedUpdateInterval) {
				lockedToken.RecordUsage(now)
				if err := u.tokenRepo.Save(ctx, tx, lockedToken); err != nil {
					return err
				}
			}
			token = lockedToken
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	return user, token, nil
}

// Delete は所有するAPIトークンをID指定で削除します。
func (u *apiTokenUsecase) Delete(ctx context.Context, userID int, id string) error {
	tokenID, err := parseAPITokenID(id)
	if err != nil {
		return err
	}
	err = u.tokenRepo.DeleteByIDAndUserID(ctx, u.db, tokenID, userID)
	if errors.Is(err, repository.ErrAPITokenNotFound) {
		return ErrAPITokenNotFound
	}
	return err
}

func generateAPIToken() (string, error) {
	buf := make([]byte, info.APITokenRandomByteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func parseAPITokenID(id string) (uint64, error) {
	parsed, err := strconv.ParseUint(id, 10, 64)
	if err != nil || parsed == 0 {
		return 0, ErrInvalidAPITokenID
	}
	return parsed, nil
}

func toAPITokenOutput(token *entity.APIToken) *APITokenOutput {
	return &APITokenOutput{
		ID:          token.ID,
		Name:        token.Name.String(),
		TokenPrefix: token.TokenPrefix,
		LastUsedAt:  token.LastUsedAt,
		CreatedAt:   token.CreatedAt,
	}
}

// hashToken は生のトークン文字列をSHA-256でハッシュ化します。
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
