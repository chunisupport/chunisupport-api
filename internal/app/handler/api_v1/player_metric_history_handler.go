package api_v1

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto "github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// PlayerMetricHistoryHandler は外部API v1の公式RATING・公式OVER POWER履歴を処理します。
type PlayerMetricHistoryHandler struct {
	usecase usecase.PlayerMetricHistoryUsecase
}

// NewPlayerMetricHistoryHandler は公式指標履歴Handlerを生成します。
func NewPlayerMetricHistoryHandler(metricHistoryUsecase usecase.PlayerMetricHistoryUsecase) *PlayerMetricHistoryHandler {
	return &PlayerMetricHistoryHandler{usecase: metricHistoryUsecase}
}

// Get は指定ユーザーの公式指標履歴を返します。
func (h *PlayerMetricHistoryHandler) Get(c *echo.Context) error {
	username, apiErr := handler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	requester, _ := c.Get("userEntity").(*entity.User)
	entries, err := h.usecase.Get(c.Request().Context(), username, requester)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	result := make([]dto.PlayerMetricHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, dto.PlayerMetricHistoryEntry{
			Rating: entry.OfficialRating, Overpower: entry.OfficialOverpower,
			DataCollectedAt: entry.DataCollectedAt,
		})
	}
	return c.JSON(http.StatusOK, &dto.PlayerMetricHistoryResponse{Entries: result})
}
