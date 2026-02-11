package chunirec

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/labstack/echo/v4"
)

// ChunirecErrorHandlerMiddleware はchunirec互換API専用のエラーハンドリングミドルウェアです
// このミドルウェアは、標準のエラーハンドラーをchunirec互換形式でオーバーライドします
func ChunirecErrorHandlerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				// エラーをchunirec互換形式で処理
				handleChunirecError(err, c)
				return nil // エラーを握りつぶして、デフォルトのエラーハンドラーに渡さない
			}
			return nil
		}
	}
}

// handleChunirecError はエラーをchunirec互換形式でレスポンスします
func handleChunirecError(err error, c echo.Context) {
	var httpStatus int
	var additionalMessage string

	// レスポンスがすでに送信されている場合は何もしない
	if c.Response().Committed {
		return
	}

	// エラーの種類に応じてHTTPステータスコードを決定
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		httpStatus = apiErr.HTTPStatus
		// APIErrorのコードを追加メッセージとして使用（オプション）
		additionalMessage = ""
	} else if he, ok := err.(*echo.HTTPError); ok {
		httpStatus = he.Code
		additionalMessage = ""
	} else {
		// その他のエラーは503として扱う
		httpStatus = http.StatusServiceUnavailable
		additionalMessage = ""
	}

	// エラーログの出力（詳細情報を含む）
	logChunirecError(httpStatus, err, c)

	// chunirec互換形式でエラーレスポンスを送信
	// 注意: chunirec API 2.0の仕様に準拠するため、以下の6パターンのステータスコードのみをサポートします。
	// chunirec API 2.0は既に更新が終了しており、仕様変更の可能性はありません。
	// 対応するHTTPステータスコード: 400, 404, 405, 429, 503
	// それ以外のステータスコードは503として処理されます。
	var errorResponse ChunirecErrorResponse
	switch httpStatus {
	case http.StatusBadRequest:
		errorResponse = NewBadRequestError(additionalMessage)
	case http.StatusNotFound:
		errorResponse = NewNotFoundError(additionalMessage)
	case http.StatusMethodNotAllowed:
		errorResponse = NewMethodNotAllowedError(additionalMessage)
	case http.StatusTooManyRequests:
		errorResponse = NewTooManyRequestsError(additionalMessage)
	case http.StatusServiceUnavailable:
		errorResponse = NewServiceUnavailableError(additionalMessage)
	default:
		// 想定外のステータスコードは503として扱う
		httpStatus = http.StatusServiceUnavailable
		errorResponse = NewServiceUnavailableError(additionalMessage)
	}

	if err := c.JSON(httpStatus, errorResponse); err != nil {
		slog.Error("Failed to send chunirec error response", "error", err)
	}
}

// logChunirecError はエラーをログに出力します（詳細情報を含む）
func logChunirecError(status int, err error, c echo.Context) {
	errorMessage := sanitizeLogValue(err.Error())
	logger := slog.With("method", c.Request().Method, "path", c.Request().URL.Path, "remote_addr", c.RealIP())
	// context.Canceled の場合はクライアントキャンセルとしてWARNログ
	if errors.Is(err, context.Canceled) {
		logger.Warn("Chunirec HTTP request canceled by client",
			"status", status,
			"error", errorMessage,
		)
		return
	}

	// 5xx系エラーはERRORログ
	if status >= 500 {
		logger.Error("Chunirec HTTP error",
			"status", status,
			"error", errorMessage,
		)
		return
	}

	// 4xx系エラーはWARNログ
	logger.Warn("Chunirec HTTP client error",
		"status", status,
		"error", errorMessage,
	)
}

func sanitizeLogValue(value string) string {
	replacer := strings.NewReplacer("\n", " ", "\r", " ")
	return replacer.Replace(value)
}
