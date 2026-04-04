package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func extractBearerToken(c echo.Context) string {
	authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
	if authHeader == "" {
		return ""
	}

	scheme, token, found := strings.Cut(authHeader, " ")
	if !found || !strings.EqualFold(strings.TrimSpace(scheme), "Bearer") {
		return ""
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}

	return token
}
