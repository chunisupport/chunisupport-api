package usecase

import "errors"

var (
	ErrUsernameTaken              = errors.New("this username is already taken") // 内部用エラー
	ErrInvalidCredentials         = errors.New("invalid username or password")   // ログイン失敗
	ErrUserIDMismatch             = errors.New("user ID mismatch")               // 権限エラー
	ErrInvalidSession             = errors.New("session invalid or expired")     // セッション無効統合
	ErrUserDeleted                = errors.New("user deleted")                   // ユーザー削除済み
	ErrUserNotFound               = errors.New("user not found")                 // ユーザー未発見（Private含む）
	ErrPlayerNotLinked            = errors.New("player not linked to user")      // プレイヤー未紐付
	ErrUserPrivate                = errors.New("user profile is private")        // 内部用：非公開プロファイル
	ErrIncorrectPassword          = errors.New("current password is incorrect")  // パスワード不一致
	ErrInvalidPassword            = errors.New("invalid password")               // パスワード無効（詳細隠蔽）
	ErrOperationFailed            = errors.New("operation failed")               // 操作失敗（詳細隠蔽）
	ErrInvalidRecoveryCredentials = errors.New("invalid recovery credentials")   // リカバリー失敗（詳細隠蔽）
	ErrUserAlreadyDeleted         = errors.New("user already deleted")           // 内部用
	ErrUserNotDeleted             = errors.New("user not deleted")               // 内部用

	// ユーザー名バリデーションエラー
	ErrUsernameEmpty       = errors.New("username cannot be empty")
	ErrUsernameTooShort    = errors.New("username must be at least 5 characters")
	ErrUsernameTooLong     = errors.New("username must be 50 characters or less")
	ErrUsernameInvalidChar = errors.New("username can only contain lowercase letters and numbers")

	// パスワードバリデーションエラー
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong  = errors.New("password must be 128 characters or less")
)
