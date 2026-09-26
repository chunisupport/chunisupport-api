package api_internal

import (
	"context"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/labstack/echo/v5"
)

type songBatchJobUsecase interface {
	StartFromAdmin(ctx context.Context, requester entity.SongBatchJobRequester, req songbatch.RunRequest) (*entity.SongBatchJob, error)
	List(ctx context.Context) ([]*entity.SongBatchJob, error)
	Get(ctx context.Context, id string) (*entity.SongBatchJob, error)
}

// SongBatchHandler は管理画面からの楽曲バッチ実行と実行履歴の参照を処理します。
type SongBatchHandler struct {
	usecase songBatchJobUsecase
}

// NewSongBatchHandler は SongBatchHandler を生成します。
func NewSongBatchHandler(jobUsecase songBatchJobUsecase) *SongBatchHandler {
	return &SongBatchHandler{usecase: jobUsecase}
}

// Start は楽曲バッチの実行を受け付けます。
// 処理は数分かかるためバックグラウンドで実行し、開始したジョブを 202 で返します。
func (h *SongBatchHandler) Start(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	var request internaldto.StartSongBatchJobRequest
	if err := apphandler.BindStrictJSON(c, &request); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	mode, err := songbatch.ParseRunMode(request.Mode)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	job, err := h.usecase.StartFromAdmin(
		c.Request().Context(),
		entity.SongBatchJobRequester{UserID: user.ID, Username: user.Username},
		songbatch.RunRequest{Mode: mode, FillMissingReleaseDate: request.FillMissingReleaseDate},
	)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusAccepted, internaldto.ToSongBatchJobDTO(job))
}

// List は直近の実行履歴を返します。
func (h *SongBatchHandler) List(c *echo.Context) error {
	jobs, err := h.usecase.List(c.Request().Context())
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusOK, internaldto.ToSongBatchJobListDTO(jobs))
}

// Get は指定したジョブを返します。
func (h *SongBatchHandler) Get(c *echo.Context) error {
	job, err := h.usecase.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusOK, internaldto.ToSongBatchJobDTO(job))
}
