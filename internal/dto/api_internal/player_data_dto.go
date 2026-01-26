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
	FullRecordsSkipped       int `json:"full_records_skipped"`
	WorldsendRecordsSkipped  int `json:"worldsend_records_skipped"`
	HonorsSkipped            int `json:"honors_skipped"`
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
	SkippedRecords []SkippedRecord   `json:"skipped_records"`
}
