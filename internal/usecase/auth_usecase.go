package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

// AuthUsecase は認証の最小責務を扱うユースケースです。
type AuthUsecase interface {
	// Register は新しいユーザーを登録し、自動ログイン後のJWTトークンを返します。
	Register(ctx context.Context, usernameStr, password string) (*api_internal.UserDTO, string, error)
	// Login はユーザー認証を行い、成功時にJWTトークンを返します。
	Login(ctx context.Context, usernameStr, password string) (string, error)
	// Logout は指定されたセッションを無効化します。
	Logout(ctx context.Context, sessionID string) error
	// Authenticate はJWTクレーム情報を検証し、有効なユーザーを返します。
	Authenticate(ctx context.Context, userID int, sessionID string) (*entity.User, error)
}

type authUsecaseImpl struct {
	db            repository.Executor
	userRepo      repository.UserRepository
	sessionRepo   repository.SessionRepository
	sessionIssuer SessionIssuer
	pepper        string
	masterCache   AccountTypeProvider
}

// NewAuthUsecase は新しいAuthUsecaseを生成します。
func NewAuthUsecase(db repository.Executor, userRepo repository.UserRepository, sessionRepo repository.SessionRepository, jwtSecret string, jwtExpirationHour int, sessionExpirationHour int, pepper string, masterCache AccountTypeProvider) AuthUsecase {
	return &authUsecaseImpl{
		db:            db,
		userRepo:      userRepo,
		sessionRepo:   sessionRepo,
		sessionIssuer: NewSessionIssuer(db, sessionRepo, jwtSecret, jwtExpirationHour, sessionExpirationHour),
		pepper:        pepper,
		masterCache:   masterCache,
	}
}
