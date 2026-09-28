package api_internal

import (
	"context"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/labstack/echo/v5"
)

type chartStatsBatchJobUsecase interface {
	StartFromAdmin(ctx context.Context, requester entity.ChartStatsBatchJobRequester) (*entity.ChartStatsBatchJob, error)
	List(ctx context.Context) ([]*entity.ChartStatsBatchJob, error)
	Get(ctx context.Context, id string) (*entity.ChartStatsBatchJob, error)
}

// ChartStatsBatchHandler は管理画面からの譜面統計バッチ実行と実行履歴の参照を処理します。
type ChartStatsBatchHandler struct {
	usecase chartStatsBatchJobUsecase
}

// NewChartStatsBatchHandler は ChartStatsBatchHandler を生成します。
func NewChartStatsBatchHandler(jobUsecase chartStatsBatchJobUsecase) *ChartStatsBatchHandler {
	return &ChartStatsBatchHandler{usecase: jobUsecase}
}

// Start は譜面統計バッチの実行を受け付けます。
// 全記録の集計には時間がかかるためバックグラウンドで実行し、開始したジョブを 202 で返します。
// 実行条件はないため、リクエストボディは参照しません。
func (h *ChartStatsBatchHandler) Start(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	job, err := h.usecase.StartFromAdmin(
		c.Request().Context(),
		entity.ChartStatsBatchJobRequester{UserID: user.ID, Username: user.Username},
	)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusAccepted, internaldto.ToChartStatsBatchJobDTO(job))
}

// List は直近の実行履歴を返します。
func (h *ChartStatsBatchHandler) List(c *echo.Context) error {
	jobs, err := h.usecase.List(c.Request().Context())
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusOK, internaldto.ToChartStatsBatchJobListDTO(jobs))
}

// Get は指定したジョブを返します。
func (h *ChartStatsBatchHandler) Get(c *echo.Context) error {
	job, err := h.usecase.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusOK, internaldto.ToChartStatsBatchJobDTO(job))
}
