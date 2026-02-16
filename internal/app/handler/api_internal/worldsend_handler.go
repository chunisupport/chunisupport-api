package api_internal

import (
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
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
func (h *WorldsendHandler) GetWorldsendSongs(c echo.Context) error {
	includeDeleted := c.QueryParam("include_deleted") == "true"
	songsWithCharts, err := h.worldsendUsecase.GetAllWorldsendSongs(c.Request().Context(), includeDeleted)
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
