package api_internal

import (
	"context"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type adminUserStatisticsUsecase interface {
	Get(ctx context.Context) (usecase.AdminUserStatisticsOutput, error)
}

// AdminUserStatisticsHandler はADMIN専用のユーザー集計HTTPリクエストを処理します。
type AdminUserStatisticsHandler struct {
	usecase adminUserStatisticsUsecase
}

// NewAdminUserStatisticsHandler はAdminUserStatisticsHandlerを生成します。
func NewAdminUserStatisticsHandler(usecase adminUserStatisticsUsecase) *AdminUserStatisticsHandler {
	return &AdminUserStatisticsHandler{usecase: usecase}
}

// Get は管理者画面向けのユーザー集計値を返します。
func (h *AdminUserStatisticsHandler) Get(c *echo.Context) error {
	statistics, err := h.usecase.Get(c.Request().Context())
	if err != nil {
		return apierror.ErrInternalError
	}

	return c.JSON(http.StatusOK, internaldto.AdminUserStatisticsResponse{
		TotalUsers:                 statistics.TotalUsers,
		UsersWithPlayerData:        statistics.UsersWithPlayerData,
		ActivePlayerDataLast30Days: statistics.ActivePlayerDataLast30Days,
	})
}
