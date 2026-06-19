package apierror

// エラーコード定数
// クライアント側で機械処理しやすい一意のコードを定義
const (
	// 汎用エラー
	CodeBadRequest    = "bad_request"
	CodeInternalError = "internal_error"

	// 認証・認可エラー
	CodeUnauthorized             = "unauthorized"
	CodeInvalidCredentials       = "invalid_credentials" // #nosec G101
	CodeInvalidToken             = "invalid_token"       // #nosec G101
	CodeInvalidTurnstileToken    = "invalid_turnstile_token"
	CodeTokenExpired             = "token_expired" // #nosec G101
	CodeMissingToken             = "missing_token" // #nosec G101
	CodeForbidden                = "forbidden"
	CodeFirebaseUIDAlreadyLinked = "firebase_uid_already_linked"
	CodeRecentSignInRequired     = "recent_sign_in_required"

	// ユーザー関連エラー
	CodeRegistrationFailed = "registration_failed" // ユーザー登録失敗（詳細を隠蔽）
	CodeUserNotFound       = "user_not_found"      // ユーザーが見つからない（private含む）
	CodeOperationFailed    = "operation_failed"    // 操作失敗（詳細を隠蔽）

	// プレイヤー関連エラー
	CodePlayerNotLinked = "player_not_linked"
	CodePlayerNotFound  = "player_not_found"

	// 楽曲・譜面関連エラー
	CodeSongNotFound         = "song_not_found"
	CodeChartNotFound        = "chart_not_found"
	CodeInvalidGenreID       = "invalid_genre_id"
	CodeInvalidDifficultyID  = "invalid_difficulty_id"
	CodeInvalidDifficulty    = "invalid_difficulty"     // 無効な難易度パラメータ
	CodeDuplicateOfficialIdx = "duplicate_official_idx" // official_idx 重複

	// データ関連エラー
	CodeValidationFailed   = "validation_failed"
	CodeResourceNotFound   = "resource_not_found"
	CodeConflict           = "conflict"
	CodeAPITokenNotFound   = "api_token_not_found" // #nosec G101
	CodePayloadTooLarge    = "payload_too_large"
	CodeUnsupportedMedia   = "unsupported_media_type"
	CodeMethodNotAllowed   = "method_not_allowed"
	CodeNotFound           = "not_found"
	CodeTooManyRequests    = "too_many_requests"
	CodeServiceUnavailable = "service_unavailable"

	// 入力バリデーション詳細エラー
	CodeUsernameEmpty       = "username_empty"
	CodeUsernameTooShort    = "username_too_short"
	CodeUsernameTooLong     = "username_too_long"
	CodeUsernameInvalidChar = "username_invalid_char"

	// 目標関連エラー
	CodeGoalNotFound                 = "goal_not_found"
	CodeGoalLimitExceeded            = "goal_limit_exceeded"
	CodeGoalInvalidTitle             = "goal_invalid_title"
	CodeGoalInvalidAchievementType   = "goal_invalid_achievement_type"
	CodeGoalInvalidAchievementParams = "goal_invalid_achievement_params"
	CodeGoalInvalidAttributes        = "goal_invalid_attributes"
	CodeInvalidGoalInput             = "invalid_goal_input"

	// 保存済みフィルタ関連エラー
	CodeRecordFilterNotFound      = "record_filter_not_found"
	CodeRecordFilterLimitExceeded = "record_filter_limit_exceeded"
	CodeInvalidRecordFilterInput  = "invalid_record_filter_input"
	CodeInvalidRecordFilterID     = "invalid_record_filter_id"
)
