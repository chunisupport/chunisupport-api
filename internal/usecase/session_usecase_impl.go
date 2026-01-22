package usecase

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
)

type sessionUsecaseImpl struct {
	sessionRepo repository.SessionRepository
	db          repository.Executor
}

// NewSessionUsecase は新しいSessionUsecaseを生成します。
func NewSessionUsecase(
	sessionRepo repository.SessionRepository,
	db repository.Executor,
) SessionUsecase {
	return &sessionUsecaseImpl{
		sessionRepo: sessionRepo,
		db:          db,
	}
}

// GetSessionCount は指定されたユーザーの有効なセッション数を取得します。
func (u *sessionUsecaseImpl) GetSessionCount(ctx context.Context, userID int) (int, error) {
	count, err := u.sessionRepo.CountByUserID(ctx, u.db, userID)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// LogoutOtherSessions は現在のセッション以外をすべてログアウトします。
func (u *sessionUsecaseImpl) LogoutOtherSessions(ctx context.Context, userID int, currentSessionID string) error {
	return u.sessionRepo.DeleteByUserIDExcept(ctx, u.db, userID, currentSessionID)
}
