package api_internal

import "encoding/json"

// RecordFilterRequest は保存済み譜面フィルタの作成・更新リクエストです。
type RecordFilterRequest struct {
	Name          string          `json:"name"`
	FilterType    string          `json:"filter_type"`
	SchemaVersion int             `json:"schema_version"`
	Filter        json.RawMessage `json:"filter"`
}

// RecordFilterResponse は保存済み譜面フィルタのレスポンスです。
type RecordFilterResponse struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	FilterType    string          `json:"filter_type"`
	SchemaVersion int             `json:"schema_version"`
	Filter        json.RawMessage `json:"filter"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}

// RecordFiltersResponse は保存済み譜面フィルタ一覧のレスポンスです。
type RecordFiltersResponse struct {
	Filters []*RecordFilterResponse `json:"filters"`
}
