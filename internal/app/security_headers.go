package app

import (
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

// newSecurityHeadersMiddleware は、すべての応答にセキュリティヘッダーを付与するミドルウェアを返します。
// 付与元をアプリに一本化し、nginxやCloudflareとの二重付与や環境ごとの差異をなくすため、ここで値を固定します。
// X-XSS-Protection は現行ブラウザで廃止され、古い実装では副作用があるため付与しません。
// HSTS はアプリがTLSを終端せず、ドメイン単位で決める方針なのでCloudflareで付与します。
func newSecurityHeadersMiddleware() echo.MiddlewareFunc {
	return echoMiddleware.SecureWithConfig(echoMiddleware.SecureConfig{
		ContentTypeNosniff:    info.SecurityHeaderContentTypeOptions,
		XFrameOptions:         info.SecurityHeaderFrameOptions,
		ContentSecurityPolicy: info.SecurityHeaderContentSecurityPolicy,
		ReferrerPolicy:        info.SecurityHeaderReferrerPolicy,
	})
}
