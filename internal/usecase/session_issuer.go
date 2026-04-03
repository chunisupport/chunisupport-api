package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/auth"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/google/uuid"
)

// SessionIssuer は認証済みユーザーに対してセッションと JWT を発行します。
type SessionIssuer interface {
	IssueSession(ctx context.Context, user *entity.User) (string, error)
}

type sessionIssuer struct {
	db                    repository.Executor
	sessionRepo           repository.SessionRepository
	jwtSecret             string
	jwtExpirationHour     int
	sessionExpirationHour int
}

// NewSessionIssuer はセッション発行器を生成します。
func NewSessionIssuer(db repository.Executor, sessionRepo repository.SessionRepository, jwtSecret string, jwtExpirationHour int, sessionExpirationHour int) SessionIssuer {
	return &sessionIssuer{
		db:                    db,
		sessionRepo:           sessionRepo,
		jwtSecret:             jwtSecret,
		jwtExpirationHour:     jwtExpirationHour,
		sessionExpirationHour: sessionExpirationHour,
	}
}

func (s *sessionIssuer) IssueSession(ctx context.Context, user *entity.User) (string, error) {
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

	if err := s.sessionRepo.DeleteOldestSessionsOverLimit(ctx, s.db, user.ID, info.MaxSessionsPerUser); err != nil {
		slog.Error("Failed to delete oldest sessions", "user_id", user.ID, "error", err)
	}

	token, err := auth.GenerateToken(user, session.ID, s.jwtSecret, s.jwtExpirationHour)
	if err != nil {
		slog.Error("Failed to generate token, cleaning up session", "session_id", session.ID, "error", err)
		if delErr := s.sessionRepo.Delete(ctx, s.db, session.ID); delErr != nil {
			slog.Error("Failed to delete session during cleanup", "session_id", session.ID, "error", delErr)
		}
		return "", err
	}

	return token, nil
}

var _ SessionIssuer = (*sessionIssuer)(nil)
