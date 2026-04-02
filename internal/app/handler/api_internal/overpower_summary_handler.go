package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// OverpowerSummaryHandler は本人向け OVER POWER 集計APIを扱います。
type OverpowerSummaryHandler struct {
	overpowerSummaryUsecase usecase.OverpowerSummaryUsecase
}

// NewOverpowerSummaryHandler は OverpowerSummaryHandler を生成します。
func NewOverpowerSummaryHandler(overpowerSummaryUsecase usecase.OverpowerSummaryUsecase) *OverpowerSummaryHandler {
	return &OverpowerSummaryHandler{overpowerSummaryUsecase: overpowerSummaryUsecase}
}

// Get は認証済みユーザーの OVER POWER 集計を返します。
func (h *OverpowerSummaryHandler) Get(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	resp, err := h.overpowerSummaryUsecase.Get(c.Request().Context(), user)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, resp)
}
