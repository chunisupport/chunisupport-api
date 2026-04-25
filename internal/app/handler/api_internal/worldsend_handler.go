package api_internal

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

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

// GetEditorWorldsendSongs は編集者向けに全 WORLD'S END 楽曲を取得します。
func (h *WorldsendHandler) GetEditorWorldsendSongs(c echo.Context) error {
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	songsWithCharts, err := h.worldsendUsecase.GetAllWorldsendSongs(c.Request().Context(), true, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, &api_internal.EditorWorldsendSongsResponse{
		Songs: h.convertToEditorWorldsendSongDTOs(songsWithCharts),
	})
}

// GetEditorWorldsendSong は編集者向けに指定された DisplayID の WORLD'S END 楽曲を取得します。
func (h *WorldsendHandler) GetEditorWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	songWithChart, err := h.worldsendUsecase.GetWorldsendSongByDisplayID(c.Request().Context(), displayID, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, h.convertToEditorWorldsendSongDTO(songWithChart))
}

// DeleteWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
func (h *WorldsendHandler) DeleteWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	if err := h.worldsendUsecase.DeleteWorldsendSong(c.Request().Context(), displayID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// CreateWorldsendSong は新規 WORLD'S END 楽曲を追加します。
func (h *WorldsendHandler) CreateWorldsendSong(c echo.Context) error {
	var req api_internal.CreateWorldsendSongRequest
	if err := c.Bind(&req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	if req.Chart != nil {
		if err := c.Validate(req.Chart); err != nil {
			return err
		}
	}

	if h.masterCache == nil {
		return apierror.ErrInternalError.WithInternal(fmt.Errorf("master cache is not initialized"))
	}

	masters := h.masterCache.SongMasters()
	if masters == nil {
		return apierror.ErrInternalError.WithInternal(fmt.Errorf("song masters are not initialized"))
	}

	// ジャンル名の検証とID変換
	genreItem, ok := masters.Genres[req.Genre]
	if !ok {
		return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("invalid genre: %s", req.Genre))
	}

	var chartInput *usecase.CreateWorldsendChartInput
	if req.Chart != nil {
		chartInput = &usecase.CreateWorldsendChartInput{
			Attribute:     req.Chart.Attribute,
			LevelStar:     req.Chart.LevelStar,
			Notes:         req.Chart.Notes,
			NotesDesigner: req.Chart.NotesDesigner,
		}
	}

	input := &usecase.CreateWorldsendSongInput{
		OfficialIdx: req.OfficialIdx,
		Title:       req.Title,
		Artist:      req.Artist,
		GenreID:     genreItem.ID,
		BPM:         req.BPM,
		ReleasedAt:  req.ReleasedAt.TimePtr(),
		Jacket:      req.Jacket,
		Chart:       chartInput,
	}

	songWithChart, err := h.worldsendUsecase.CreateWorldsendSong(c.Request().Context(), input)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusCreated, h.convertToEditorWorldsendSongDTO(songWithChart))
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
		return apierror.ErrInternalError.WithInternal(fmt.Errorf("song masters are not initialized in master cache"))
	}

	inputs := convertToUpdateWorldsendSongInputs(requests)

	if err := h.worldsendUsecase.UpdateWorldsendSongs(c.Request().Context(), inputs, masters); err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func convertToUpdateWorldsendSongInputs(requests []*api_internal.UpdateWorldsendSongRequest) []*usecase.UpdateWorldsendSongInput {
	inputs := make([]*usecase.UpdateWorldsendSongInput, 0, len(requests))
	for _, req := range requests {
		input := &usecase.UpdateWorldsendSongInput{
			DisplayID: req.DisplayID,
			Title:     req.Title,
			Artist:    req.Artist,
			Genre:     req.Genre,
			BPM:       req.BPM,
			Jacket:    req.Jacket,
		}

		if req.ReleasedAt != nil {
			input.ReleasedAt = req.ReleasedAt.TimePtr()
		}

		if req.Charts != nil {
			input.Charts = make(map[string]*usecase.UpdateWorldsendChartInput, len(req.Charts))
			for key, chartReq := range req.Charts {
				if chartReq == nil {
					input.Charts[key] = nil
					continue
				}

				input.Charts[key] = &usecase.UpdateWorldsendChartInput{
					Attribute:     chartReq.Attribute,
					LevelStar:     chartReq.LevelStar,
					Notes:         chartReq.Notes,
					NotesDesigner: chartReq.NotesDesigner,
				}
			}
		}

		inputs = append(inputs, input)
	}

	return inputs
}

// convertToWorldsendSongDTOs は WorldsendSongWithChart のスライスを WorldsendSongDTO のスライスに変換します。
func (h *WorldsendHandler) convertToWorldsendSongDTOs(songsWithCharts []*entity.WorldsendSongWithChart) []*api_internal.WorldsendSongDTO {
	songDTOs := make([]*api_internal.WorldsendSongDTO, 0, len(songsWithCharts))
	for _, swc := range songsWithCharts {
		songDTOs = append(songDTOs, h.convertToWorldsendSongDTO(swc))
	}
	return songDTOs
}

// convertToWorldsendSongDTO は WorldsendSongWithChart を WorldsendSongDTO に変換します。
func (h *WorldsendHandler) convertToWorldsendSongDTO(swc *entity.WorldsendSongWithChart) *api_internal.WorldsendSongDTO {
	if swc.Song != nil && swc.Song.GenreID != nil {
		if _, ok := h.masterCache.GenreNamesByID[*swc.Song.GenreID]; !ok {
			slog.Warn("genre name not found for genre_id", "genre_id", *swc.Song.GenreID, "song_display_id", swc.Song.DisplayID)
		}
	}
	return api_internal.ToWorldsendSongDTO(swc.Song, swc.Chart, h.masterCache.GenreNamesByID)
}

// convertToEditorWorldsendSongDTOs は WorldsendSongWithChart のスライスを EditorWorldsendSongDTO のスライスに変換します。
func (h *WorldsendHandler) convertToEditorWorldsendSongDTOs(songsWithCharts []*entity.WorldsendSongWithChart) []*api_internal.EditorWorldsendSongDTO {
	songDTOs := make([]*api_internal.EditorWorldsendSongDTO, 0, len(songsWithCharts))
	for _, swc := range songsWithCharts {
		songDTOs = append(songDTOs, h.convertToEditorWorldsendSongDTO(swc))
	}
	return songDTOs
}

// convertToEditorWorldsendSongDTO は WorldsendSongWithChart を EditorWorldsendSongDTO に変換します。
// EditorWorldsendChartDTO を使用して譜面の updated_at を含めます。
func (h *WorldsendHandler) convertToEditorWorldsendSongDTO(swc *entity.WorldsendSongWithChart) *api_internal.EditorWorldsendSongDTO {
	if swc == nil {
		return nil
	}
	base := h.convertToWorldsendSongDTO(swc)

	charts := map[string]*api_internal.EditorWorldsendChartDTO{
		"WORLDSEND": api_internal.ToEditorWorldsendChartDTO(swc.Chart),
	}

	return &api_internal.EditorWorldsendSongDTO{
		WorldsendSongDTO: base,
		IsDeleted:        swc.Song.IsDeleted,
		UpdatedAt:        swc.Song.UpdatedAt,
		Charts:           charts,
	}
}
