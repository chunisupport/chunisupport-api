package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v4"
)

// AccountType の定数定義
const (
	AccountTypePlayer = 1
	AccountTypeEditor = 2
	AccountTypeAdmin  = 3
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

			// 権限チェック
			if user.AccountTypeID < minRoleID {
				return apierror.ErrForbidden
			}

			return next(c)
		}
	}
}
