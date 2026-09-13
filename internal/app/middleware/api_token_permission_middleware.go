package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
)

// RequireAPITokenWrite はAPIトークンに更新権限があることを要求します。
// APIトークン認証の後に使用し、ユーザーのロール要件は別のRequireRoleで判定します。
func RequireAPITokenWrite() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			token, ok := c.Get(contextKeyAPIToken).(*entity.APIToken)
			if !ok || token == nil {
				return apierror.ErrUnauthorized
			}
			if !token.CanWrite() {
				return apierror.ErrForbidden
			}
			return next(c)
		}
	}
}
