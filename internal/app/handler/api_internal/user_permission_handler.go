package api_internal

import (
	"errors"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type updateUserPermissionRequest struct {
	Permission string `json:"permission"`
}

// UserPermissionHandler はADMINによるユーザー権限変更のHTTPリクエストを処理します。
type UserPermissionHandler struct {
	usecase usecase.UserPermissionUsecase
}

// NewUserPermissionHandler は新しいUserPermissionHandlerを生成します。
func NewUserPermissionHandler(usecase usecase.UserPermissionUsecase) *UserPermissionHandler {
	return &UserPermissionHandler{usecase: usecase}
}

// UpdatePermission は指定ユーザーの権限を変更します。
func (h *UserPermissionHandler) UpdatePermission(c *echo.Context) error {
	username, apiErr := apphandler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}

	var request updateUserPermissionRequest
	if err := apphandler.BindStrictJSON(c, &request); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	requester, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		return apierror.ErrUnauthorized
	}

	if err := h.usecase.ChangePermission(c.Request().Context(), requester, username, request.Permission); err != nil {
		switch {
		case errors.Is(err, entity.ErrCannotDemoteOwnAdmin), errors.Is(err, usecase.ErrAdminRequired):
			return apierror.ErrForbidden.WithInternal(err)
		case errors.Is(err, repository.ErrUserConflict):
			return apierror.ErrConflict.WithInternal(err)
		case errors.Is(err, entity.ErrInvalidAccountType):
			return apierror.ErrBadRequest.WithInternal(err)
		default:
			return apierror.FromUsecaseError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
