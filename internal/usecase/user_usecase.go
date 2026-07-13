package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// UserUsecase はユーザー関連のユースケースを定義します。
type UserUsecase interface {
	// GetUserProfile はユーザー名をキーにプロファイル（username + player）のみを軽量に取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人または承認済みフレンドでなければ ErrUserPrivate を返します。
	GetUserProfile(ctx context.Context, username string, requester *entity.User) (*UserProfileOutput, error)

	// GetUserUpdatedAt はユーザーのプロフィールとレコードの updated_at のうち新しい方を取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人または承認済みフレンドでなければ ErrUserPrivate を返します。
	GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*UserUpdatedAtOutput, error)

	// GetUserProfileWithRecords はユーザー名をキーにプロファイルとレコードを一括取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人または承認済みフレンドでなければ ErrUserPrivate を返します。
	GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*UserProfileWithRecordsOutput, error)

	// GetUserProfileRatingView はユーザー名をキーにレーティング表示向けのプロファイルとレコードを取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人または承認済みフレンドでなければ ErrUserPrivate を返します。
	GetUserProfileRatingView(ctx context.Context, username string, requester *entity.User) (*UserProfileRatingViewOutput, error)

	// GetUserProfileRecordView はユーザー名をキーにレコード表示向けのプロファイルとレコードを取得します。
	// 対象ユーザーが非公開設定の場合、閲覧者が本人または承認済みフレンドでなければ ErrUserPrivate を返します。
	GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*UserProfileRecordViewOutput, error)

	// GetUserSongRecord は通常楽曲1曲分のユーザーレコードを取得します。
	GetUserSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, includeNoPlay bool, difficulty string) (*UserSongRecordOutput, error)

	// GetUserWorldsendSongRecord は WORLD'S END 楽曲1曲分のユーザーレコードを取得します。
	GetUserWorldsendSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, includeNoPlay bool) (*UserWorldsendSongRecordOutput, error)

	// GetAllUsersForAdmin はADMIN用にすべてのユーザー一覧を取得します。
	GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]AdminUserOutput, error)

	// DeleteUser はユーザーを物理削除します。
	DeleteUser(ctx context.Context, requester *entity.User, username string) error
}
