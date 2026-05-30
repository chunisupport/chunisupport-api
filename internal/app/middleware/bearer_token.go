package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/app/httpheader"

	"github.com/labstack/echo/v4"
)

func extractBearerToken(c echo.Context) string {
	return httpheader.ExtractBearerToken(c.Request().Header)
}
