package chunirec

import (
	"net/http"

	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// ChunirecHandler はchunirec互換APIのハンドラです
type ChunirecHandler struct {
	songUsecase usecase.SongUsecase
	masterCache *masterdata.Cache
}

// NewChunirecHandler はChunirecHandlerの新しいインスタンスを返します
func NewChunirecHandler(songUsecase usecase.SongUsecase, masterCache *masterdata.Cache) *ChunirecHandler {
	return &ChunirecHandler{
		songUsecase: songUsecase,
		masterCache: masterCache,
	}
}

// GetMusicShowAll は全楽曲情報をchunirec互換形式で返します
// GET /compat/chunirec/v2.0/music/showall
func (h *ChunirecHandler) GetMusicShowAll(c echo.Context) error {
	ctx := c.Request().Context()

	// 楽曲を取得 (削除済みを含まない)
	songs, err := h.songUsecase.GetAllSongsExcludingWorldsend(ctx, false)
	if err != nil {
		return err
	}

	// DTOに変換
	masters := h.masterCache.SongMasters()
	response := ToMusicShowAllResponse(songs, masters)

	return c.JSON(http.StatusOK, response)
}
