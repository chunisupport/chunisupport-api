package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

// UserUsecase はユーザー関連のユースケースを定義します。
type UserUsecase interface {
	// GetUserProfile はユーザー名をキーにプロファイル（username + player）のみを軽量に取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人でなければ ErrUserPrivate を返します。
	// プレイヤーが紐づいていない場合は ErrPlayerNotLinked を返します。
	GetUserProfile(ctx context.Context, username string, requester *entity.User) (*api_internal.UserProfileDTO, error)

	// GetUserProfileWithRecords はユーザー名をキーにプロファイルとレコードを一括取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人でなければ ErrUserPrivate を返します。
	// プレイヤーが紐づいていない場合は ErrPlayerNotLinked を返します。
	GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*api_internal.UserProfileWithRecordsDTO, error)

	// GetUserProfileRatingView はユーザー名をキーにレーティング表示向けのプロファイルとレコードを取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人でなければ ErrUserPrivate を返します。
	// プレイヤーが紐づいていない場合は ErrPlayerNotLinked を返します。
	GetUserProfileRatingView(ctx context.Context, username string, requester *entity.User) (*api_internal.UserProfileRatingViewDTO, error)

	// GetUserProfileRecordView はユーザー名をキーにレコード表示向けのプロファイルとレコードを取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人でなければ ErrUserPrivate を返します。
	// プレイヤーが紐づいていない場合は ErrPlayerNotLinked を返します。
	GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*api_internal.UserProfileRecordViewDTO, error)

	// GetUserUpdatedAt はユーザー名をキーにプレイヤーデータの updated_at のみを取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人でなければ ErrUserPrivate を返します。
	// プレイヤーが紐づいていない場合は ErrPlayerNotLinked を返します。
	GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*api_internal.UserUpdatedAtDTO, error)

	// GetAllUsersForAdmin はADMIN用にすべてのユーザー一覧を取得します。
	GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]api_internal.AdminUserListResponse, error)

	// DeleteUser はユーザーを物理削除します。
	DeleteUser(ctx context.Context, requester *entity.User, username string) error
}
