package repository

import "errors"

// リポジトリ層の共通エラー定義
// infraからusecaseへの依存を避けるため、ドメイン関連エラーはここで定義します。
var (
	// ErrUserNotFound はユーザーが見つからなかった場合に返されるエラーです。
	ErrUserNotFound = errors.New("user not found")

	// ErrSongNotFound は楽曲が見つからなかった場合に返されるエラーです。
	ErrSongNotFound = errors.New("song not found")

	// ErrDuplicateDisplayID はリクエスト内に重複したdisplay_idが含まれる場合に返されるエラーです。
	ErrDuplicateDisplayID = errors.New("duplicate display_id")
)
