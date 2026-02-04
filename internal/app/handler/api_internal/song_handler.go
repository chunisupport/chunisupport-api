package api_internal

import (
	"net/http"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/app/handler"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/dto"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// SongHandler は曲関連のHTTPリクエストを処理します。
type SongHandler struct {
	songUsecase  usecase.SongUsecase
	statsUsecase usecase.ChartStatsUsecase
	masterCache  *masterdata.Cache
}

// NewSongHandler は新しいSongHandlerを生成します。
func NewSongHandler(songUsecase usecase.SongUsecase, statsUsecase usecase.ChartStatsUsecase, masterCache *masterdata.Cache) *SongHandler {
	return &SongHandler{
		songUsecase:  songUsecase,
		statsUsecase: statsUsecase,
		masterCache:  masterCache,
	}
}

// GetSongs はWORLD'S END以外の全楽曲を取得します。
// クエリパラメータ include_deleted=true で削除済み楽曲も含めることができます。
func (h *SongHandler) GetSongs(c echo.Context) error {
	includeDeleted := c.QueryParam("include_deleted") == "true"

	songsWithCharts, err := h.songUsecase.GetAllSongsExcludingWorldsend(c.Request().Context(), includeDeleted)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	// DTOに変換
	songDTOs := h.convertToSongDTOs(songsWithCharts)

	result := &api_internal.SongsResponse{
		Songs: songDTOs,
	}

	return c.JSON(http.StatusOK, result)
}

// GetSong は指定されたDisplayIDの楽曲を取得します。
func (h *SongHandler) GetSong(c echo.Context) error {
	displayID := c.Param("displayid")
	swc, err := h.songUsecase.GetSongByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	// DTOに変換
	songDTO := h.convertToSongDTO(swc)

	return c.JSON(http.StatusOK, songDTO)
}

// GetSongStats は指定されたDisplayIDの譜面統計を取得します。
func (h *SongHandler) GetSongStats(c echo.Context) error {
	displayID := c.Param("displayid")
	stats, err := h.statsUsecase.GetSongStatsByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, dto.ToChartStatsResponse(stats))
}

// DeleteSong は指定されたDisplayIDの楽曲を論理削除します。
func (h *SongHandler) DeleteSong(c echo.Context) error {
	displayID := c.Param("displayid")
	if err := h.songUsecase.DeleteSong(c.Request().Context(), displayID); err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// RestoreSong は指定されたDisplayIDの楽曲を復活させます。
func (h *SongHandler) RestoreSong(c echo.Context) error {
	displayID := c.Param("displayid")
	if err := h.songUsecase.RestoreSong(c.Request().Context(), displayID); err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// UpdateSongs は楽曲および譜面情報を一括更新します。
func (h *SongHandler) UpdateSongs(c echo.Context) error {
	var requests []*api_internal.UpdateSongRequest
	if err := c.Bind(&requests); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	// バリデーション
	if err := c.Validate(requests); err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}

	// サービス層での更新処理
	if err := h.songUsecase.UpdateSongs(c.Request().Context(), requests); err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// convertToSongDTOs は SongWithCharts のスライスを SongDTO のスライスに変換します。
func (h *SongHandler) convertToSongDTOs(songsWithCharts []*repository.SongWithCharts) []*api_internal.SongDTO {
	songDTOs := make([]*api_internal.SongDTO, 0, len(songsWithCharts))
	for _, swc := range songsWithCharts {
		songDTOs = append(songDTOs, h.convertToSongDTO(swc))
	}
	return songDTOs
}

// convertToSongDTO は SongWithCharts を SongDTO に変換します。
// Charts フィールドは難易度名をキーとするマップに変換されます。
// マッピングルール: 1->"BASIC", 2->"ADVANCED", 3->"EXPERT", 4->"MASTER", 5->"ULTIMA"
func (h *SongHandler) convertToSongDTO(swc *repository.SongWithCharts) *api_internal.SongDTO {
	songDTO := api_internal.ToSongDTO(swc.Song, h.masterCache.GenreNamesByID)

	// 難易度IDから名称へのマッピング（マスタデータから取得）
	difficultyNames := h.masterCache.DifficultyNamesByID

	songDTO.Charts = handler.BuildChartsMap(swc.Charts, difficultyNames, func(chart *entity.Chart) *api_internal.ChartDTO {
		return api_internal.ToChartDTO(chart)
	})
	return songDTO
}
