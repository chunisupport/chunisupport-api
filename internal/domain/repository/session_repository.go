package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// SessionRepository はセッションデータへのアクセスを抽象化するインターフェースです。
type SessionRepository interface {
	// Create は新しいセッションを保存します。
	Create(ctx context.Context, exec Executor, session *entity.Session) error
	// FindByID はセッションIDでセッションを検索します。対象が存在しない場合は ErrSessionNotFound を返します。
	FindByID(ctx context.Context, exec Executor, id string) (*entity.Session, error)
	// Delete はセッションIDでセッションを削除します。
	Delete(ctx context.Context, exec Executor, id string) error
	// CountByUserID は指定されたユーザーのセッション数を取得します。
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
	// DeleteByUserID は指定されたユーザーのセッションを全て削除します。
	DeleteByUserID(ctx context.Context, exec Executor, userID int) error
	// DeleteByUserIDExcept は指定されたセッションID以外のユーザーのセッションを全て削除します。
	DeleteByUserIDExcept(ctx context.Context, exec Executor, userID int, excludeSessionID string) error
	// DeleteOldestSessionsOverLimit は指定されたユーザーのセッション数が上限を超えている場合、古い順に削除します。
	DeleteOldestSessionsOverLimit(ctx context.Context, exec Executor, userID int, maxCount int) error
}
