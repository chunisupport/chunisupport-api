package apierror

import (
	"fmt"
	"net/http"
)

// APIError はAPIエラーレスポンスを表す構造体
// クライアントにはCodeのみを返し、詳細はサーバーログに記録する
type APIError struct {
	Code       string // 機械処理用エラーコード (例: "invalid_token")
	HTTPStatus int    // HTTPステータスコード
	Internal   error  // 内部エラー（ログ用、レスポンスには含めない）
}

// Error はerrorインターフェースを実装
func (e *APIError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Internal)
	}
	return e.Code
}

// Unwrap は内部エラーを返す（errors.Unwrap対応）
func (e *APIError) Unwrap() error {
	return e.Internal
}

// WithInternal は内部エラーを設定した新しいAPIErrorを返す
func (e *APIError) WithInternal(err error) *APIError {
	return &APIError{
		Code:       e.Code,
		HTTPStatus: e.HTTPStatus,
		Internal:   err,
	}
}

// New は新しいAPIErrorを生成する
func New(code string, httpStatus int) *APIError {
	return &APIError{
		Code:       code,
		HTTPStatus: httpStatus,
	}
}

// NewWithInternal は内部エラー付きのAPIErrorを生成する
func NewWithInternal(code string, httpStatus int, internal error) *APIError {
	return &APIError{
		Code:       code,
		HTTPStatus: httpStatus,
		Internal:   internal,
	}
}

// 事前定義済みAPIエラー（よく使うものを定義）
var (
	// 汎用エラー
	ErrBadRequest    = New(CodeBadRequest, http.StatusBadRequest)
	ErrInternalError = New(CodeInternalError, http.StatusInternalServerError)

	// 認証・認可エラー
	ErrUnauthorized             = New(CodeUnauthorized, http.StatusUnauthorized)
	ErrInvalidCredentials       = New(CodeInvalidCredentials, http.StatusUnauthorized)
	ErrInvalidToken             = New(CodeInvalidToken, http.StatusUnauthorized)
	ErrTokenExpired             = New(CodeTokenExpired, http.StatusUnauthorized)
	ErrMissingToken             = New(CodeMissingToken, http.StatusUnauthorized)
	ErrForbidden                = New(CodeForbidden, http.StatusForbidden)
	ErrFirebaseUIDAlreadyLinked = New(CodeFirebaseUIDAlreadyLinked, http.StatusConflict)

	// ユーザー関連エラー
	ErrRegistrationFailed = New(CodeRegistrationFailed, http.StatusBadRequest) // 409→400に変更
	ErrUserNotFound       = New(CodeUserNotFound, http.StatusNotFound)         // private含む
	ErrOperationFailed    = New(CodeOperationFailed, http.StatusBadRequest)    // 削除系操作失敗

	// プレイヤー関連エラー
	ErrPlayerNotLinked = New(CodePlayerNotLinked, http.StatusNotFound)
	ErrPlayerNotFound  = New(CodePlayerNotFound, http.StatusNotFound)

	// 楽曲・譜面関連エラー
	ErrSongNotFound      = New(CodeSongNotFound, http.StatusNotFound)
	ErrChartNotFound     = New(CodeChartNotFound, http.StatusNotFound)
	ErrInvalidDifficulty = New(CodeInvalidDifficulty, http.StatusBadRequest)

	// データ関連エラー
	ErrValidationFailed   = New(CodeValidationFailed, http.StatusUnprocessableEntity)
	ErrResourceNotFound   = New(CodeResourceNotFound, http.StatusBadRequest)
	ErrConflict           = New(CodeConflict, http.StatusConflict)
	ErrAPITokenNotFound   = New(CodeAPITokenNotFound, http.StatusNotFound)
	ErrPayloadTooLarge    = New(CodePayloadTooLarge, http.StatusRequestEntityTooLarge)
	ErrUnsupportedMedia   = New(CodeUnsupportedMedia, http.StatusUnsupportedMediaType)
	ErrMethodNotAllowed   = New(CodeMethodNotAllowed, http.StatusMethodNotAllowed)
	ErrNotFound           = New(CodeNotFound, http.StatusNotFound)
	ErrTooManyRequests    = New(CodeTooManyRequests, http.StatusTooManyRequests)
	ErrServiceUnavailable = New(CodeServiceUnavailable, http.StatusServiceUnavailable)

	// 入力バリデーション詳細エラー
	ErrUsernameEmpty         = New(CodeUsernameEmpty, http.StatusBadRequest)
	ErrUsernameTooShort      = New(CodeUsernameTooShort, http.StatusBadRequest)
	ErrUsernameTooLong       = New(CodeUsernameTooLong, http.StatusBadRequest)
	ErrUsernameInvalidChar   = New(CodeUsernameInvalidChar, http.StatusBadRequest)
	ErrAppVersionUnsupported = New(CodeAppVersionUnsupported, http.StatusBadRequest) // 対応していないアプリバージョン

	ErrGoalNotFound                 = New(CodeGoalNotFound, http.StatusNotFound)
	ErrGoalLimitExceeded            = New(CodeGoalLimitExceeded, http.StatusBadRequest)
	ErrGoalInvalidTitle             = New(CodeGoalInvalidTitle, http.StatusBadRequest)
	ErrGoalInvalidAchievementType   = New(CodeGoalInvalidAchievementType, http.StatusBadRequest)
	ErrGoalInvalidAchievementParams = New(CodeGoalInvalidAchievementParams, http.StatusBadRequest)
	ErrGoalInvalidAttributes        = New(CodeGoalInvalidAttributes, http.StatusBadRequest)
	ErrInvalidGoalInput             = New(CodeInvalidGoalInput, http.StatusBadRequest)
)

// ErrorResponse はエラーレスポンスの構造体です
// クライアントにはエラーコードとステータスを返します
type ErrorResponse struct {
	Error struct {
		Status  int                     `json:"status"`
		Code    string                  `json:"code"`
		Message string                  `json:"message,omitempty"`
		Details []ValidationErrorDetail `json:"details,omitempty"`
	} `json:"error"`
}

// ValidationErrorDetail は入力バリデーション失敗時の詳細情報です。
// 入力フォーマットに関する情報のみを返し、認証成否などの機微情報は含めません。
type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
