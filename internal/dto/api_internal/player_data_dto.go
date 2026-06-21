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

// PlayerDataProfile は登録後のプレイヤープロフィール情報です。
type PlayerDataProfile struct {
	PlayerID          int        `json:"player_id"`
	Name              string     `json:"name"`
	Level             int        `json:"level"`
	Rating            *float64   `json:"rating"`
	ClassEmblemID     *int       `json:"class_emblem_id"`
	ClassEmblemBaseID *int       `json:"class_emblem_base_id"`
	LastPlayedAt      *time.Time `json:"last_played_at"`
	OverpowerValue    *float64   `json:"overpower_value"`
	OverpowerPercent  *float64   `json:"overpower_percent"`
}

// PlayerDataInt64Diff は64bit整数の登録前後差分です。
type PlayerDataInt64Diff struct {
	Before int64 `json:"before"`
	After  int64 `json:"after"`
	Delta  int64 `json:"delta"`
}

// PlayerDataIntDiff は整数の登録前後差分です。
type PlayerDataIntDiff struct {
	Before int `json:"before"`
	After  int `json:"after"`
	Delta  int `json:"delta"`
}

// PlayerDataRecordStatisticsDiff は通常譜面の達成件数差分です。
type PlayerDataRecordStatisticsDiff struct {
	AJ      PlayerDataIntDiff `json:"aj"`
	FC      PlayerDataIntDiff `json:"fc"`
	CLR     PlayerDataIntDiff `json:"clr"`
	FCH     PlayerDataIntDiff `json:"fch"`
	MAX     PlayerDataIntDiff `json:"max"`
	SSSPlus PlayerDataIntDiff `json:"sss_plus"`
	SSS     PlayerDataIntDiff `json:"sss"`
	SSPlus  PlayerDataIntDiff `json:"ss_plus"`
	SS      PlayerDataIntDiff `json:"ss"`
}

// PlayerDataStatisticsGroup はスコア合計と達成件数の差分です。
type PlayerDataStatisticsGroup struct {
	TotalHighScore   PlayerDataInt64Diff            `json:"total_high_score"`
	RecordStatistics PlayerDataRecordStatisticsDiff `json:"record_statistics"`
}

// PlayerDataStatistics は全体・難易度別の通常譜面集計差分です。
type PlayerDataStatistics struct {
	Overall      PlayerDataStatisticsGroup            `json:"overall"`
	ByDifficulty map[string]PlayerDataStatisticsGroup `json:"by_difficulty"`
}

// PlayerDataCounts は各種レコードのアップサート件数を表します。
type PlayerDataCounts struct {
	FullRecordsUpserted             int `json:"standard_records_upserted"`
	WorldsendRecordsUpserted        int `json:"worldsend_records_upserted"`
	FullRecordsSkipped              int `json:"standard_records_skipped"`
	WorldsendRecordsSkipped         int `json:"worldsend_records_skipped"`
	HonorsSkipped                   int `json:"honors_skipped"`
	FullRecordsActuallyChanged      int `json:"standard_records_actually_changed"`
	WorldsendRecordsActuallyChanged int `json:"worldsend_records_actually_changed"`
}

// SkippedRecord はスキップされたレコードの情報です。
type SkippedRecord struct {
	RecordType string `json:"record_type"` // "standard", "worldsend", "honor"
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
	Profile        PlayerDataProfile        `json:"profile"`
	Summary        PlayerDataSummary        `json:"summary"`
	Statistics     PlayerDataStatistics     `json:"statistics"`
	Counts         PlayerDataCounts         `json:"counts"`
	Changes        []PlayerDataRecordChange `json:"changes"`
	SkippedRecords []SkippedRecord          `json:"skipped_records"`
}
