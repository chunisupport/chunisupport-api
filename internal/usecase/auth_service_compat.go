package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

// legacyAuthService は既存コード向けの互換ラッパーです。
type legacyAuthService struct {
	auth            AuthUsecase
	userCredential  UserCredentialUsecase
	recoveryUsecase RecoveryUsecase
}

// NewAuthService は後方互換のためのコンストラクタです。新規実装では NewAuthUsecase を使用してください。
func NewAuthService(db repository.Executor, tm TransactionManager, userRepo repository.UserRepository, sessionRepo repository.SessionRepository, recoveryCodeRepo repository.RecoveryCodeRepository, playerRecordRepo repository.PlayerRecordRepository, jwtSecret string, jwtExpirationHour int, sessionExpirationHour int, pepper string, masterCache AccountTypeProvider) *legacyAuthService {
	return &legacyAuthService{
		auth:            NewAuthUsecase(db, userRepo, sessionRepo, jwtSecret, jwtExpirationHour, sessionExpirationHour, pepper, masterCache),
		userCredential:  NewUserCredentialUsecase(db, userRepo, playerRecordRepo, pepper, masterCache),
		recoveryUsecase: NewRecoveryUsecase(db, tm, userRepo, recoveryCodeRepo, pepper),
	}
}

func (s *legacyAuthService) Register(ctx context.Context, usernameStr, password string) (*api_internal.UserDTO, string, error) {
	return s.auth.Register(ctx, usernameStr, password)
}
func (s *legacyAuthService) Login(ctx context.Context, usernameStr, password string) (string, error) {
	return s.auth.Login(ctx, usernameStr, password)
}
func (s *legacyAuthService) Logout(ctx context.Context, sessionID string) error {
	return s.auth.Logout(ctx, sessionID)
}
func (s *legacyAuthService) Authenticate(ctx context.Context, userID int, sessionID string) (*entity.User, error) {
	return s.auth.Authenticate(ctx, userID, sessionID)
}
func (s *legacyAuthService) GetUser(ctx context.Context, id int) (*api_internal.UserDTO, error) {
	return s.userCredential.GetUser(ctx, id)
}
func (s *legacyAuthService) UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error {
	return s.userCredential.UpdatePrivacy(ctx, userID, isPrivate)
}
func (s *legacyAuthService) ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error {
	return s.userCredential.ChangePassword(ctx, userID, currentPassword, newPassword)
}
func (s *legacyAuthService) IssueRecoveryCodes(ctx context.Context, userID int) ([]string, error) {
	return s.recoveryUsecase.IssueRecoveryCodes(ctx, userID)
}
func (s *legacyAuthService) RecoverWithRecoveryCode(ctx context.Context, recoveryCode, newPassword string) error {
	return s.recoveryUsecase.RecoverWithRecoveryCode(ctx, recoveryCode, newPassword)
}
func (s *legacyAuthService) DeleteUser(ctx context.Context, userID int) error {
	return s.userCredential.DeleteUser(ctx, userID)
}
