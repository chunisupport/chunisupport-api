package repository

import "errors"

// リポジトリ層の共通エラー定義
// infraからusecaseへの依存を避けるため、ドメイン関連エラーはここで定義します。
var (
	// ErrUserNotFound はユーザーが見つからなかった場合に返されるエラーです。
	ErrUserNotFound = errors.New("user not found")

	// ErrSessionNotFound はセッションが見つからなかった場合に返されるエラーです。
	ErrSessionNotFound = errors.New("session not found")

	// ErrPlayerNotFound はプレイヤーが見つからなかった場合に返されるエラーです。
	ErrPlayerNotFound = errors.New("player not found")

	// ErrRecoveryCodeNotFound はリカバリーコードが見つからなかった場合に返されるエラーです。
	ErrRecoveryCodeNotFound = errors.New("recovery code not found")

	// ErrAPITokenNotFound はAPIトークンが見つからなかった場合に返されるエラーです。
	ErrAPITokenNotFound = errors.New("api token not found")

	// ErrGoalNotFound は目標が見つからなかった場合に返されるエラーです。
	ErrGoalNotFound = errors.New("goal not found")

	// ErrSongNotFound は楽曲が見つからなかった場合に返されるエラーです。
	ErrSongNotFound = errors.New("song not found")

	// ErrDuplicateDisplayID はリクエスト内に重複したdisplay_idが含まれる場合に返されるエラーです。
	ErrDuplicateDisplayID = errors.New("duplicate display_id")

	// ErrFirebaseUIDAlreadyLinked は Firebase UID が他ユーザーへ既に紐付いている場合に返されるエラーです。
	ErrFirebaseUIDAlreadyLinked = errors.New("firebase uid already linked")
)
