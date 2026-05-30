package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/labstack/echo/v4"
)

// CustomHTTPErrorHandler はカスタムエラーハンドラーです
func CustomHTTPErrorHandler(err error, c echo.Context) {
	var apiErr *apierror.APIError
	var httpStatus int
	var errorCode string
	errorMessage := ""
	var errorDetails []apierror.ValidationErrorDetail

	// APIErrorの場合
	if errors.As(err, &apiErr) {
		httpStatus = apiErr.HTTPStatus
		errorCode = apiErr.Code
		errorMessage, errorDetails = buildClientErrorInfo(apiErr)
	} else if he, ok := err.(*echo.HTTPError); ok {
		// echo.HTTPErrorの場合（フォールバック）
		httpStatus = he.Code
		errorCode = httpStatusToErrorCode(he.Code)
	} else {
		// その他のエラー
		httpStatus = http.StatusInternalServerError
		errorCode = apierror.CodeInternalError
	}

	// レスポンスがすでに送信されている場合は何もしない
	if c.Response().Committed {
		return
	}

	// エラーログの出力（詳細情報を含む）
	logError(httpStatus, errorCode, err, c)

	// エラーレスポンスの送信（コードとステータス）
	if err := c.JSON(httpStatus, apierror.ErrorResponse{
		Error: struct {
			Status  int                              `json:"status"`
			Code    string                           `json:"code"`
			Message string                           `json:"message,omitempty"`
			Details []apierror.ValidationErrorDetail `json:"details,omitempty"`
		}{
			Status:  httpStatus,
			Code:    errorCode,
			Message: errorMessage,
			Details: errorDetails,
		},
	}); err != nil {
		slog.Error("Failed to send error response", "error", err)
	}
}

func buildClientErrorInfo(apiErr *apierror.APIError) (string, []apierror.ValidationErrorDetail) {
	if apiErr == nil || apiErr.Code != apierror.CodeValidationFailed {
		return "", nil
	}

	if apiErr.Internal == nil {
		return "入力値の形式を確認してください。", nil
	}

	var validationErrors apierror.ValidationErrors
	if !errors.As(apiErr.Internal, &validationErrors) {
		return "入力値の形式を確認してください。", nil
	}

	return "入力値の形式に誤りがあります。", validationErrors.Details()
}

// httpStatusToErrorCode はHTTPステータスコードからエラーコードを生成します
// echo.HTTPErrorなど、APIError以外のエラーに対するフォールバック用
func httpStatusToErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return apierror.CodeBadRequest
	case http.StatusUnauthorized:
		return apierror.CodeUnauthorized
	case http.StatusForbidden:
		return apierror.CodeForbidden
	case http.StatusNotFound:
		return apierror.CodeNotFound
	case http.StatusMethodNotAllowed:
		return apierror.CodeMethodNotAllowed
	case http.StatusConflict:
		return apierror.CodeConflict
	case http.StatusRequestEntityTooLarge:
		return apierror.CodePayloadTooLarge
	case http.StatusUnsupportedMediaType:
		return apierror.CodeUnsupportedMedia
	case http.StatusUnprocessableEntity:
		return apierror.CodeValidationFailed
	case http.StatusTooManyRequests:
		return apierror.CodeTooManyRequests
	case http.StatusServiceUnavailable:
		return apierror.CodeServiceUnavailable
	default:
		if status >= 500 {
			return apierror.CodeInternalError
		}
		return apierror.CodeBadRequest
	}
}

// logError はエラーをログに出力します（詳細情報を含む）
func logError(status int, code string, err error, c echo.Context) {
	errorMessage := sanitizeLogValue(err.Error())
	logger := slog.With("method", c.Request().Method, "path", c.Request().URL.Path, "remote_addr", c.RealIP())
	// context.Canceled の場合はクライアントキャンセルとしてWARNログ
	if errors.Is(err, context.Canceled) {
		logger.Warn("HTTP request canceled by client",
			"status", status,
			"code", code,
			"error", errorMessage,
		)
		return
	}

	// 4xx系は警告、5xx系はエラーとして出力
	if status >= 500 {
		logger.Error("HTTP error occurred",
			"status", status,
			"code", code,
			"error", errorMessage,
		)
	} else if status >= 400 {
		logger.Warn("HTTP client error",
			"status", status,
			"code", code,
			"error", errorMessage,
		)
	}
}

func sanitizeLogValue(value string) string {
	replacer := strings.NewReplacer("\n", " ", "\r", " ")
	return replacer.Replace(value)
}
