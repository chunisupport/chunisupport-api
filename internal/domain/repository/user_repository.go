package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// UserRepository はユーザーに関する永続化を扱うリポジトリです。
type UserRepository interface {
	// FindByID はIDでユーザーを検索します。
	FindByID(ctx context.Context, exec Executor, id int) (*entity.User, error)
	// FindByIDForUpdate はIDでユーザーを検索し、更新用に行ロックします。
	FindByIDForUpdate(ctx context.Context, exec Executor, id int) (*entity.User, error)
	// FindByUsername はユーザー名でユーザーを検索します。
	FindByUsername(ctx context.Context, exec Executor, username string) (*entity.User, error)
	// FindAllWithPlayer はユーザー一覧をプレイヤー情報付きで取得します。
	// 通常のユーザー一覧取得用で、プライベート・削除済み・プレイヤー未紐付けアカウントを除外します。
	FindAllWithPlayer(ctx context.Context, exec Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error)
	// FindAllWithPlayerForAdmin はADMIN用にすべてのユーザー一覧をプレイヤー情報付きで取得します。
	// プライベート・削除済み・プレイヤー未紐付けアカウントを含みます。
	FindAllWithPlayerForAdmin(ctx context.Context, exec Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error)
	// FindByFirebaseUID はFirebase UIDでユーザーを検索します。
	FindByFirebaseUID(ctx context.Context, exec Executor, uid string) (*entity.User, error)
	// LinkFirebaseUID は現在値が一致する場合のみ Firebase UID を更新します。
	LinkFirebaseUID(ctx context.Context, exec Executor, userID int, currentUID *string, newUID string, updatedAt time.Time) error
	// CountByAccountType は指定したアカウント種別のユーザー数を取得します。
	CountByAccountType(ctx context.Context, exec Executor, accountTypeID int) (int, error)
	// DeleteByID はユーザーを物理削除します。
	DeleteByID(ctx context.Context, exec Executor, id int) error
	// Save はユーザーを集約単位で保存します。IDが存在する場合は更新、存在しない場合は作成します。
	Save(ctx context.Context, exec Executor, user *entity.User) error
}
