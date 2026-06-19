package repository

import "errors"

// リポジトリ層の共通エラー定義
// infraからusecaseへの依存を避けるため、ドメイン関連エラーはここで定義します。
var (
	// ErrUserNotFound はユーザーが見つからなかった場合に返されるエラーです。
	ErrUserNotFound = errors.New("user not found")

	// ErrUserConflict はユーザー更新時の前提条件が一致しなかった場合に返されるエラーです。
	ErrUserConflict = errors.New("user conflict")

	// ErrPlayerNotFound はプレイヤーが見つからなかった場合に返されるエラーです。
	ErrPlayerNotFound = errors.New("player not found")

	// ErrAPITokenNotFound はAPIトークンが見つからなかった場合に返されるエラーです。
	ErrAPITokenNotFound = errors.New("api token not found")

	// ErrGoalNotFound は目標が見つからなかった場合に返されるエラーです。
	ErrGoalNotFound = errors.New("goal not found")

	// ErrRecordFilterNotFound は譜面フィルタが見つからなかった場合に返されるエラーです。
	ErrRecordFilterNotFound = errors.New("record filter not found")

	// ErrHonorNotFound は称号が見つからなかった場合に返されるエラーです。
	ErrHonorNotFound = errors.New("honor not found")

	// ErrHonorConflict は称号の一意制約または参照制約により操作できない場合に返されるエラーです。
	ErrHonorConflict = errors.New("honor conflict")

	// ErrSongNotFound は楽曲が見つからなかった場合に返されるエラーです。
	ErrSongNotFound = errors.New("song not found")

	// ErrDuplicateDisplayID はリクエスト内に重複したdisplay_idが含まれる場合に返されるエラーです。
	ErrDuplicateDisplayID = errors.New("duplicate display_id")

	// ErrFirebaseUIDAlreadyLinked は Firebase UID が他ユーザーへ既に紐付いている場合に返されるエラーです。
	ErrFirebaseUIDAlreadyLinked = errors.New("firebase uid already linked")

	// ErrDuplicateUsername はユーザー名が既に使用されている場合に返されるエラーです。
	// 事前チェックをすり抜けた競合状態でのINSERT失敗時に使用します。
	ErrDuplicateUsername = errors.New("username already taken")

	// ErrTemporaryPlayerDataNotFound は一時プレイヤーデータが見つからない場合に返されるエラーです。
	ErrTemporaryPlayerDataNotFound = errors.New("temporary player data not found")

	// ErrTemporaryPlayerDataPerIPLimitExceeded はIP単位の一時データ上限を超えた場合に返されるエラーです。
	ErrTemporaryPlayerDataPerIPLimitExceeded = errors.New("temporary player data per ip limit exceeded")

	// ErrTemporaryPlayerDataTotalSizeLimitExceeded は一時データ総量の上限を超えた場合に返されるエラーです。
	ErrTemporaryPlayerDataTotalSizeLimitExceeded = errors.New("temporary player data total size limit exceeded")

	// ErrDuplicateOfficialIdx は official_idx が既に使用されている場合に返されるエラーです。
	ErrDuplicateOfficialIdx = errors.New("official_idx already exists")

	// ErrRepositoryOperationFailed はリポジトリ操作が永続化層の事情で失敗した場合に返されるエラーです。
	ErrRepositoryOperationFailed = errors.New("repository operation failed")
)
