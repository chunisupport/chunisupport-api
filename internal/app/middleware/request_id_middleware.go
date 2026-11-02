package middleware

import (
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/logger"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// RequestIDMiddleware は X-Request-ID を受理し、無いまたは不正な場合はUUIDを生成します。
// 不正値でリクエスト自体を拒否すると、その失敗を相関できなくなるため、安全なIDへ置き換えます。
// アクセスログは処理後にリクエストヘッダーを読むので、確定したIDをヘッダーへ書き戻します。
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			request := c.Request()
			requestID := resolveRequestID(request.Header.Get(echo.HeaderXRequestID))

			request.Header.Set(echo.HeaderXRequestID, requestID)
			c.Response().Header().Set(echo.HeaderXRequestID, requestID)
			c.SetRequest(request.WithContext(logger.WithRequestID(request.Context(), requestID)))

			return next(c)
		}
	}
}

// resolveRequestID は外部IDを検証し、受理できない場合はサーバー生成のUUIDを返します。
func resolveRequestID(headerValue string) string {
	candidate := strings.TrimSpace(headerValue)
	if acceptableRequestID(candidate) {
		return candidate
	}
	return uuid.NewString()
}

// acceptableRequestID はログ注入やヘッダー肥大にならないIDだけを受理します。
// 先頭は英数字、以降は英数字と . _ - のみです。
func acceptableRequestID(id string) bool {
	if id == "" || len(id) > info.RequestIDMaxLength {
		return false
	}
	for i := 0; i < len(id); i++ {
		if isRequestIDAlphaNum(id[i]) {
			continue
		}
		if i > 0 && (id[i] == '.' || id[i] == '_' || id[i] == '-') {
			continue
		}
		return false
	}
	return true
}

func isRequestIDAlphaNum(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}
