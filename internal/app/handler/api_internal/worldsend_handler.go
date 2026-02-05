package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// WorldsendHandler は WORLD'S END 楽曲関連の HTTP リクエストを処理します。
type WorldsendHandler struct {
	worldsendUsecase usecase.WorldsendUsecase
}

// NewWorldsendHandler は新しい WorldsendHandler を生成します。
func NewWorldsendHandler(worldsendUsecase usecase.WorldsendUsecase) *WorldsendHandler {
	return &WorldsendHandler{
		worldsendUsecase: worldsendUsecase,
	}
}

// GetWorldsendSongs は全 WORLD'S END 楽曲を取得します。
// クエリパラメータ include_deleted=true で削除済み楽曲も含めることができます。
func (h *WorldsendHandler) GetWorldsendSongs(c echo.Context) error {
	includeDeleted := c.QueryParam("include_deleted") == "true"
	songs, err := h.worldsendUsecase.GetAllWorldsendSongs(c.Request().Context(), includeDeleted)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"songs": songs,
	})
}

// GetWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を取得します。
func (h *WorldsendHandler) GetWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	song, err := h.worldsendUsecase.GetWorldsendSongByDisplayID(c.Request().Context(), displayID)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusOK, song)
}

// DeleteWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
func (h *WorldsendHandler) DeleteWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	if err := h.worldsendUsecase.DeleteWorldsendSong(c.Request().Context(), displayID); err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// RestoreWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を復活させます。
func (h *WorldsendHandler) RestoreWorldsendSong(c echo.Context) error {
	displayID := c.Param("displayid")
	if err := h.worldsendUsecase.RestoreWorldsendSong(c.Request().Context(), displayID); err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}
	return c.NoContent(http.StatusNoContent)
}
