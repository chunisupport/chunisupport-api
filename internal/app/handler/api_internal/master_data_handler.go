package api_internal

import (
	"cmp"
	"net/http"
	"slices"

	"github.com/Qman110101/chunisupport-api/internal/dto"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/labstack/echo/v4"
)

// MasterDataHandler はマスタデータ関連のハンドラです。
type MasterDataHandler struct {
	masterCache *masterdata.Cache
}

// NewMasterDataHandler は新しい MasterDataHandler を生成します。
func NewMasterDataHandler(masterCache *masterdata.Cache) *MasterDataHandler {
	return &MasterDataHandler{
		masterCache: masterCache,
	}
}

// sortMasterItemsByID はMasterItemDTOスライスをID順にソートします。
func sortMasterItemsByID(items []*dto.MasterItemDTO) {
	slices.SortFunc(items, func(a, b *dto.MasterItemDTO) int {
		return cmp.Compare(a.ID, b.ID)
	})
}

// GetMasterData はフロントエンド向けにマスタデータを返却します。
func (h *MasterDataHandler) GetMasterData(c echo.Context) error {
	// Genres をID順にソートして配列化
	genres := make([]*dto.MasterItemDTO, 0, len(h.masterCache.Genres))
	for _, item := range h.masterCache.Genres {
		genres = append(genres, &dto.MasterItemDTO{
			ID:   item.ID,
			Name: item.Name,
		})
	}
	sortMasterItemsByID(genres)

	// Difficulties をID順にソートして配列化
	difficulties := make([]*dto.MasterItemDTO, 0, len(h.masterCache.Difficulties))
	for _, item := range h.masterCache.Difficulties {
		difficulties = append(difficulties, &dto.MasterItemDTO{
			ID:   item.ID,
			Name: item.Name,
		})
	}
	sortMasterItemsByID(difficulties)

	// AccountTypes をID順にソートして配列化
	accountTypes := make([]*dto.MasterItemDTO, 0, len(h.masterCache.AccountTypes))
	for _, item := range h.masterCache.AccountTypes {
		accountTypes = append(accountTypes, &dto.MasterItemDTO{
			ID:   item.ID,
			Name: item.Name,
		})
	}
	sortMasterItemsByID(accountTypes)

	// Versions をID順にソートして配列化
	versions := make([]*dto.VersionDTO, 0, len(h.masterCache.Versions))
	for _, item := range h.masterCache.Versions {
		versions = append(versions, &dto.VersionDTO{
			ID:         int(item.ID),
			Name:       item.Name,
			ReleasedAt: item.ReleasedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	slices.SortFunc(versions, func(a, b *dto.VersionDTO) int {
		return cmp.Compare(a.ID, b.ID)
	})

	response := &dto.MasterDataResponse{
		Genres:       genres,
		Difficulties: difficulties,
		AccountTypes: accountTypes,
		Versions:     versions,
	}

	return c.JSON(http.StatusOK, response)
}
