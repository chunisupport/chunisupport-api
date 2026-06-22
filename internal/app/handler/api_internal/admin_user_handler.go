package api_internal

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// AdminUserHandler はADMIN専用のユーザー関連HTTPリクエストを処理します。
type AdminUserHandler struct {
	userUsecase usecase.UserUsecase
}

// NewAdminUserHandler は新しいAdminUserHandlerを生成します。
func NewAdminUserHandler(userUsecase usecase.UserUsecase) *AdminUserHandler {
	return &AdminUserHandler{userUsecase: userUsecase}
}

// GetAllUsers handles GET /internal/users/
// ADMIN専用で、プライベート・削除済み・プレイヤー未紐付けアカウントを含むすべてのユーザーを返します。
func (h *AdminUserHandler) GetAllUsers(c echo.Context) error {
	pageParam := c.QueryParam("page")
	page := 1
	if pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	name := c.QueryParam("name")

	limit := info.DefaultUserListLimit

	users, err := h.userUsecase.GetAllUsersForAdmin(c.Request().Context(), page, limit, name)
	if err != nil {
		// Logged in usecase
		return apierror.ErrInternalError
	}

	return c.JSON(http.StatusOK, users)
}

// UpdateUserAccountType はADMIN専用で、指定ユーザーの権限を変更します。
func (h *AdminUserHandler) UpdateUserAccountType(c echo.Context) error {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil || userID <= 0 {
		return apierror.ErrBadRequest
	}

	var req dto_internal.UpdateUserAccountTypeRequest
	if err := c.Bind(&req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	requester, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		return apierror.ErrUnauthorized
	}

	result, err := h.userUsecase.ChangeUserAccountType(c.Request().Context(), requester, userID, req.AccountType)
	if err != nil {
		if !errors.Is(err, usecase.ErrAdminRequired) && !errors.Is(err, usecase.ErrUserNotFound) && !errors.Is(err, usecase.ErrInvalidAccountType) {
			return apierror.ErrInternalError.WithInternal(err)
		}
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, dto_internal.AdminUserAccountTypeResponse{
		ID:          result.ID,
		UserName:    result.Username.String(),
		AccountType: req.AccountType,
		UpdatedAt:   result.UpdatedAt,
	})
}
