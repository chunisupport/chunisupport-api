package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
)

// RequireRole は指定されたロール要件を満たすことを要求するミドルウェアを返します。
// userEntity を Context に設定する認証ミドルウェアの後に使用することを想定しています。
func RequireRole(requiredRoleID int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Contextからログインユーザー情報を取得
			userObj := c.Get("userEntity")
			if userObj == nil {
				return apierror.ErrUnauthorized
			}

			user, ok := userObj.(*entity.User)
			if !ok {
				return apierror.ErrUnauthorized
			}

			// 権限チェック（未知ロールIDは拒否）
			if !info.HasRole(user.AccountTypeID, requiredRoleID) {
				return apierror.ErrForbidden
			}

			return next(c)
		}
	}
}
