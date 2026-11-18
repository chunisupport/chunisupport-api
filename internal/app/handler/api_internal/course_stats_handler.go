package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// CourseStatsHandler はinternal APIのコース統計取得を処理します。
type CourseStatsHandler struct {
	usecase        usecase.CourseStatsUsecase
	masterProvider repository.ChartStatsMasterProvider
}

// NewCourseStatsHandler はコース統計Handlerを生成します。
func NewCourseStatsHandler(courseStatsUsecase usecase.CourseStatsUsecase, masterProvider repository.ChartStatsMasterProvider) *CourseStatsHandler {
	return &CourseStatsHandler{usecase: courseStatsUsecase, masterProvider: masterProvider}
}

// Get は指定コースのレーティング帯別統計を返します。
func (h *CourseStatsHandler) Get(c *echo.Context) error {
	displayID, apiErr := handler.ValidateDisplayID(c.Param("id"))
	if apiErr != nil {
		return apiErr
	}
	stats, err := h.usecase.GetByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, dto.ToCourseStatsResponse(c.Request().Context(), stats, h.masterProvider.RatingBands()))
}
