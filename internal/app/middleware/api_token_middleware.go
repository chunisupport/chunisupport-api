package middleware

import (
	"strings"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// APITokenMiddleware は外部API向けのトークン認証を提供します。
func APITokenMiddleware(usecase usecase.APITokenUsecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rawToken := extractAPIToken(c)
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

func extractAPIToken(c echo.Context) string {
	authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token := strings.TrimSpace(parts[1])
			if token != "" {
				return token
			}
		}
	}

	return ""
}
