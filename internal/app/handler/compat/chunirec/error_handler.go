package chunirec

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
)

// ChunirecErrorHandlerMiddleware はchunirec互換API専用のエラーハンドリングミドルウェアです
// このミドルウェアは、標準のエラーハンドラーをchunirec互換形式でオーバーライドします
func ChunirecErrorHandlerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := next(c)
			if err != nil {
				handleChunirecError(err, c)
				// 互換形式で応答済みのため、Echo標準ハンドラーへエラーを渡しません。
				return nil
			}
			return nil
		}
	}
}

// handleChunirecError はエラーをchunirec互換形式でレスポンスします
func handleChunirecError(err error, c *echo.Context) {
	var httpStatus int
	var additionalMessage string

	if response, _ := echo.UnwrapResponse(c.Response()); response != nil && response.Committed {
		return
	}

	// クライアント切断はサーバー障害ではないため、互換形式のエラー応答は返さずに記録だけ残します。
	if errors.Is(err, context.Canceled) {
		slog.InfoContext(c.Request().Context(), "Chunirec HTTP request canceled by client",
			"method", c.Request().Method,
			"path", c.Request().URL.Path,
			"remote_addr", c.RealIP(),
			"status", info.StatusClientClosedRequest,
			"error", sanitizeLogValue(err.Error()),
		)
		if err := c.NoContent(info.StatusClientClosedRequest); err != nil {
			slog.DebugContext(c.Request().Context(), "Failed to send chunirec client closed response", "error", err)
		}
		return
	}

	// エラーの種類に応じてHTTPステータスコードを決定
	var apiErr *apierror.APIError
	var statusCoder echo.HTTPStatusCoder
	if errors.As(err, &apiErr) {
		httpStatus = apiErr.HTTPStatus
		additionalMessage = ""
	} else if errors.As(err, &statusCoder) {
		// Echo v5のルート不一致（ErrNotFound等）は*echo.HTTPErrorではないため、ステータスを持つエラー全般で判定します。
		httpStatus = statusCoder.StatusCode()
		additionalMessage = ""
	} else {
		httpStatus = http.StatusServiceUnavailable
		additionalMessage = ""
	}

	logChunirecError(httpStatus, err, c)

	// chunirec互換形式でエラーレスポンスを送信
	// 注意: chunirec API 2.0の仕様に準拠するため、以下の5種類のステータスコードのみをサポートします。
	// chunirec API 2.0は既に更新が終了しており、仕様変更の可能性はありません。
	// 対応するHTTPステータスコード: 400, 404, 405, 429, 503
	// それ以外のステータスコードは503として処理されます。
	// ただしクライアント切断時は応答が届かないため、この変換より前に499を返しています。
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
		httpStatus = http.StatusServiceUnavailable
		errorResponse = NewServiceUnavailableError(additionalMessage)
	}

	if err := c.JSON(httpStatus, errorResponse); err != nil {
		slog.ErrorContext(c.Request().Context(), "Failed to send chunirec error response", "error", err)
	}
}

// logChunirecError はエラーをログに出力します（詳細情報を含む）
func logChunirecError(status int, err error, c *echo.Context) {
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) && apiErr.Code == apierror.CodeMaintenanceMode {
		return
	}

	errorMessage := sanitizeLogValue(err.Error())
	logger := slog.With("method", c.Request().Method, "path", c.Request().URL.Path, "remote_addr", c.RealIP()).With(middleware.UserLogAttrs(c)...)
	ctx := c.Request().Context()

	if status >= 500 {
		logger.ErrorContext(ctx, "Chunirec HTTP error",
			"status", status,
			"error", errorMessage,
		)
		return
	}

	logger.WarnContext(ctx, "Chunirec HTTP client error",
		"status", status,
		"error", errorMessage,
	)
}

func sanitizeLogValue(value string) string {
	replacer := strings.NewReplacer("\n", " ", "\r", " ")
	return replacer.Replace(value)
}
