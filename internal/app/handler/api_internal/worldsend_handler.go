package api_internal

import (
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

const worldsendChartKey = "WORLDSEND"

// WorldsendHandler は WORLD'S END 楽曲関連の HTTP リクエストを処理します。
type WorldsendHandler struct {
	worldsendUsecase usecase.WorldsendUsecase
	masterCache      *masterdata.Cache
}

// NewWorldsendHandler は新しい WorldsendHandler を生成します。
func NewWorldsendHandler(worldsendUsecase usecase.WorldsendUsecase, masterCache *masterdata.Cache) *WorldsendHandler {
	return &WorldsendHandler{
		worldsendUsecase: worldsendUsecase,
		masterCache:      masterCache,
	}
}

// GetWorldsendSongs は全 WORLD'S END 楽曲を取得します。
// クエリパラメータ include_deleted=true で削除済み楽曲も含めることができます。
// ただし、EDITOR 権限未満のユーザーの場合、削除済み楽曲は自動的に除外されます。
func (h *WorldsendHandler) GetWorldsendSongs(c echo.Context) error {
	includeDeleted := c.QueryParam("include_deleted") == "true"
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	songsWithCharts, err := h.worldsendUsecase.GetAllWorldsendSongs(c.Request().Context(), includeDeleted, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	songDTOs := h.convertToWorldsendSongDTOs(songsWithCharts)
	return c.JSON(http.StatusOK, &api_internal.WorldsendSongsResponse{
		Songs: songDTOs,
	})
}

// GetWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を取得します。
func (h *WorldsendHandler) GetWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	songWithChart, err := h.worldsendUsecase.GetWorldsendSongByDisplayID(c.Request().Context(), displayID, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	songDTO := h.convertToWorldsendSongDTO(songWithChart)
	return c.JSON(http.StatusOK, songDTO)
}

// DeleteWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
func (h *WorldsendHandler) DeleteWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	if err := h.worldsendUsecase.DeleteWorldsendSong(c.Request().Context(), displayID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// RestoreWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を復活させます。
func (h *WorldsendHandler) RestoreWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	if err := h.worldsendUsecase.RestoreWorldsendSong(c.Request().Context(), displayID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// UpdateWorldsendSongs は WORLD'S END 楽曲および譜面情報を一括更新します。
func (h *WorldsendHandler) UpdateWorldsendSongs(c echo.Context) error {
	var requests []*api_internal.UpdateWorldsendSongRequest
	if err := c.Bind(&requests); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if requests == nil {
		return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests: must be array, not null"))
	}

	for idx, req := range requests {
		if req == nil {
			return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests[%d]: request is null", idx))
		}
		if err := c.Validate(req); err != nil {
			return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("requests[%d]: %w", idx, err))
		}
	}

	if h.masterCache == nil {
		return apierror.ErrInternalError.WithInternal(fmt.Errorf("master cache is not initialized"))
	}

	masters := h.masterCache.SongMasters()
	if masters == nil {
		return apierror.ErrInternalError.WithInternal(fmt.Errorf("master cache is not initialized"))
	}

	songs, charts, err := convertWorldsendRequestsToEntities(requests, masters)
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("%w: %w", usecase.ErrInvalidWorldsendInput, err))
	}

	if err := h.worldsendUsecase.UpdateWorldsendSongs(c.Request().Context(), songs, charts); err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func convertWorldsendRequestsToEntities(requests []*api_internal.UpdateWorldsendSongRequest, masters *domainmasterdata.SongMasters) ([]*entity.Song, []*entity.WorldsendChart, error) {
	songs := make([]*entity.Song, 0, len(requests))
	charts := make([]*entity.WorldsendChart, 0, len(requests))

	for idx, req := range requests {
		if req == nil {
			return nil, nil, fmt.Errorf("requests[%d]: request is null", idx)
		}

		chartReq, hasChartUpdate, err := validateAndGetWorldsendChartRequest(req.Charts)
		if err != nil {
			return nil, nil, fmt.Errorf("requests[%d].charts: %w", idx, err)
		}

		var genreID *int
		if req.Genre != nil {
			genreMaster, ok := masters.Genres[*req.Genre]
			if !ok {
				return nil, nil, fmt.Errorf("invalid genre: %s", *req.Genre)
			}
			genreID = &genreMaster.ID
		}

		updatedSong := entity.NewSong()
		updatedSong.DisplayID = req.DisplayID
		updatedSong.Title = req.Title
		updatedSong.Artist = req.Artist
		updatedSong.GenreID = genreID
		updatedSong.BPM = req.BPM
		updatedSong.ReleasedAt = req.ReleasedAt.TimePtr()
		updatedSong.Jacket = req.Jacket
		updatedSong.IsWorldsend = true

		var updatedChart *entity.WorldsendChart
		if hasChartUpdate {
			var levelStarVO *levelstar.LevelStar
			if chartReq.LevelStar != nil {
				ls, lsErr := levelstar.NewLevelStar(*chartReq.LevelStar)
				if lsErr != nil {
					return nil, nil, fmt.Errorf("requests[%d].charts.%s.level_star: %w", idx, worldsendChartKey, lsErr)
				}
				levelStarVO = &ls
			}

			var notesVO *notes.Notes
			if chartReq.Notes != nil {
				n, nErr := notes.NewNotes(*chartReq.Notes)
				if nErr != nil {
					return nil, nil, fmt.Errorf("requests[%d].charts.%s.notes: %w", idx, worldsendChartKey, nErr)
				}
				notesVO = &n
			}

			updatedChart = &entity.WorldsendChart{
				LevelStar: levelStarVO,
				Attribute: chartReq.Attribute,
				Notes:     notesVO,
			}
		}

		songs = append(songs, updatedSong)
		charts = append(charts, updatedChart)
	}

	return songs, charts, nil
}

func validateAndGetWorldsendChartRequest(charts map[string]*api_internal.UpdateWorldsendChartRequest) (*api_internal.UpdateWorldsendChartRequest, bool, error) {
	if len(charts) == 0 {
		return nil, false, nil
	}

	if len(charts) > 1 {
		keys := slices.Sorted(maps.Keys(charts))
		return nil, false, fmt.Errorf("only one chart key (%s) is allowed: got %v", worldsendChartKey, keys)
	}

	chart, ok := charts[worldsendChartKey]
	if !ok {
		var invalidKey string
		for k := range charts {
			invalidKey = k
		}
		return nil, false, fmt.Errorf("unsupported chart key: %s", invalidKey)
	}

	if chart == nil {
		return nil, false, fmt.Errorf("chart for %s is null", worldsendChartKey)
	}

	return chart, true, nil
}

// convertToWorldsendSongDTOs は WorldsendSongWithChart のスライスを WorldsendSongDTO のスライスに変換します。
func (h *WorldsendHandler) convertToWorldsendSongDTOs(songsWithCharts []*repository.WorldsendSongWithChart) []*api_internal.WorldsendSongDTO {
	songDTOs := make([]*api_internal.WorldsendSongDTO, 0, len(songsWithCharts))
	for _, swc := range songsWithCharts {
		songDTOs = append(songDTOs, h.convertToWorldsendSongDTO(swc))
	}
	return songDTOs
}

// convertToWorldsendSongDTO は WorldsendSongWithChart を WorldsendSongDTO に変換します。
func (h *WorldsendHandler) convertToWorldsendSongDTO(swc *repository.WorldsendSongWithChart) *api_internal.WorldsendSongDTO {
	if swc.Song != nil && swc.Song.GenreID != nil {
		if _, ok := h.masterCache.GenreNamesByID[*swc.Song.GenreID]; !ok {
			slog.Warn("genre name not found for genre_id", "genre_id", *swc.Song.GenreID, "song_display_id", swc.Song.DisplayID)
		}
	}
	return api_internal.ToWorldsendSongDTO(swc.Song, swc.Chart, h.masterCache.GenreNamesByID)
}
