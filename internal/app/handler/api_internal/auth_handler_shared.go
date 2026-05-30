package api_internal

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v4"
)

func getUserEntityFromContext(c echo.Context) (*entity.User, error) {
	user, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		return nil, apierror.ErrUnauthorized
	}
	return user, nil
}
