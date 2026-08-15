package api_internal

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
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
func (h *AdminUserHandler) GetAllUsers(c *echo.Context) error {
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

	result := make([]internaldto.AdminUserListResponse, 0, len(users))
	for _, user := range users {
		result = append(result, internaldto.AdminUserListResponse{UserName: user.UserName, AccountType: user.AccountType, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, PlayerName: user.PlayerName, Rating: user.Rating, OverPowerValue: user.OverPowerValue, IsSuspicious: user.IsSuspicious, IsPrivate: user.IsPrivate})
	}
	return c.JSON(http.StatusOK, result)
}
