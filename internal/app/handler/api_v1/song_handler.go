package api_v1

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// V1SongHandler は外部API v1 の楽曲関連エンドポイントを処理します。
type V1SongHandler struct {
	songUsecase       usecase.SongUsecase
	statsUsecase      usecase.ChartStatsUsecase
	masterCache       *masterdata.Cache
	staticMasterCache *masterdata.StaticCache
}

// NewV1SongHandler は新しい V1SongHandler を生成します。
func NewV1SongHandler(songUsecase usecase.SongUsecase, statsUsecase usecase.ChartStatsUsecase, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache) *V1SongHandler {
	return &V1SongHandler{
		songUsecase:       songUsecase,
		statsUsecase:      statsUsecase,
		masterCache:       masterCache,
		staticMasterCache: staticMasterCache,
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
	song, err := h.songUsecase.GetSongByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		// usecaseからのエラーをAPIエラーに変換
		return apierror.FromUsecaseError(err)
	}

	// V1DTOに変換
	v1SongDTO := h.convertToV1SongDTO(song)

	return c.JSON(http.StatusOK, v1SongDTO)
}

// GetChartStatsByDifficulty は指定されたDisplayIDと難易度の譜面統計を取得します。
func (h *V1SongHandler) GetChartStatsByDifficulty(c echo.Context) error {
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

// convertToV1SongDTOs は Song のスライスを V1SongDTO のスライスに変換します。
func (h *V1SongHandler) convertToV1SongDTOs(songs []*entity.Song) []*api_v1.V1SongDTO {
	v1Songs := make([]*api_v1.V1SongDTO, 0, len(songs))
	for _, song := range songs {
		v1Songs = append(v1Songs, h.convertToV1SongDTO(song))
	}
	return v1Songs
}

// convertToV1SongDTO は Song を V1SongDTO に変換します。
// Charts フィールドは難易度名をキーとするマップに変換されます。
// マッピングルール: 1->"BASIC", 2->"ADVANCED", 3->"EXPERT", 4->"MASTER", 5->"ULTIMA"
func (h *V1SongHandler) convertToV1SongDTO(song *entity.Song) *api_v1.V1SongDTO {
	maxOP := h.songUsecase.CalcSongMaxOP(song)
	v1SongDTO := api_v1.ToV1SongDTO(song, h.masterCache.GenreNamesByID, maxOP)

	// 難易度IDから名称へのマッピング（マスタデータから取得）
	difficultyNames := h.masterCache.DifficultyNamesByID

	v1SongDTO.Charts = handler.BuildChartsMap(song.Charts, difficultyNames, func(chart *entity.Chart) *api_v1.V1ChartDTO {
		return api_v1.ToV1ChartDTO(chart)
	})
	return v1SongDTO
}
