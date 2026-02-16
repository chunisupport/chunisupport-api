package api_internal

import (
	"fmt"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// SongHandler は曲関連のHTTPリクエストを処理します。
type SongHandler struct {
	songUsecase       usecase.SongUsecase
	statsUsecase      usecase.ChartStatsUsecase
	masterCache       *masterdata.Cache
	staticMasterCache *masterdata.StaticCache
}

// NewSongHandler は新しいSongHandlerを生成します。
func NewSongHandler(songUsecase usecase.SongUsecase, statsUsecase usecase.ChartStatsUsecase, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache) *SongHandler {
	return &SongHandler{
		songUsecase:       songUsecase,
		statsUsecase:      statsUsecase,
		masterCache:       masterCache,
		staticMasterCache: staticMasterCache,
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
	song, err := h.songUsecase.GetSongByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	// DTOに変換
	songDTO := h.convertToSongDTO(song)

	return c.JSON(http.StatusOK, songDTO)
}

// GetChartStatsByDifficulty は指定されたDisplayIDと難易度の譜面統計を取得します。
func (h *SongHandler) GetChartStatsByDifficulty(c echo.Context) error {
	displayID := c.Param("displayid")
	difficultyPath := c.Param("difficulty")

	// パスパラメータを内部難易度名に変換
	difficultyName, ok := handler.ParseDifficultyPath(difficultyPath)
	if !ok {
		return apierror.ErrInvalidDifficulty
	}

	stats, err := h.statsUsecase.GetChartStatsByDisplayIDAndDifficulty(c.Request().Context(), displayID, difficultyName)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	// rating_bandsはキャッシュから取得
	ratingBands := h.staticMasterCache.RatingBands

	return c.JSON(http.StatusOK, dto.ToSingleChartStatsResponse(stats, ratingBands))
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

	if requests == nil {
		return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests must be array, not null"))
	}

	// バリデーション
	for idx, req := range requests {
		if req == nil {
			return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests[%d]: request is null", idx))
		}

		for cIdx, chart := range req.Charts {
			if chart == nil {
				return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests[%d].charts[%d]: chart is null", idx, cIdx))
			}
		}

		if err := c.Validate(req); err != nil {
			return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests[%d]: %w", idx, err))
		}
	}

	// サービス層での更新処理
	if err := h.songUsecase.UpdateSongs(c.Request().Context(), requests); err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// convertToSongDTOs は Song のスライスを SongDTO のスライスに変換します。
func (h *SongHandler) convertToSongDTOs(songs []*entity.Song) []*api_internal.SongDTO {
	songDTOs := make([]*api_internal.SongDTO, 0, len(songs))
	for _, song := range songs {
		songDTOs = append(songDTOs, h.convertToSongDTO(song))
	}
	return songDTOs
}

// convertToSongDTO は Song を SongDTO に変換します。
// Charts フィールドは難易度名をキーとするマップに変換されます。
// マッピングルール: 1->"BASIC", 2->"ADVANCED", 3->"EXPERT", 4->"MASTER", 5->"ULTIMA"
func (h *SongHandler) convertToSongDTO(song *entity.Song) *api_internal.SongDTO {
	maxOP := h.songUsecase.CalcSongMaxOP(song)
	songDTO := api_internal.ToSongDTO(song, h.masterCache.GenreNamesByID, maxOP)

	// 難易度IDから名称へのマッピング（マスタデータから取得）
	difficultyNames := h.masterCache.DifficultyNamesByID

	songDTO.Charts = handler.BuildChartsMap(song.Charts, difficultyNames, func(chart *entity.Chart) *api_internal.ChartDTO {
		return api_internal.ToChartDTO(chart)
	})
	return songDTO
}
