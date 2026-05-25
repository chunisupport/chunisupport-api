package dto

import "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"

// MasterItemDTO はマスタデータの単一項目を表します。
type MasterItemDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// VersionDTO はバージョンマスタを表します。
type VersionDTO struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ReleasedAt string `json:"released_at"`
}

// ToVersionDTOs は []masterdata.Version を []*VersionDTO に変換します。
func ToVersionDTOs(versions []masterdata.Version) []*VersionDTO {
	dtos := make([]*VersionDTO, len(versions))
	for i, v := range versions {
		dtos[i] = &VersionDTO{
			ID:         int(v.ID),
			Name:       v.Name,
			ReleasedAt: v.ReleasedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return dtos
}

// VersionSummaryDTO は専用バージョン一覧API向けのバージョンマスタを表します。
type VersionSummaryDTO struct {
	Name       string `json:"name"`
	ReleasedAt string `json:"released_at"`
}

// ToVersionSummaryDTOs は []masterdata.Version を []*VersionSummaryDTO に変換します。
func ToVersionSummaryDTOs(versions []masterdata.Version) []*VersionSummaryDTO {
	dtos := make([]*VersionSummaryDTO, len(versions))
	for i, v := range versions {
		dtos[i] = &VersionSummaryDTO{
			Name:       v.Name,
			ReleasedAt: v.ReleasedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return dtos
}

// VersionSummariesResponse は専用バージョン一覧取得APIのレスポンスを表します。
type VersionSummariesResponse struct {
	Versions []*VersionSummaryDTO `json:"versions"`
}

// HonorTypesResponse は称号タイプ一覧取得APIのレスポンスを表します。
type HonorTypesResponse struct {
	HonorTypes []*MasterItemDTO `json:"honor_types"`
}

// MasterDataResponse はマスタデータ取得APIのレスポンスを表します。
type MasterDataResponse struct {
	Genres           []*MasterItemDTO `json:"genres"`
	Difficulties     []*MasterItemDTO `json:"difficulties"`
	AccountTypes     []*MasterItemDTO `json:"account_types"`
	Versions         []*VersionDTO    `json:"versions"`
	RatingBands      []*RatingBandDTO `json:"rating_bands"`
	AchievementTypes []*MasterItemDTO `json:"achievement_types"`
	ClassEmblems     []*MasterItemDTO `json:"class_emblems"`
	ClassEmblemBases []*MasterItemDTO `json:"class_emblem_bases"`
	ClearLamps       []*MasterItemDTO `json:"clear_lamps"`
	ComboLamps       []*MasterItemDTO `json:"combo_lamps"`
	FullChains       []*MasterItemDTO `json:"full_chains"`
	Slots            []*MasterItemDTO `json:"slots"`
	HonorTypes       []*MasterItemDTO `json:"honor_types"`
}

// RatingBandDTO はレーティング帯マスタのDTOです。
type RatingBandDTO struct {
	ID           int      `json:"id"`
	Label        string   `json:"label"`
	MinInclusive *float64 `json:"min_inclusive"`
	MaxExclusive *float64 `json:"max_exclusive"`
	SortOrder    int      `json:"sort_order"`
}
