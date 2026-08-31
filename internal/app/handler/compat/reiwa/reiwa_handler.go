package reiwa

import (
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// ReiwaHandler はreiwa互換APIのハンドラです
type ReiwaHandler struct {
	songUsecase       usecase.SongUsecase
	masterDataUsecase usecase.MasterDataUsecase
	masterCache       *masterdata.Cache
}

// NewReiwaHandler はReiwaHandlerの新しいインスタンスを返します
func NewReiwaHandler(songUsecase usecase.SongUsecase, masterDataUsecase usecase.MasterDataUsecase, masterCache *masterdata.Cache) *ReiwaHandler {
	return &ReiwaHandler{
		songUsecase:       songUsecase,
		masterDataUsecase: masterDataUsecase,
		masterCache:       masterCache,
	}
}

// GetChunithmRecordOriginal はWORLD'S END以外の全楽曲の通常譜面情報をフラットな配列で返します
// GET /compat/reiwa/1/chunithm_record/original
func (h *ReiwaHandler) GetChunithmRecordOriginal(c *echo.Context) error {
	ctx := c.Request().Context()

	songs, err := h.songUsecase.GetAllSongsExcludingWorldsend(ctx, false, nil)
	if err != nil {
		slog.Error("failed to get songs for reiwa compat", "error", err)
		return apierror.ErrInternalError.WithInternal(err)
	}

	response := ToChunithmRecordOriginalResponse(songs, h.masterCache)

	return c.JSON(http.StatusOK, response)
}

// GetChunithmVersions は元APIと同じトップレベル配列を維持してCHUNITHMのバージョン一覧を返します。
// GET /compat/reiwa/1/chunithm_versions.json
func (h *ReiwaHandler) GetChunithmVersions(c *echo.Context) error {
	versions := h.masterDataUsecase.GetVersions(c.Request().Context())
	return c.JSON(http.StatusOK, ToChunithmVersionsResponse(versions))
}
