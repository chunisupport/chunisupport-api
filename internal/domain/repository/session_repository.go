package repository

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// SessionRepository はセッションデータへのアクセスを抽象化するインターフェースです。
type SessionRepository interface {
	// Create は新しいセッションを保存します。
	Create(ctx context.Context, exec Executor, session *entity.Session) error
	// FindByID はセッションIDでセッションを検索します。
	FindByID(ctx context.Context, exec Executor, id string) (*entity.Session, error)
	// Delete はセッションIDでセッションを削除します。
	Delete(ctx context.Context, exec Executor, id string) error
	// CountByUserID は指定されたユーザーのセッション数を取得します。
	CountByUserID(ctx context.Context, exec Executor, userID int) (int, error)
	// DeleteByUserIDExcept は指定されたセッションID以外のユーザーのセッションを全て削除します。
	DeleteByUserIDExcept(ctx context.Context, exec Executor, userID int, excludeSessionID string) error
}
