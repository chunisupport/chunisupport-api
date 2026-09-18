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

type updateUserSuspiciousRequest struct {
	IsSuspicious *bool `json:"is_suspicious"`
}

// UserSuspiciousHandler はADMINによる不審アカウントフラグ変更のHTTPリクエストを処理します。
type UserSuspiciousHandler struct {
	usecase usecase.UserSuspiciousUsecase
}

// NewUserSuspiciousHandler は新しいUserSuspiciousHandlerを生成します。
func NewUserSuspiciousHandler(usecase usecase.UserSuspiciousUsecase) *UserSuspiciousHandler {
	return &UserSuspiciousHandler{usecase: usecase}
}

// UpdateSuspicious は指定ユーザーの不審アカウントフラグを変更します。
func (h *UserSuspiciousHandler) UpdateSuspicious(c *echo.Context) error {
	username, apiErr := apphandler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}

	var request updateUserSuspiciousRequest
	if err := apphandler.BindStrictJSON(c, &request); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if request.IsSuspicious == nil {
		return apierror.ErrBadRequest
	}

	requester, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		return apierror.ErrUnauthorized
	}

	if err := h.usecase.ChangeSuspicious(c.Request().Context(), requester, username, *request.IsSuspicious); err != nil {
		switch {
		case errors.Is(err, usecase.ErrAdminRequired):
			return apierror.ErrForbidden.WithInternal(err)
		case errors.Is(err, repository.ErrUserConflict):
			return apierror.ErrConflict.WithInternal(err)
		default:
			return apierror.FromUsecaseError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
