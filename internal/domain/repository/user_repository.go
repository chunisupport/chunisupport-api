package repository

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// UserRepository はユーザーに関する永続化を扱うリポジトリです。
type UserRepository interface {
	// FindByID はIDでユーザーを検索します。
	FindByID(ctx context.Context, exec Executor, id int) (*entity.User, error)
	// FindByUsername はユーザー名でユーザーを検索します。
	FindByUsername(ctx context.Context, exec Executor, username string) (*entity.User, error)
	// FindAllWithPlayer はユーザー一覧をプレイヤー情報付きで取得します。
	// 通常のユーザー一覧取得用で、プライベート・削除済み・プレイヤー未紐付けアカウントを除外します。
	FindAllWithPlayer(ctx context.Context, exec Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error)
	// FindAllWithPlayerForAdmin はADMIN用にすべてのユーザー一覧をプレイヤー情報付きで取得します。
	// プライベート・削除済み・プレイヤー未紐付けアカウントを含みます。
	FindAllWithPlayerForAdmin(ctx context.Context, exec Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error)
	// Create は新しいユーザーを作成します。
	Create(ctx context.Context, exec Executor, user *entity.User) error
	// Save はユーザーを集約単位で保存します。IDが存在する場合は更新、存在しない場合は作成します。
	Save(ctx context.Context, exec Executor, user *entity.User) error
	// UpdatePrivacy はユーザーの非公開設定を更新します。
	UpdatePrivacy(ctx context.Context, exec Executor, userID int, isPrivate bool) error
	// UpdatePassword はユーザーのパスワードハッシュを更新します。
	UpdatePassword(ctx context.Context, exec Executor, userID int, passwordHash string) error
	// SoftDelete はユーザーの論理削除フラグを立てます。
	SoftDelete(ctx context.Context, exec Executor, userID int) error
	// Restore はユーザーの論理削除フラグを解除します。
	Restore(ctx context.Context, exec Executor, userID int) error
	// LinkPlayer はユーザーにプレイヤーIDを紐付けます。
	LinkPlayer(ctx context.Context, exec Executor, userID int, playerID int) error
}
