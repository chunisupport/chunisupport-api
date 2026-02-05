package api_v1

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// V1WorldsendHandler は外部 API v1 用の WORLD'S END 楽曲ハンドラです。
type V1WorldsendHandler struct {
	worldsendUsecase usecase.WorldsendUsecase
}

// NewV1WorldsendHandler は新しい V1WorldsendHandler を生成します。
func NewV1WorldsendHandler(worldsendUsecase usecase.WorldsendUsecase) *V1WorldsendHandler {
	return &V1WorldsendHandler{
		worldsendUsecase: worldsendUsecase,
	}
}

// GetWorldsendSongs は全 WORLD'S END 楽曲を取得します（公開 API）。
// 削除済み楽曲は含まれません。
func (h *V1WorldsendHandler) GetWorldsendSongs(c echo.Context) error {
	songs, err := h.worldsendUsecase.GetAllWorldsendSongs(c.Request().Context(), false)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"songs": songs,
	})
}

// GetWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を取得します（公開 API）。
func (h *V1WorldsendHandler) GetWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	song, err := h.worldsendUsecase.GetWorldsendSongByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusOK, song)
}
