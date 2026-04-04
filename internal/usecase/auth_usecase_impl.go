package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/utils"
)

func (s *authUsecaseImpl) Register(ctx context.Context, usernameStr, password string) (*api_internal.UserDTO, string, error) {
	if err := s.ensureSessionIssuer(); err != nil {
		return nil, "", err
	}

	if len(password) < info.PasswordMinLength {
		return nil, "", ErrPasswordTooShort
	}
	if len(password) > info.PasswordMaxLength {
		return nil, "", ErrPasswordTooLong
	}

	if _, err := s.userRepo.FindByUsername(ctx, s.db, usernameStr); err == nil {
		return nil, "", ErrUsernameTaken
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		slog.Error("failed to find user by username", "username", usernameStr, "error", err)
		return nil, "", err
	}

	hashedPassword, err := utils.HashPasswordWithPepper(password, s.pepper)
	if err != nil {
		return nil, "", err
	}

	un, err := username.NewUserName(usernameStr)
	if err != nil {
		return nil, "", convertUsernameError(err)
	}
	ph, err := passwordhash.NewPasswordHash(hashedPassword)
	if err != nil {
		return nil, "", err
	}

	user := entity.NewUser(un, ph, info.AccountTypePlayer)
	if err := s.userRepo.Save(ctx, s.db, user); err != nil {
		if errors.Is(err, repository.ErrDuplicateUsername) {
			return nil, "", ErrUsernameTaken
		}
		return nil, "", err
	}

	token, err := s.issueSession(ctx, user)
	if err != nil {
		slog.Error("failed to create session after registration", "user_id", user.ID, "error", err)
		return nil, "", err
	}

	accountTypeName := s.masterCache.GetAccountTypeNameByID(user.AccountTypeID)
	return api_internal.ToUserDTO(user, accountTypeName, user.IsPrivate, nil), token, nil
}

func (s *authUsecaseImpl) Login(ctx context.Context, usernameStr, password string) (string, error) {
	if err := s.ensureSessionIssuer(); err != nil {
		return "", err
	}

	user, err := s.userRepo.FindByUsername(ctx, s.db, usernameStr)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
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
	return s.issueSession(ctx, user)
}

func (s *authUsecaseImpl) ensureSessionIssuer() error {
	if s.sessionIssuer == nil {
		return errors.Join(ErrInternalError, errors.New("session issuer is nil"))
	}

	return nil
}

func (s *authUsecaseImpl) issueSession(ctx context.Context, user *entity.User) (string, error) {
	if err := s.ensureSessionIssuer(); err != nil {
		return "", err
	}

	return s.sessionIssuer.IssueSession(ctx, user)
}

func (s *authUsecaseImpl) Logout(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Delete(ctx, s.db, sessionID)
}

func (s *authUsecaseImpl) Authenticate(ctx context.Context, userID int, sessionID string) (*entity.User, error) {
	session, err := s.sessionRepo.FindByID(ctx, s.db, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil, ErrInvalidSession
		}
		slog.Error("failed to find session by id", "session_id", sessionID, "error", err)
		return nil, err
	}
	if session.IsExpired(time.Now()) {
		if err := s.sessionRepo.Delete(ctx, s.db, sessionID); err != nil {
			slog.Error("Failed to delete expired session", "session_id", sessionID, "error", err)
		}
		return nil, ErrInvalidSession
	}
	if session.UserID != userID {
		return nil, ErrUserIDMismatch
	}

	user, err := s.userRepo.FindByID(ctx, s.db, userID)
	if err != nil {
		if !errors.Is(err, repository.ErrUserNotFound) {
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

func convertUsernameError(err error) error {
	if err == nil {
		return nil
	}
	// TODO: internal/domain/vo/username パッケージでエラー変数を公開し、
	// errors.Is() を使った判定に切り替えることを検討してください。
	// 例: case errors.Is(err, username.ErrEmpty):
	switch err.Error() {
	case "username cannot be empty":
		return ErrUsernameEmpty
	case "username must be at least 5 characters":
		return ErrUsernameTooShort
	case "username must be 50 characters or less":
		return ErrUsernameTooLong
	case "username can only contain lowercase letters and numbers":
		return ErrUsernameInvalidChar
	default:
		return err
	}
}
