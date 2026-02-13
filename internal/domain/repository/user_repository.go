package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
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
}
