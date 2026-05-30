package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// itemsToDTOs は []masterdata.Item を []*dto.MasterItemDTO に変換します。
func itemsToDTOs(items []masterdata.Item) []*dto.MasterItemDTO {
	dtos := make([]*dto.MasterItemDTO, len(items))
	for i, item := range items {
		dtos[i] = &dto.MasterItemDTO{ID: item.ID, Name: item.Name}
	}
	return dtos
}

// MasterDataHandler はマスタデータ関連のハンドラです。
type MasterDataHandler struct {
	masterDataUsecase usecase.MasterDataUsecase
}

// NewMasterDataHandler は新しい MasterDataHandler を生成します。
func NewMasterDataHandler(masterDataUsecase usecase.MasterDataUsecase) *MasterDataHandler {
	return &MasterDataHandler{masterDataUsecase: masterDataUsecase}
}

// GetMasterData はフロントエンド向けにマスタデータを返却します。
func (h *MasterDataHandler) GetMasterData(c echo.Context) error {
	out := h.masterDataUsecase.GetMasterData(c.Request().Context())

	genres := itemsToDTOs(out.Genres)
	difficulties := itemsToDTOs(out.Difficulties)
	accountTypes := itemsToDTOs(out.AccountTypes)

	ratingBands := make([]*dto.RatingBandDTO, len(out.RatingBands))
	for i, band := range out.RatingBands {
		ratingBands[i] = &dto.RatingBandDTO{
			ID:           band.ID,
			Label:        band.Label,
			MinInclusive: band.MinInclusive,
			MaxExclusive: band.MaxExclusive,
			SortOrder:    band.SortOrder,
		}
	}

	achievementTypes := itemsToDTOs(out.AchievementTypes)

	return c.JSON(http.StatusOK, &dto.MasterDataResponse{
		Genres:           genres,
		Difficulties:     difficulties,
		AccountTypes:     accountTypes,
		Versions:         dto.ToVersionDTOs(out.Versions),
		RatingBands:      ratingBands,
		AchievementTypes: achievementTypes,
		ClassEmblems:     itemsToDTOs(out.ClassEmblems),
		ClassEmblemBases: itemsToDTOs(out.ClassEmblemBases),
		ClearLamps:       itemsToDTOs(out.ClearLamps),
		ComboLamps:       itemsToDTOs(out.ComboLamps),
		FullChains:       itemsToDTOs(out.FullChains),
		Slots:            itemsToDTOs(out.Slots),
		HonorTypes:       itemsToDTOs(out.HonorTypes),
	})
}

// GetVersions はバージョン一覧を返却します。
func (h *MasterDataHandler) GetVersions(c echo.Context) error {
	return c.JSON(http.StatusOK, &dto.VersionSummariesResponse{
		Versions: dto.ToVersionSummaryDTOs(h.masterDataUsecase.GetVersions(c.Request().Context())),
	})
}

// GetHonorTypes は称号タイプ一覧を返却します。
func (h *MasterDataHandler) GetHonorTypes(c echo.Context) error {
	return c.JSON(http.StatusOK, &dto.HonorTypesResponse{
		HonorTypes: itemsToDTOs(h.masterDataUsecase.GetHonorTypes(c.Request().Context())),
	})
}
