package dto

// MasterItemDTO はマスタデータの単一項目を表します。
type MasterItemDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BooleanChoiceDTO は真偽値選択肢を表します。
type BooleanChoiceDTO struct {
	Value bool   `json:"value"`
	Label string `json:"label"`
}

// MasterDataResponse はマスタデータ取得APIのレスポンスを表します。
type MasterDataResponse struct {
	Genres         []*MasterItemDTO    `json:"genres"`
	Difficulties   []*MasterItemDTO    `json:"difficulties"`
	IsConstUnknown []*BooleanChoiceDTO `json:"is_const_unknown"`
	AccountTypes   []*MasterItemDTO    `json:"account_types"`
}
