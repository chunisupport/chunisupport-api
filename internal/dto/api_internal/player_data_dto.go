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
	FullRecordsUpserted             int `json:"full_records_upserted"`
	WorldsendRecordsUpserted        int `json:"worldsend_records_upserted"`
	FullRecordsSkipped              int `json:"full_records_skipped"`
	WorldsendRecordsSkipped         int `json:"worldsend_records_skipped"`
	HonorsSkipped                   int `json:"honors_skipped"`
	FullRecordsActuallyChanged      int `json:"full_records_actually_changed"`
	WorldsendRecordsActuallyChanged int `json:"worldsend_records_actually_changed"`
}

// SkippedRecord はスキップされたレコードの情報です。
type SkippedRecord struct {
	RecordType string `json:"record_type"` // "full", "worldsend", "honor"
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

// PlayerDataRecordState は差分表示で比較対象にするスコア状態です。
// ランプ名はマスタの Name を返し、none 相当および未設定は null で返します。
type PlayerDataRecordState struct {
	Score     int     `json:"score"`
	ClearLamp *string `json:"clear_lamp"`
	ComboLamp *string `json:"combo_lamp"`
	FullChain *string `json:"full_chain"`
}

// PlayerDataRecordChange は登録前後で実際に変化したレコードの差分です。
type PlayerDataRecordChange struct {
	RecordType string                 `json:"record_type"`
	ChangeType string                 `json:"change_type"`
	Idx        string                 `json:"idx"`
	Diff       string                 `json:"diff"`
	Before     *PlayerDataRecordState `json:"before"`
	After      PlayerDataRecordState  `json:"after"`
}

// PlayerDataResult は登録APIのレスポンス全体です。
type PlayerDataResult struct {
	PlayerID       int                      `json:"player_id"`
	AppVersion     string                   `json:"app_ver"`
	ImportedAt     time.Time                `json:"imported_at"`
	Summary        PlayerDataSummary        `json:"summary"`
	Counts         PlayerDataCounts         `json:"counts"`
	Changes        []PlayerDataRecordChange `json:"changes"`
	SkippedRecords []SkippedRecord          `json:"skipped_records"`
}
