package api_v1

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// V1VersionHandler は外部API v1 のバージョン関連エンドポイントを処理します。
type V1VersionHandler struct {
	masterDataUsecase usecase.MasterDataUsecase
}

// NewV1VersionHandler は新しい V1VersionHandler を生成します。
func NewV1VersionHandler(masterDataUsecase usecase.MasterDataUsecase) *V1VersionHandler {
	return &V1VersionHandler{masterDataUsecase: masterDataUsecase}
}

// GetVersions はバージョン一覧を返却します。
func (h *V1VersionHandler) GetVersions(c *echo.Context) error {
	return c.JSON(http.StatusOK, &dto.VersionSummariesResponse{
		Versions: dto.ToVersionSummaryDTOs(h.masterDataUsecase.GetVersions(c.Request().Context())),
	})
}
