package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// APITokenMiddleware は外部API向けのトークン認証を提供します。
func APITokenMiddleware(usecase usecase.APITokenUsecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rawToken := extractBearerToken(c)
			if rawToken == "" {
				return apierror.ErrMissingToken
			}

			user, token, err := usecase.Validate(c.Request().Context(), rawToken)
			if err != nil {
				return apierror.FromUsecaseError(err)
			}

			c.Set("userEntity", user)
			c.Set("apiToken", token)
			return next(c)
		}
	}
}

// OptionalAPITokenMiddleware はAPIトークンが指定された場合だけ認証します。
func OptionalAPITokenMiddleware(usecase usecase.APITokenUsecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rawToken := extractBearerToken(c)
			if rawToken == "" {
				return next(c)
			}

			user, token, err := usecase.Validate(c.Request().Context(), rawToken)
			if err != nil {
				return apierror.FromUsecaseError(err)
			}

			c.Set("userEntity", user)
			c.Set("apiToken", token)
			return next(c)
		}
	}
}
