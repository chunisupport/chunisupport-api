package usecase

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
)

// UserUsecase はユーザー関連のユースケースを提供します。
type UserUsecase interface {
	// GetUserProfileWithRecords はユーザー名をキーにプロファイルとレコードを一括取得します。
	// 対象ユーザーが非公開設定の場合は、本人以外は ErrUserPrivate を返します。
	// プレイヤーが紐付いていない場合は ErrPlayerNotLinked を返します。
	GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User) (*api_internal.UserProfileWithRecordsDTO, error)

	// GetAllUsersForAdmin はADMIN用にすべてのユーザー一覧を取得します（ADMIN権限必須）。
	// ページングと検索条件（ユーザー名またはプレイヤー名の前方一致）をサポートします。
	// プライベート・削除済み・プレイヤー未紐付けアカウントも含みます。
	GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]api_internal.AdminUserListResponse, error)

	// DeleteUser はユーザーを論理削除します（ADMIN権限必須）。
	// 既に削除済みのユーザーの場合は ErrUserAlreadyDeleted を返します。
	DeleteUser(ctx context.Context, username string) error

	// RestoreUser はユーザーを復活させます（ADMIN権限必須）。
	// 削除されていないユーザーの場合は ErrUserNotDeleted を返します。
	RestoreUser(ctx context.Context, username string) error
}
