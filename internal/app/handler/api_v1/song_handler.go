package api_v1

import (
	"net/http"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/app/handler"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/dto"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_v1"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// V1SongHandler は外部API v1 の楽曲関連エンドポイントを処理します。
type V1SongHandler struct {
	songUsecase  usecase.SongUsecase
	statsUsecase usecase.ChartStatsUsecase
	masterCache  *masterdata.Cache
}

// NewV1SongHandler は新しい V1SongHandler を生成します。
func NewV1SongHandler(songUsecase usecase.SongUsecase, statsUsecase usecase.ChartStatsUsecase, masterCache *masterdata.Cache) *V1SongHandler {
	return &V1SongHandler{
		songUsecase:  songUsecase,
		statsUsecase: statsUsecase,
		masterCache:  masterCache,
	}
}

// GetSongs は全楽曲を取得します（WORLD'S END以外、削除済み除外）。
func (h *V1SongHandler) GetSongs(c echo.Context) error {
	// 外部APIでは削除済み楽曲は含めない
	songsWithCharts, err := h.songUsecase.GetAllSongsExcludingWorldsend(c.Request().Context(), false)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	// V1DTOに変換
	v1Songs := h.convertToV1SongDTOs(songsWithCharts)

	return c.JSON(http.StatusOK, &api_v1.V1SongsResponse{
		Songs: v1Songs,
	})
}

// GetSong は指定された displayid の楽曲を取得します。
func (h *V1SongHandler) GetSong(c echo.Context) error {
	displayID := c.Param("displayid")
	swc, err := h.songUsecase.GetSongByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		// usecaseからのエラーをAPIエラーに変換
		return apierror.FromUsecaseError(err)
	}

	// V1DTOに変換
	v1SongDTO := h.convertToV1SongDTO(swc)

	return c.JSON(http.StatusOK, v1SongDTO)
}

// GetSongStats は指定されたDisplayIDの譜面統計を取得します。
func (h *V1SongHandler) GetSongStats(c echo.Context) error {
	displayID := c.Param("displayid")
	stats, err := h.statsUsecase.GetSongStatsByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, dto.ToChartStatsResponse(stats))
}

// convertToV1SongDTOs は SongWithCharts のスライスを V1SongDTO のスライスに変換します。
func (h *V1SongHandler) convertToV1SongDTOs(songsWithCharts []*repository.SongWithCharts) []*api_v1.V1SongDTO {
	v1Songs := make([]*api_v1.V1SongDTO, 0, len(songsWithCharts))
	for _, swc := range songsWithCharts {
		v1Songs = append(v1Songs, h.convertToV1SongDTO(swc))
	}
	return v1Songs
}

// convertToV1SongDTO は SongWithCharts を V1SongDTO に変換します。
// Charts フィールドは難易度名をキーとするマップに変換されます。
// マッピングルール: 1->"basic", 2->"advanced", 3->"expert", 4->"master", 5->"ultima"
func (h *V1SongHandler) convertToV1SongDTO(swc *repository.SongWithCharts) *api_v1.V1SongDTO {
	v1SongDTO := api_v1.ToV1SongDTO(swc.Song, h.masterCache.GenreNamesByID)

	// 難易度IDから名称へのマッピング（マスタデータから取得）
	difficultyNames := h.masterCache.DifficultyNamesByID

	v1SongDTO.Charts = handler.BuildChartsMap(swc.Charts, difficultyNames, func(chart *entity.Chart) *api_v1.V1ChartDTO {
		return api_v1.ToV1ChartDTO(chart)
	})
	return v1SongDTO
}
