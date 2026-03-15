package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v4"
)

// RequireRole は指定された権限レベル以上を要求するミドルウェアを返します。
// JWTMiddleware の後に使用することを想定しています。
func RequireRole(minRoleID int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
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
			if !info.HasRole(user.AccountTypeID, minRoleID) {
				return apierror.ErrForbidden
			}

			return next(c)
		}
	}
}
