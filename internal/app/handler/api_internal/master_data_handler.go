package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

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

	genres := make([]*dto.MasterItemDTO, len(out.Genres))
	for i, g := range out.Genres {
		genres[i] = &dto.MasterItemDTO{ID: g.ID, Name: g.Name}
	}

	difficulties := make([]*dto.MasterItemDTO, len(out.Difficulties))
	for i, d := range out.Difficulties {
		difficulties[i] = &dto.MasterItemDTO{ID: d.ID, Name: d.Name}
	}

	accountTypes := make([]*dto.MasterItemDTO, len(out.AccountTypes))
	for i, a := range out.AccountTypes {
		accountTypes[i] = &dto.MasterItemDTO{ID: a.ID, Name: a.Name}
	}

	versions := make([]*dto.VersionDTO, len(out.Versions))
	for i, v := range out.Versions {
		versions[i] = &dto.VersionDTO{
			ID:         int(v.ID),
			Name:       v.Name,
			ReleasedAt: v.ReleasedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

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

	achievementTypes := make([]*dto.MasterItemDTO, len(out.AchievementTypes))
	for i, a := range out.AchievementTypes {
		achievementTypes[i] = &dto.MasterItemDTO{ID: a.ID, Name: a.Name}
	}

	return c.JSON(http.StatusOK, &dto.MasterDataResponse{
		Genres:           genres,
		Difficulties:     difficulties,
		AccountTypes:     accountTypes,
		Versions:         versions,
		RatingBands:      ratingBands,
		AchievementTypes: achievementTypes,
	})
}
