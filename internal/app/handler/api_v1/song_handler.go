package api_v1

import (
	"context"
	"net/http"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/app/handler"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_v1"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// V1SongHandler は外部API v1 の楽曲関連エンドポイントを処理します。
type V1SongHandler struct {
	songUsecase usecase.SongUsecase
	masterCache *masterdata.Cache
}

// NewV1SongHandler は新しい V1SongHandler を生成します。
func NewV1SongHandler(songUsecase usecase.SongUsecase, masterCache *masterdata.Cache) *V1SongHandler {
	return &V1SongHandler{
		songUsecase: songUsecase,
		masterCache: masterCache,
	}
}

// GetSongs は全楽曲を取得します（WORLD'S END以外、削除済み除外）。
func (h *V1SongHandler) GetSongs(c echo.Context) error {
	// content=full パラメータで統計データを含むか判定
	// 外部APIでは削除済み楽曲は含めない
	songsWithCharts, err := h.songUsecase.GetAllSongsExcludingWorldsend(c.Request().Context(), false)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	// V1DTOに変換
	v1Songs := h.convertToV1SongDTOs(c.Request().Context(), songsWithCharts, false)

	return c.JSON(http.StatusOK, &api_v1.V1SongsResponse{
		Songs: v1Songs,
	})
}

// GetSong は指定された songId の楽曲を取得します。
func (h *V1SongHandler) GetSong(c echo.Context) error {
	// content=full パラメータで統計データを含むか判定
	includeFull := c.QueryParam("content") == "full"

	songID := c.Param("songId")
	swc, err := h.songUsecase.GetSongByDisplayID(c.Request().Context(), songID)
	if err != nil {
		// usecaseからのエラーをAPIエラーに変換
		return apierror.FromUsecaseError(err)
	}

	// V1DTOに変換
	v1SongDTO := h.convertToV1SongDTO(c.Request().Context(), swc, includeFull)

	return c.JSON(http.StatusOK, v1SongDTO)
}

// convertToV1SongDTOs は SongWithCharts のスライスを V1SongDTO のスライスに変換します。
// includeFullがtrueの場合、統計データを含めます。
func (h *V1SongHandler) convertToV1SongDTOs(ctx context.Context, songsWithCharts []*repository.SongWithCharts, includeFull bool) []*api_v1.V1SongDTO {
	// 統計データが必要な場合は一括取得（N+1問題回避）
	var statsMap map[int][]*entity.ChartStatistics
	if includeFull {
		chartIDs := h.extractChartIDs(songsWithCharts)
		var err error
		statsMap, err = h.songUsecase.GetChartStatisticsByChartIDs(ctx, chartIDs)
		if err != nil {
			// エラーが発生しても統計データなしで続行（統計はオプショナルなデータ）
			statsMap = make(map[int][]*entity.ChartStatistics)
		}
	}

	v1Songs := make([]*api_v1.V1SongDTO, 0, len(songsWithCharts))
	for _, swc := range songsWithCharts {
		v1Songs = append(v1Songs, h.convertToV1SongDTOWithStats(swc, statsMap))
	}
	return v1Songs
}

// convertToV1SongDTO は SongWithCharts を V1SongDTO に変換します。
// includeFullがtrueの場合、統計データを含めます。
func (h *V1SongHandler) convertToV1SongDTO(ctx context.Context, swc *repository.SongWithCharts, includeFull bool) *api_v1.V1SongDTO {
	// 統計データが必要な場合は取得
	var statsMap map[int][]*entity.ChartStatistics
	if includeFull {
		chartIDs := make([]int, len(swc.Charts))
		for i, chart := range swc.Charts {
			chartIDs[i] = chart.ID
		}
		var err error
		statsMap, err = h.songUsecase.GetChartStatisticsByChartIDs(ctx, chartIDs)
		if err != nil {
			// エラーが発生しても統計データなしで続行
			statsMap = make(map[int][]*entity.ChartStatistics)
		}
	}

	return h.convertToV1SongDTOWithStats(swc, statsMap)
}

// convertToV1SongDTOWithStats は SongWithCharts と統計マップを V1SongDTO に変換します。
// Charts フィールドは難易度名をキーとするマップに変換されます。
// マッピングルール: 1->"basic", 2->"advanced", 3->"expert", 4->"master", 5->"ultima"
func (h *V1SongHandler) convertToV1SongDTOWithStats(swc *repository.SongWithCharts, statsMap map[int][]*entity.ChartStatistics) *api_v1.V1SongDTO {
	v1SongDTO := api_v1.ToV1SongDTO(swc.Song, h.masterCache.GenreNamesByID)

	// 難易度IDから名称へのマッピング（マスタデータから取得）
	difficultyNames := h.masterCache.DifficultyNamesByID

	v1SongDTO.Charts = handler.BuildChartsMap(swc.Charts, difficultyNames, func(chart *entity.Chart) *api_v1.V1ChartDTO {
		dto := api_v1.ToV1ChartDTO(chart)
		// 統計データの設定（content=fullの場合のみ）
		if statsMap != nil {
			// 譜面定数10.0以上の場合のみ統計データを設定
			if chart.Const >= 10.0 {
				if statsList, ok := statsMap[chart.ID]; ok && len(statsList) > 0 {
					dto.Statistics = api_v1.ToChartStatisticsDTO(statsList)
				} else {
					// 統計データが存在しない場合は空のマップを設定
					dto.Statistics = api_v1.NewEmptyChartStatisticsDTO()
				}
			}
			// 譜面定数10.0未満の場合はnullのまま（統計対象外）
		}
		return dto
	})
	return v1SongDTO
}

// extractChartIDs はすべての譜面IDを抽出します（統計一括取得用）。
func (h *V1SongHandler) extractChartIDs(songsWithCharts []*repository.SongWithCharts) []int {
	var chartIDs []int
	for _, swc := range songsWithCharts {
		for _, chart := range swc.Charts {
			// 譜面定数10.0以上のみを対象とする
			if chart.Const >= 10.0 {
				chartIDs = append(chartIDs, chart.ID)
			}
		}
	}
	return chartIDs
}
