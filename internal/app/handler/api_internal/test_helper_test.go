package api_internal_test

import (
	appinternal "github.com/chunisupport/chunisupport-api/internal/app"
	"github.com/labstack/echo/v4"
)

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = appinternal.NewCustomValidator()
	return e
}
