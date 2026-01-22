package usecase

import (
	"context"
)

// SessionUsecase はセッション管理のビジネスロジックを提供します。
type SessionUsecase interface {
	// GetSessionCount は指定されたユーザーの有効なセッション数を取得します。
	GetSessionCount(ctx context.Context, userID int) (int, error)
	// LogoutOtherSessions は現在のセッション以外をすべてログアウトします。
	LogoutOtherSessions(ctx context.Context, userID int, currentSessionID string) error
}
