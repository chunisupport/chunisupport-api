package api_internal

import (
	"context"
	"net/http"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/app/handler"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// SongHandler は曲関連のHTTPリクエストを処理します。
type SongHandler struct {
	songUsecase usecase.SongUsecase
	masterCache *masterdata.Cache
}

// NewSongHandler は新しいSongHandlerを生成します。
func NewSongHandler(songUsecase usecase.SongUsecase, masterCache *masterdata.Cache) *SongHandler {
	return &SongHandler{
		songUsecase: songUsecase,
		masterCache: masterCache,
	}
}

// GetSongs はWORLD'S END以外の全楽曲を取得します。
// クエリパラメータ include_deleted=true で削除済み楽曲も含めることができます。
// クエリパラメータ content=full で統計データを含めることができます。
func (h *SongHandler) GetSongs(c echo.Context) error {
	includeDeleted := c.QueryParam("include_deleted") == "true"

	songsWithCharts, err := h.songUsecase.GetAllSongsExcludingWorldsend(c.Request().Context(), includeDeleted)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	// DTOに変換
	songDTOs := h.convertToSongDTOs(c.Request().Context(), songsWithCharts, false)

	result := &api_internal.SongsResponse{
		Songs: songDTOs,
	}

	return c.JSON(http.StatusOK, result)
}

// GetSong は指定されたDisplayIDの楽曲を取得します。
// クエリパラメータ content=full で統計データを含めることができます。
func (h *SongHandler) GetSong(c echo.Context) error {
	includeFull := c.QueryParam("content") == "full"

	displayID := c.Param("displayid")
	swc, err := h.songUsecase.GetSongByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	// DTOに変換
	songDTO := h.convertToSongDTO(c.Request().Context(), swc, includeFull)

	return c.JSON(http.StatusOK, songDTO)
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
// includeFullがtrueの場合、統計データを含めます。
func (h *SongHandler) convertToSongDTOs(ctx context.Context, songsWithCharts []*repository.SongWithCharts, includeFull bool) []*api_internal.SongDTO {
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

	songDTOs := make([]*api_internal.SongDTO, 0, len(songsWithCharts))
	for _, swc := range songsWithCharts {
		songDTOs = append(songDTOs, h.convertToSongDTOWithStats(swc, statsMap))
	}
	return songDTOs
}

// convertToSongDTO は SongWithCharts を SongDTO に変換します。
// includeFullがtrueの場合、統計データを含めます。
func (h *SongHandler) convertToSongDTO(ctx context.Context, swc *repository.SongWithCharts, includeFull bool) *api_internal.SongDTO {
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

	return h.convertToSongDTOWithStats(swc, statsMap)
}

// convertToSongDTOWithStats は SongWithCharts と統計マップを SongDTO に変換します。
// Charts フィールドは難易度名をキーとするマップに変換されます。
// マッピングルール: 1->"BASIC", 2->"ADVANCED", 3->"EXPERT", 4->"MASTER", 5->"ULTIMA"
func (h *SongHandler) convertToSongDTOWithStats(swc *repository.SongWithCharts, statsMap map[int][]*entity.ChartStatistics) *api_internal.SongDTO {
	songDTO := api_internal.ToSongDTO(swc.Song, h.masterCache.GenreNamesByID)

	// 難易度IDから名称へのマッピング（マスタデータから取得）
	difficultyNames := h.masterCache.DifficultyNamesByID

	songDTO.Charts = handler.BuildChartsMap(swc.Charts, difficultyNames, func(chart *entity.Chart) *api_internal.ChartDTO {
		dto := api_internal.ToChartDTO(chart)
		// 統計データの設定（content=fullの場合のみ）
		if statsMap != nil {
			// 譜面定数10.0以上の場合のみ統計データを設定
			if chart.Const >= 10.0 {
				if statsList, ok := statsMap[chart.ID]; ok && len(statsList) > 0 {
					dto.Statistics = api_internal.ToChartStatisticsDTO(statsList)
				} else {
					// 統計データが存在しない場合は空のマップを設定
					dto.Statistics = api_internal.NewEmptyChartStatisticsDTO()
				}
			}
			// 譜面定数10.0未満の場合はnullのまま（統計対象外）
		}
		return dto
	})
	return songDTO
}

// extractChartIDs はすべての譜面IDを抽出します（統計一括取得用）。
func (h *SongHandler) extractChartIDs(songsWithCharts []*repository.SongWithCharts) []int {
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
