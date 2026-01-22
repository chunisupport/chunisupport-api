package api_internal

import "time"

// PlayerDataSummary は登録結果のサマリー情報です。
type PlayerDataSummary struct {
	Name             string     `json:"name"`
	Level            int        `json:"level"`
	Rating           *float64   `json:"rating"`
	LastPlayedAt     *time.Time `json:"last_played_at"`
	OverpowerValue   *float64   `json:"overpower_value"`
	OverpowerPercent *float64   `json:"overpower_percentage"`
}

// PlayerDataCounts は各種レコードのアップサート件数を表します。
type PlayerDataCounts struct {
	FullRecordsUpserted      int `json:"full_records_upserted"`
	WorldsendRecordsUpserted int `json:"worldsend_records_upserted"`
	FullRecordsChanged       int `json:"full_records_changed"`
	WorldsendRecordsChanged  int `json:"worldsend_records_changed"`
	FullRecordsSkipped       int `json:"full_records_skipped"`
	WorldsendRecordsSkipped  int `json:"worldsend_records_skipped"`
	HonorsSkipped            int `json:"honors_skipped"`
}

// PlayerDataDiffRecord はスコア差分で使用する軽量なレコード情報です。
// レスポンスサイズ削減のため、PlayerRecordDTO から必要最小限のフィールドのみ抽出しています。
type PlayerDataDiffRecord struct {
	Difficulty     string  `json:"difficulty"`
	Title          string  `json:"title"`
	Const          float64 `json:"const"`
	IsConstUnknown bool    `json:"is_const_unknown"`
	Score          uint32  `json:"score"`
	ClearLamp      string  `json:"clear_lamp"`
	ComboLamp      *string `json:"combo_lamp"`
	FullChain      *string `json:"full_chain"`
}

// PlayerDataDiff は1件のスコア差分情報を表します。
type PlayerDataDiff struct {
	Before        *PlayerDataDiffRecord `json:"before,omitempty"`
	After         *PlayerDataDiffRecord `json:"after"`
	ChangedFields []string              `json:"changed_fields"`
}

// PlayerDataDiffSet は種別ごとの差分リストです。
type PlayerDataDiffSet struct {
	Full      []PlayerDataDiff `json:"full"`
	Worldsend []PlayerDataDiff `json:"worldsend"`
}

// SkippedRecord はスキップされたレコードの情報です。
type SkippedRecord struct {
	RecordType string `json:"record_type"` // "full", "worldsend", "honor"
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

// PlayerDataResult は登録APIのレスポンス全体です。
type PlayerDataResult struct {
	PlayerID       int               `json:"player_id"`
	AppVersion     string            `json:"app_ver"`
	ImportedAt     time.Time         `json:"imported_at"`
	Summary        PlayerDataSummary `json:"summary"`
	Counts         PlayerDataCounts  `json:"counts"`
	DiffRecords    PlayerDataDiffSet `json:"diff_records"`
	SkippedRecords []SkippedRecord   `json:"skipped_records"`
}
