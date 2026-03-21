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
// ただし、EDITOR 権限未満のユーザーの場合、削除済み楽曲は自動的に除外されます。
func (h *SongHandler) GetSongs(c echo.Context) error {
	includeDeleted := c.QueryParam("include_deleted") == "true"
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)

	songsWithCharts, err := h.songUsecase.GetAllSongsExcludingWorldsend(c.Request().Context(), includeDeleted, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	updatedAt, err := h.songUsecase.GetSongsLastUpdatedAt(c.Request().Context(), includeDeleted, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	// DTOに変換
	songDTOs := h.convertToSongDTOs(songsWithCharts)

	result := &api_internal.SongsResponse{
		Songs:     songDTOs,
		UpdatedAt: updatedAt,
	}

	return c.JSON(http.StatusOK, result)
}

// GetSong は指定されたDisplayIDの楽曲を取得します。
func (h *SongHandler) GetSong(c echo.Context) error {
	displayID := c.Param("displayid")
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	song, err := h.songUsecase.GetSongByDisplayID(c.Request().Context(), displayID, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	// DTOに変換
	songDTO := h.convertToSongDTO(song)

	return c.JSON(http.StatusOK, songDTO)
}

// GetEditorSongs は編集者向けにWORLD'S END以外の全楽曲を取得します。
func (h *SongHandler) GetEditorSongs(c echo.Context) error {
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	songsWithCharts, err := h.songUsecase.GetAllSongsExcludingWorldsend(c.Request().Context(), true, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	updatedAt, err := h.songUsecase.GetSongsLastUpdatedAt(c.Request().Context(), true, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, &api_internal.EditorSongsResponse{
		Songs:     h.convertToEditorSongDTOs(songsWithCharts),
		UpdatedAt: updatedAt,
	})
}

// GetEditorSong は編集者向けに指定されたDisplayIDの楽曲を取得します。
func (h *SongHandler) GetEditorSong(c echo.Context) error {
	displayID := c.Param("displayid")
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	song, err := h.songUsecase.GetSongByDisplayID(c.Request().Context(), displayID, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, h.convertToEditorSongDTO(song))
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

	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	stats, err := h.statsUsecase.GetChartStatsByDisplayIDAndDifficulty(c.Request().Context(), displayID, difficultyName, requesterAccountTypeID)
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
		return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests: must be array, not null"))
	}

	// バリデーション
	for idx, req := range requests {
		if req == nil {
			return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests[%d]: request is null", idx))
		}
		for diff, chart := range req.Charts {
			if chart == nil {
				return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests[%d].charts[%s]: chart is null", idx, diff))
			}
		}
		if err := c.Validate(req); err != nil {
			return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests[%d]: %w", idx, err))
		}
	}

	// ユースケース層での更新処理
	if err := h.songUsecase.UpdateSongs(c.Request().Context(), requests); err != nil {
		return apierror.FromUsecaseError(err)
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

// convertToEditorSongDTOs は Song のスライスを EditorSongDTO のスライスに変換します。
func (h *SongHandler) convertToEditorSongDTOs(songs []*entity.Song) []*api_internal.EditorSongDTO {
	songDTOs := make([]*api_internal.EditorSongDTO, 0, len(songs))
	for _, song := range songs {
		songDTOs = append(songDTOs, h.convertToEditorSongDTO(song))
	}
	return songDTOs
}

// convertToEditorSongDTO は Song を EditorSongDTO に変換します。
func (h *SongHandler) convertToEditorSongDTO(song *entity.Song) *api_internal.EditorSongDTO {
	if song == nil {
		return nil
	}
	base := h.convertToSongDTO(song)

	return &api_internal.EditorSongDTO{
		SongDTO:   base,
		IsDeleted: song.IsDeleted,
	}
}
