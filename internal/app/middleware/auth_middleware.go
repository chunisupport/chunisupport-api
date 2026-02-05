package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// JWTMiddleware はJWT認証を行うミドルウェアを返します。
func JWTMiddleware(secret string, authUsecase usecase.AuthUsecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("token")
			if err != nil {
				return apierror.ErrMissingToken
			}

			tokenString := cookie.Value
			claims, err := auth.ValidateToken(tokenString, secret)
			if err != nil {
				return apierror.ErrInvalidToken.WithInternal(err)
			}

			// セッションの有効性を検証
			user, err := authUsecase.Authenticate(c.Request().Context(), claims.UserID, claims.SessionID)
			if err != nil {
				return apierror.FromUsecaseError(err)
			}

			// コンテキストにuserエンティティとclaimsをセットして後続のハンドラで利用できるようにする
			// Note: userエンティティもセットすることで、ハンドラ側で再取得する手間を省く
			c.Set("userEntity", user)
			c.Set("user", claims) // 既存の処理との互換性のため残す
			return next(c)
		}
	}
}

// OptionalJWTMiddleware はCookieが存在する場合のみJWT認証を行います。
func OptionalJWTMiddleware(secret string, authUsecase usecase.AuthUsecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("token")
			if err != nil {
				return next(c)
			}

			tokenString := cookie.Value
			claims, err := auth.ValidateToken(tokenString, secret)
			if err != nil {
				return apierror.ErrInvalidToken.WithInternal(err)
			}

			user, err := authUsecase.Authenticate(c.Request().Context(), claims.UserID, claims.SessionID)
			if err != nil {
				return apierror.FromUsecaseError(err)
			}

			c.Set("userEntity", user)
			c.Set("user", claims)
			return next(c)
		}
	}
}
