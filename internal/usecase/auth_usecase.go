package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/auth"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/username"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
	"github.com/Qman110101/chunisupport-api/internal/info"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/utils"
	"github.com/google/uuid"
)

// AuthUsecase は認証に関するビジネスロジックを扱うユースケースです。
type AuthUsecase interface {
	// Register は新しいユーザーを登録し、自動ログイン後のJWTトークンを返します。
	Register(ctx context.Context, usernameStr, password string) (*api_internal.UserDTO, string, error)
	// Login はユーザー認証を行い、成功時にJWTトークンを返します。
	Login(ctx context.Context, usernameStr, password string) (string, error)
	// Logout は指定されたセッションを無効化します。
	Logout(ctx context.Context, sessionID string) error
	// Authenticate はJWTクレーム情報を検証し、有効なユーザーを返します。
	Authenticate(ctx context.Context, userID int, sessionID string) (*entity.User, error)
	// GetUser はIDでユーザー情報を取得します。
	GetUser(ctx context.Context, id int) (*api_internal.UserDTO, error)
	// UpdatePrivacy はユーザーの非公開設定を更新します。
	UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error
	// ChangePassword はユーザーのパスワードを変更します。
	ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error
	// IssueRecoveryCodes はリカバリーコードを再発行します。
	IssueRecoveryCodes(ctx context.Context, userID int) ([]string, error)
	// RecoverWithRecoveryCode はリカバリーコードでパスワードを再設定します。
	RecoverWithRecoveryCode(ctx context.Context, recoveryCode, newPassword string) error
	// DeleteUser はユーザーを論理削除します。
	DeleteUser(ctx context.Context, userID int) error
}

type authService struct {
	db                    repository.Executor
	tm                    TransactionManager
	userRepo              repository.UserRepository
	sessionRepo           repository.SessionRepository
	recoveryCodeRepo      repository.RecoveryCodeRepository
	playerRecordRepo      repository.PlayerRecordRepository
	jwtSecret             string
	jwtExpirationHour     int
	sessionExpirationHour int
	pepper                string
	masterCache           *masterdata.Cache
}

// NewAuthService は新しいAuthUsecaseを生成します。
func NewAuthService(db repository.Executor, tm TransactionManager, userRepo repository.UserRepository, sessionRepo repository.SessionRepository, recoveryCodeRepo repository.RecoveryCodeRepository, playerRecordRepo repository.PlayerRecordRepository, jwtSecret string, jwtExpirationHour int, sessionExpirationHour int, pepper string, masterCache *masterdata.Cache) AuthUsecase {
	return &authService{
		db:                    db,
		tm:                    tm,
		userRepo:              userRepo,
		sessionRepo:           sessionRepo,
		recoveryCodeRepo:      recoveryCodeRepo,
		playerRecordRepo:      playerRecordRepo,
		jwtSecret:             jwtSecret,
		jwtExpirationHour:     jwtExpirationHour,
		sessionExpirationHour: sessionExpirationHour,
		pepper:                pepper,
		masterCache:           masterCache,
	}
}

// Register は新しいユーザーを登録し、セッションを作成してJWTトークンを返します。
func (s *authService) Register(ctx context.Context, usernameStr, password string) (*api_internal.UserDTO, string, error) {
	// パスワードのバリデーション
	if len(password) < 8 {
		return nil, "", ErrPasswordTooShort
	}
	if len(password) > 128 {
		return nil, "", ErrPasswordTooLong
	}

	if _, err := s.userRepo.FindByUsername(ctx, s.db, usernameStr); err == nil {
		return nil, "", ErrUsernameTaken
	} else if !errors.Is(err, sql.ErrNoRows) {
		slog.Error("failed to find user by username", "username", usernameStr, "error", err)
		return nil, "", err
	}

	hashedPassword, err := utils.HashPasswordWithPepper(password, s.pepper)
	if err != nil {
		return nil, "", err
	}

	un, err := username.NewUserName(usernameStr)
	if err != nil {
		// ユーザー名のバリデーションエラーを適切なエラーに変換
		return nil, "", convertUsernameError(err)
	}
	ph, err := passwordhash.NewPasswordHash(hashedPassword)
	if err != nil {
		return nil, "", err
	}

	user := &entity.User{
		Username:      un,
		PasswordHash:  ph,
		AccountTypeID: 1,
	}

	if err := s.userRepo.Create(ctx, s.db, user); err != nil {
		return nil, "", err
	}

	// 登録後に自動ログイン（セッション作成とJWTトークン生成）
	token, err := s.createSessionAndToken(ctx, user)
	if err != nil {
		slog.Error("failed to create session after registration", "user_id", user.ID, "error", err)
		return nil, "", err
	}

	// 登録直後はレコードが存在しないため、last_score_updateはnil
	return api_internal.ToUserDTO(user, s.masterCache, nil), token, nil
}

// UpdatePrivacy はユーザーの非公開設定を更新します。
func (s *authService) UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error {
	user, err := s.userRepo.FindByID(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	user.ChangePrivacy(isPrivate)
	return s.userRepo.Save(ctx, s.db, user)
}

// ChangePassword はユーザーのパスワードを変更します。
func (s *authService) ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error {
	// パスワードのバリデーション
	if len(newPassword) < info.PasswordMinLength {
		return ErrPasswordTooShort
	}
	if len(newPassword) > info.PasswordMaxLength {
		return ErrPasswordTooLong
	}

	user, err := s.userRepo.FindByID(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	if !utils.CheckPasswordHashWithPepper(currentPassword, s.pepper, user.PasswordHash.String()) {
		return ErrIncorrectPassword
	}

	// セキュリティ: 新旧パスワードが同じ場合も汎用的なエラーを返す
	if utils.CheckPasswordHashWithPepper(newPassword, s.pepper, user.PasswordHash.String()) {
		return ErrInvalidPassword
	}

	hashedNewPassword, err := utils.HashPasswordWithPepper(newPassword, s.pepper)
	if err != nil {
		return err
	}

	newHash, err := passwordhash.NewPasswordHash(hashedNewPassword)
	if err != nil {
		return err
	}

	user.ChangePassword(newHash)
	return s.userRepo.Save(ctx, s.db, user)
}

// IssueRecoveryCodes はユーザーのリカバリーコードを再発行します。
func (s *authService) IssueRecoveryCodes(ctx context.Context, userID int) ([]string, error) {
	if _, err := s.userRepo.FindByID(ctx, s.db, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	displayCodes := make([]string, 0, info.RecoveryCodeCount)
	recoveryCodes := make([]*entity.RecoveryCode, 0, info.RecoveryCodeCount)
	seen := make(map[string]struct{}, info.RecoveryCodeCount)

	const generationAttemptMultiplier = 10
	maxGenerationAttempts := info.RecoveryCodeCount * generationAttemptMultiplier
	for attempts := 0; len(recoveryCodes) < info.RecoveryCodeCount; attempts++ {
		// 衝突頻発時の無限ループを防ぐため、試行回数に上限を設ける
		if attempts >= maxGenerationAttempts {
			return nil, fmt.Errorf("failed to generate unique recovery codes after %d attempts", maxGenerationAttempts)
		}

		display, normalized, hashBytes, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}

		recoveryCodes = append(recoveryCodes, &entity.RecoveryCode{
			UserID:   userID,
			CodeHash: hashBytes,
		})
		displayCodes = append(displayCodes, display)
	}

	if err := s.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := s.recoveryCodeRepo.DeleteByUserID(ctx, tx, userID); err != nil {
			return err
		}
		return s.recoveryCodeRepo.CreateBatch(ctx, tx, recoveryCodes)
	}); err != nil {
		return nil, err
	}

	return displayCodes, nil
}

// RecoverWithRecoveryCode はリカバリーコードでパスワードを再設定します。
func (s *authService) RecoverWithRecoveryCode(ctx context.Context, recoveryCode, newPassword string) error {
	if len(newPassword) < info.PasswordMinLength {
		return ErrPasswordTooShort
	}
	if len(newPassword) > info.PasswordMaxLength {
		return ErrPasswordTooLong
	}

	normalized := normalizeRecoveryCode(recoveryCode)
	hashBytes := hashRecoveryCode(normalized)

	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		code, err := s.recoveryCodeRepo.FindByHashForUpdate(ctx, tx, hashBytes)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidRecoveryCredentials
			}
			return err
		}
		user, err := s.userRepo.FindByID(ctx, tx, code.UserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidRecoveryCredentials
			}
			return err
		}
		if !user.IsActive() {
			return ErrInvalidRecoveryCredentials
		}
		if utils.CheckPasswordHashWithPepper(newPassword, s.pepper, user.PasswordHash.String()) {
			return ErrInvalidPassword
		}

		hashedNewPassword, err := utils.HashPasswordWithPepper(newPassword, s.pepper)
		if err != nil {
			return err
		}
		newHash, err := passwordhash.NewPasswordHash(hashedNewPassword)
		if err != nil {
			return err
		}

		user.ChangePassword(newHash)
		if err := s.userRepo.Save(ctx, tx, user); err != nil {
			return err
		}

		return s.recoveryCodeRepo.DeleteByID(ctx, tx, code.ID)
	})
}

// DeleteUser はユーザーの論理削除フラグを立てます。
func (s *authService) DeleteUser(ctx context.Context, userID int) error {
	user, err := s.userRepo.FindByID(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	user.Delete()
	return s.userRepo.Save(ctx, s.db, user)
}

func generateRecoveryCode() (string, string, []byte, error) {
	if info.RecoveryCodeSegmentCount <= 0 || info.RecoveryCodeSegmentLength <= 0 {
		return "", "", nil, fmt.Errorf("invalid recovery code format configuration")
	}

	totalLength := info.RecoveryCodeSegmentCount * info.RecoveryCodeSegmentLength
	displayBuilder := strings.Builder{}
	displayBuilder.Grow(totalLength + info.RecoveryCodeSegmentCount - 1)
	normalizedBuilder := strings.Builder{}
	normalizedBuilder.Grow(totalLength)

	charsetMax := big.NewInt(int64(len(info.RecoveryCodeCharset)))
	for i := 0; i < totalLength; i++ {
		index, err := rand.Int(rand.Reader, charsetMax)
		if err != nil {
			return "", "", nil, err
		}
		ch := info.RecoveryCodeCharset[int(index.Int64())]
		if i > 0 && i%info.RecoveryCodeSegmentLength == 0 {
			displayBuilder.WriteByte('-')
		}
		displayBuilder.WriteByte(ch)
		normalizedBuilder.WriteByte(ch)
	}

	normalized := normalizedBuilder.String()
	sum := sha256.Sum256([]byte(normalized))
	return displayBuilder.String(), normalized, sum[:], nil
}

func normalizeRecoveryCode(raw string) string {
	normalized := strings.ReplaceAll(raw, "-", "")
	return strings.ToUpper(normalized)
}

func hashRecoveryCode(normalized string) []byte {
	sum := sha256.Sum256([]byte(normalized))
	return sum[:]
}

// Login はユーザーをログインさせ、成功した場合にJWTを返します。
func (s *authService) Login(ctx context.Context, usernameStr, password string) (string, error) {
	user, err := s.userRepo.FindByUsername(ctx, s.db, usernameStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidCredentials
		}
		slog.Error("failed to find user by username", "username", usernameStr, "error", err)
		return "", err
	}

	if !user.IsActive() {
		return "", ErrInvalidCredentials
	}

	if !utils.CheckPasswordHashWithPepper(password, s.pepper, user.PasswordHash.String()) {
		return "", ErrInvalidCredentials
	}

	return s.createSessionAndToken(ctx, user)
}

// createSessionAndToken はセッションを作成しJWTトークンを生成して返します。
// セッション数が上限（info.MaxSessionsPerUser）を超える場合、最も古いセッションから削除します。
func (s *authService) createSessionAndToken(ctx context.Context, user *entity.User) (string, error) {
	sessionID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	session := &entity.Session{
		ID:        sessionID.String(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Duration(s.sessionExpirationHour) * time.Hour),
	}
	if err := s.sessionRepo.Create(ctx, s.db, session); err != nil {
		return "", err
	}

	// セッション数制限: 上限を超えたら古いセッションを削除
	if err := s.sessionRepo.DeleteOldestSessionsOverLimit(ctx, s.db, user.ID, info.MaxSessionsPerUser); err != nil {
		slog.Error("Failed to delete oldest sessions", "user_id", user.ID, "error", err)
		// セッション削除の失敗は致命的ではないため、処理を続行
	}

	// SessionIDを含むJWTを生成
	token, err := auth.GenerateToken(user, session.ID, s.jwtSecret, s.jwtExpirationHour)
	if err != nil {
		// トークン生成に失敗したら、作成したセッションを削除しておく
		slog.Error("Failed to generate token, cleaning up session", "session_id", session.ID, "error", err)
		if delErr := s.sessionRepo.Delete(ctx, s.db, session.ID); delErr != nil {
			slog.Error("Failed to delete session during cleanup", "session_id", session.ID, "error", delErr)
		}
		return "", err
	}

	return token, nil
}

// Logout はセッションを無効化します。
func (s *authService) Logout(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Delete(ctx, s.db, sessionID)
}

// Authenticate はJWTのクレーム内の情報（UserID, SessionID）が有効か検証します。
func (s *authService) Authenticate(ctx context.Context, userID int, sessionID string) (*entity.User, error) {
	// 1. セッションの存在確認
	session, err := s.sessionRepo.FindByID(ctx, s.db, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// セキュリティ: セッションが見つからない場合も統合エラー
			return nil, ErrInvalidSession
		}
		slog.Error("failed to find session by id", "session_id", sessionID, "error", err)
		return nil, err
	}

	// 2. セッションの有効期限チェック
	if session.ExpiresAt.Before(time.Now()) {
		// 期限切れのセッションは削除しておく
		if err := s.sessionRepo.Delete(ctx, s.db, sessionID); err != nil {
			slog.Error("Failed to delete expired session", "session_id", sessionID, "error", err)
		}
		// セキュリティ: 期限切れも統合エラー
		return nil, ErrInvalidSession
	}

	// 3. セッションのUserIDとJWTのUserIDが一致するか確認
	if session.UserID != userID {
		return nil, ErrUserIDMismatch
	}

	// 4. ユーザーの存在確認
	user, err := s.userRepo.FindByID(ctx, s.db, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Error("failed to find user by id", "user_id", userID, "error", err)
			return nil, err
		}
		return nil, ErrUserNotFound
	}

	if !user.IsActive() {
		if err := s.sessionRepo.Delete(ctx, s.db, sessionID); err != nil {
			slog.Error("Failed to delete session for deleted user", "session_id", sessionID, "error", err)
		}
		return nil, ErrUserDeleted
	}

	return user, nil
}

// GetUser はIDでユーザー情報を取得しDTOに変換して返します。
func (s *authService) GetUser(ctx context.Context, id int) (*api_internal.UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}

	// プレイヤーが紐付いている場合のみ、スコア最終更新日を取得
	var lastScoreUpdate *time.Time
	if user.PlayerID != nil {
		lastScoreUpdate, err = s.playerRecordRepo.GetLastScoreUpdate(ctx, s.db, *user.PlayerID)
		if err != nil {
			slog.Error("failed to get last score update", "player_id", *user.PlayerID, "error", err)
			// エラーがあっても処理は継続（last_score_updateがnilになるだけ）
		}
	}

	return api_internal.ToUserDTO(user, s.masterCache, lastScoreUpdate), nil
}

// convertUsernameError はユーザー名のバリデーションエラーを適切なエラーに変換します。
func convertUsernameError(err error) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	switch {
	case errMsg == "username cannot be empty":
		return ErrUsernameEmpty
	case errMsg == "username must be at least 5 characters":
		return ErrUsernameTooShort
	case errMsg == "username must be 50 characters or less":
		return ErrUsernameTooLong
	case errMsg == "username can only contain lowercase letters and numbers":
		return ErrUsernameInvalidChar
	default:
		return err
	}
}
