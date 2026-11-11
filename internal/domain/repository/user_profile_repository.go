package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// UserProfileRepository はUser集約と独立したプロフィールの永続化を扱います。
type UserProfileRepository interface {
	// FindByUserID は未設定の場合、エラーではなくnilを返します。
	FindByUserID(ctx context.Context, exec Executor, userID int) (*entity.UserProfile, error)
	// Save はユーザーごとに1件のプロフィールを保存します。
	Save(ctx context.Context, exec Executor, profile *entity.UserProfile) error
	// DeleteByUserID は未設定に戻します。行がなくても成功します。
	DeleteByUserID(ctx context.Context, exec Executor, userID int) error
}
