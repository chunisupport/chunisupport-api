package chunirec

import "net/http"

// ChunirecErrorResponse はchunirec互換APIのエラーレスポンス形式です
type ChunirecErrorResponse struct {
	Error ChunirecError `json:"error"`
}

// ChunirecError はエラー詳細を表します
type ChunirecError struct {
	Code              int    `json:"code"`
	Message           string `json:"message"`
	AdditionalMessage string `json:"additional_message"`
}

// NewChunirecErrorResponse は指定されたHTTPステータスコードに対応するchunirec互換エラーレスポンスを生成します
func NewChunirecErrorResponse(statusCode int, additionalMessage string) ChunirecErrorResponse {
	return ChunirecErrorResponse{
		Error: ChunirecError{
			Code:              statusCode,
			Message:           getMessageForStatusCode(statusCode),
			AdditionalMessage: additionalMessage,
		},
	}
}

// getMessageForStatusCode はHTTPステータスコードに対応するメッセージを返します
//
// 注意: chunirec API v2.0の仕様に準拠するため、以下のメッセージはハードコードされています。
// chunirec API v2.0は既に更新が終了しており、仕様変更の可能性はありません。
// 将来的にchunirec側で仕様が変更された場合は、新しいバージョン（v3.0等）として実装します。
// 対応するHTTPステータスコード: 400, 404, 405, 429, 503
func getMessageForStatusCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "bad request."
	case http.StatusNotFound:
		return "not found."
	case http.StatusMethodNotAllowed:
		return "the requested method is not allowed."
	case http.StatusTooManyRequests:
		return "too many requests."
	case http.StatusServiceUnavailable:
		return "service unavailable."
	default:
		// 想定外のステータスコードの場合は503として扱う
		return "service unavailable."
	}
}

// NewBadRequestError は400 Bad Requestエラーを生成します
func NewBadRequestError(additionalMessage string) ChunirecErrorResponse {
	return NewChunirecErrorResponse(http.StatusBadRequest, additionalMessage)
}

// NewNotFoundError は404 Not Foundエラーを生成します
func NewNotFoundError(additionalMessage string) ChunirecErrorResponse {
	return NewChunirecErrorResponse(http.StatusNotFound, additionalMessage)
}

// NewMethodNotAllowedError は405 Method Not Allowedエラーを生成します
func NewMethodNotAllowedError(additionalMessage string) ChunirecErrorResponse {
	return NewChunirecErrorResponse(http.StatusMethodNotAllowed, additionalMessage)
}

// NewTooManyRequestsError は429 Too Many Requestsエラーを生成します
func NewTooManyRequestsError(additionalMessage string) ChunirecErrorResponse {
	return NewChunirecErrorResponse(http.StatusTooManyRequests, additionalMessage)
}

// NewServiceUnavailableError は503 Service Unavailableエラーを生成します
func NewServiceUnavailableError(additionalMessage string) ChunirecErrorResponse {
	return NewChunirecErrorResponse(http.StatusServiceUnavailable, additionalMessage)
}
