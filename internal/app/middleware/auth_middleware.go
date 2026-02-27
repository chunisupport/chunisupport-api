package middleware

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v4"
)

// Authenticator はJWT検証後にセッションの有効性を確認するための最小インターフェースです。
type Authenticator interface {
	Authenticate(ctx context.Context, userID int, sessionID string) (*entity.User, error)
}

// JWTMiddleware はJWT認証を行うミドルウェアを返します。
func JWTMiddleware(secret string, authenticator Authenticator) echo.MiddlewareFunc {
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

			user, err := authenticator.Authenticate(c.Request().Context(), claims.UserID, claims.SessionID)
			if err != nil {
				return apierror.FromUsecaseError(err)
			}

			c.Set("userEntity", user)
			c.Set("user", claims)
			return next(c)
		}
	}
}

// OptionalJWTMiddleware はCookieが存在する場合のみJWT認証を行います。
func OptionalJWTMiddleware(secret string, authenticator Authenticator) echo.MiddlewareFunc {
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

			user, err := authenticator.Authenticate(c.Request().Context(), claims.UserID, claims.SessionID)
			if err != nil {
				return apierror.FromUsecaseError(err)
			}

			c.Set("userEntity", user)
			c.Set("user", claims)
			return next(c)
		}
	}
}
