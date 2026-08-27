package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// APITokenMiddleware は外部API向けのトークン認証を提供します。
func APITokenMiddleware(usecase usecase.APITokenUsecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if hasResolvedAPIToken(c) {
				return next(c)
			}

			rawToken := extractBearerToken(c)
			if rawToken == "" {
				return apierror.ErrMissingToken
			}

			user, token, err := usecase.Validate(c.Request().Context(), rawToken)
			if err != nil {
				return apierror.FromUsecaseError(err)
			}

			c.Set(contextKeyUserEntity, user)
			c.Set(contextKeyAPIToken, token)
			return next(c)
		}
	}
}

func hasResolvedAPIToken(c *echo.Context) bool {
	user, userOK := c.Get(contextKeyUserEntity).(*entity.User)
	token, tokenOK := c.Get(contextKeyAPIToken).(*entity.APIToken)
	return userOK && user != nil && tokenOK && token != nil
}
