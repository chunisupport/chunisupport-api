package api_internal

import (
	"errors"
	"fmt"
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

// GetAllUsers は GET /internal/users/ のユーザー一覧を返します。
// ADMIN専用で、プライベート・削除済み・プレイヤー未紐付けアカウントを含むすべてのユーザーを返します。
func (h *AdminUserHandler) GetAllUsers(c *echo.Context) error {
	pageParam := c.QueryParam("page")
	page := 1
	if pageParam != "" {
		p, err := strconv.Atoi(pageParam)
		if err != nil {
			// 正の桁あふれだけ422にし、非数値・空白・負の桁あふれは従来どおり1に補正する。
			// Atoiは桁あふれ時に最大・最小値へ飽和するため、p>0で正の巨大値だけを区別する。
			if p > 0 && errors.Is(err, strconv.ErrRange) {
				return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("pageがintに収まりません: %w", err))
			}
		} else if p > 0 {
			page = p
		}
	}

	name := c.QueryParam("name")

	limit := info.DefaultUserListLimit

	users, err := h.userUsecase.GetAllUsersForAdmin(c.Request().Context(), page, limit, name)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	result := make([]internaldto.AdminUserListResponse, 0, len(users))
	for _, user := range users {
		result = append(result, internaldto.AdminUserListResponse{UserName: user.UserName, AccountType: user.AccountType, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, PlayerName: user.PlayerName, Rating: user.Rating, OverPowerValue: user.OverPowerValue, IsSuspicious: user.IsSuspicious, IsPrivate: user.IsPrivate})
	}
	return c.JSON(http.StatusOK, result)
}
