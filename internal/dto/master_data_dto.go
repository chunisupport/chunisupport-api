package dto

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
)

// MasterItemDTO はマスタデータの単一項目を表します。
type MasterItemDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GenreDTO はジャンルマスタを表します。
type GenreDTO struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
}

// ToGenreDTOs は []masterdata.Genre を []*GenreDTO に変換します。
func ToGenreDTOs(genres []masterdata.Genre) []*GenreDTO {
	dtos := make([]*GenreDTO, len(genres))
	for i, g := range genres {
		dtos[i] = &GenreDTO{ID: g.ID, Name: g.Name, ShortName: g.ShortName}
	}
	return dtos
}

// GenresResponse はジャンル一覧取得APIのレスポンスを表します。
type GenresResponse struct {
	Genres []*GenreDTO `json:"genres"`
}

// VersionDTO はバージョンマスタを表します。
type VersionDTO struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ShortName  string `json:"short_name"`
	ReleasedAt string `json:"released_at"`
}

// ToVersionDTOs は []masterdata.Version を []*VersionDTO に変換します。
func ToVersionDTOs(versions []masterdata.Version) []*VersionDTO {
	dtos := make([]*VersionDTO, len(versions))
	for i, v := range versions {
		dtos[i] = &VersionDTO{
			ID:         int(v.ID),
			Name:       v.Name,
			ShortName:  v.ShortName,
			ReleasedAt: v.ReleasedAt.Format(time.DateOnly),
		}
	}
	return dtos
}

// VersionSummaryDTO は専用バージョン一覧API向けのバージョンマスタを表します。
type VersionSummaryDTO struct {
	Name       string `json:"name"`
	ShortName  string `json:"short_name"`
	ReleasedAt string `json:"released_at"`
}

// ToVersionSummaryDTOs は []masterdata.Version を []*VersionSummaryDTO に変換します。
func ToVersionSummaryDTOs(versions []masterdata.Version) []*VersionSummaryDTO {
	dtos := make([]*VersionSummaryDTO, len(versions))
	for i, v := range versions {
		dtos[i] = &VersionSummaryDTO{
			Name:       v.Name,
			ShortName:  v.ShortName,
			ReleasedAt: v.ReleasedAt.Format(time.DateOnly),
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
	NameFolders      []*NameFolderDTO `json:"name_folders"`
	Genres           []*GenreDTO      `json:"genres"`
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
	Possessions      []*MasterItemDTO `json:"possessions"`
}

// RatingBandDTO はレーティング帯マスタのDTOです。
type RatingBandDTO struct {
	ID           int      `json:"id"`
	Label        string   `json:"label"`
	MinInclusive *float64 `json:"min_inclusive"`
	MaxExclusive *float64 `json:"max_exclusive"`
	SortOrder    int      `json:"sort_order"`
}

// PermissionsResponse は数値IDを公開せず、権限変更にそのまま利用できる名前を返します。
type PermissionsResponse struct {
	Permissions []string `json:"permissions"`
}

// NameFolderDTO は内部IDを公開せず選択肢を識別する名前順フォルダです。
type NameFolderDTO struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// NameFoldersResponse は名前順フォルダ一覧取得APIのレスポンスです。
type NameFoldersResponse struct {
	NameFolders []*NameFolderDTO `json:"name_folders"`
}

// ToNameFolderDTOs は名前順フォルダをAPI公開用に変換します。
func ToNameFolderDTOs(folders []masterdata.NameFolder) []*NameFolderDTO {
	items := make([]*NameFolderDTO, len(folders))
	for i, folder := range folders {
		items[i] = &NameFolderDTO{Code: folder.Code, Name: folder.Name, SortOrder: folder.SortOrder}
	}
	return items
}
