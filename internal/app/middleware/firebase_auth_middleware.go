package middleware

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v4"
)

// FirebaseAuthenticator はFirebase IDトークンからユーザーを解決する最小インターフェースです。
type FirebaseAuthenticator interface {
	Authenticate(ctx context.Context, idToken string) (*entity.User, error)
}

// FirebaseIDTokenMiddleware はBearerのFirebase IDトークン認証を行います。
func FirebaseIDTokenMiddleware(authenticator FirebaseAuthenticator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			idToken := extractBearerToken(c)
			if idToken == "" {
				return apierror.ErrMissingToken
			}
			if authenticator == nil {
				return apierror.ErrInternalError.WithInternal(errors.New("firebase authenticator is nil"))
			}

			user, err := authenticator.Authenticate(c.Request().Context(), idToken)
			if err != nil {
				return apierror.FromUsecaseError(err)
			}
			if user == nil {
				return apierror.ErrInternalError.WithInternal(errors.New("firebase authenticator returned nil user"))
			}

			c.Set("userEntity", user)
			return next(c)
		}
	}
}
