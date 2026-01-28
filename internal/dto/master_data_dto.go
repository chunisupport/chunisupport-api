package dto

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

// MasterDataResponse はマスタデータ取得APIのレスポンスを表します。
type MasterDataResponse struct {
	Genres       []*MasterItemDTO `json:"genres"`
	Difficulties []*MasterItemDTO `json:"difficulties"`
	AccountTypes []*MasterItemDTO `json:"account_types"`
	Versions     []*VersionDTO    `json:"versions"`
}
