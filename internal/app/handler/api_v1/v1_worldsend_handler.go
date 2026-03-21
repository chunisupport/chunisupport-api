package api_v1

import (
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// V1WorldsendHandler は外部 API v1 用の WORLD'S END 楽曲ハンドラです。
type V1WorldsendHandler struct {
	worldsendUsecase usecase.WorldsendUsecase
	masterCache      *masterdata.Cache
}

// NewV1WorldsendHandler は新しい V1WorldsendHandler を生成します。
func NewV1WorldsendHandler(worldsendUsecase usecase.WorldsendUsecase, masterCache *masterdata.Cache) *V1WorldsendHandler {
	return &V1WorldsendHandler{
		worldsendUsecase: worldsendUsecase,
		masterCache:      masterCache,
	}
}

// GetWorldsendSongs は全 WORLD'S END 楽曲を取得します（公開 API）。
// 削除済み楽曲は含まれません。
func (h *V1WorldsendHandler) GetWorldsendSongs(c echo.Context) error {
	// 外部APIでは削除済み楽曲は含めない、requesterAccountTypeIDはnilを渡す
	songsWithCharts, err := h.worldsendUsecase.GetAllWorldsendSongs(c.Request().Context(), false, nil)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	updatedAt, err := h.worldsendUsecase.GetWorldsendSongsLastUpdatedAt(c.Request().Context(), false, nil)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	songDTOs := h.convertToV1WorldsendSongDTOs(songsWithCharts)
	return c.JSON(http.StatusOK, &api_v1.V1WorldsendSongsResponse{
		Songs:     songDTOs,
		UpdatedAt: updatedAt,
	})
}

// GetWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を取得します（公開 API）。
func (h *V1WorldsendHandler) GetWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	requesterAccountTypeID := handler.GetRequesterAccountTypeID(c)
	songWithChart, err := h.worldsendUsecase.GetWorldsendSongByDisplayID(c.Request().Context(), displayID, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	songDTO := h.convertToV1WorldsendSongDTO(songWithChart)
	return c.JSON(http.StatusOK, songDTO)
}

// convertToV1WorldsendSongDTOs は WorldsendSongWithChart のスライスを V1WorldsendSongDTO のスライスに変換します。
func (h *V1WorldsendHandler) convertToV1WorldsendSongDTOs(songsWithCharts []*repository.WorldsendSongWithChart) []*api_v1.V1WorldsendSongDTO {
	songDTOs := make([]*api_v1.V1WorldsendSongDTO, 0, len(songsWithCharts))
	for _, swc := range songsWithCharts {
		songDTOs = append(songDTOs, h.convertToV1WorldsendSongDTO(swc))
	}
	return songDTOs
}

// convertToV1WorldsendSongDTO は WorldsendSongWithChart を V1WorldsendSongDTO に変換します。
func (h *V1WorldsendHandler) convertToV1WorldsendSongDTO(swc *repository.WorldsendSongWithChart) *api_v1.V1WorldsendSongDTO {
	if swc.Song != nil && swc.Song.GenreID != nil {
		if _, ok := h.masterCache.GenreNamesByID[*swc.Song.GenreID]; !ok {
			slog.Warn("genre name not found for genre_id", "genre_id", *swc.Song.GenreID, "song_display_id", swc.Song.DisplayID)
		}
	}
	return api_v1.ToV1WorldsendSongDTO(swc.Song, swc.Chart, h.masterCache.GenreNamesByID)
}
