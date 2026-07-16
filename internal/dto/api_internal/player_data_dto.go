package api_internal

import "time"

type PlayerDataSummary struct {
	Name             string     `json:"name"`
	Level            int        `json:"level"`
	Rating           *float64   `json:"rating"`
	LastPlayedAt     *time.Time `json:"last_played_at"`
	OverpowerValue   *float64   `json:"overpower_value"`
	OverpowerPercent *float64   `json:"overpower_percentage"`
}
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
type PlayerDataInt64Diff struct {
	Before int64 `json:"before"`
	After  int64 `json:"after"`
	Delta  int64 `json:"delta"`
}
type PlayerDataIntDiff struct {
	Before int `json:"before"`
	After  int `json:"after"`
	Delta  int `json:"delta"`
}
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
	SPlus   PlayerDataIntDiff `json:"s_plus"`
	S       PlayerDataIntDiff `json:"s"`
}
type PlayerDataStatisticsGroup struct {
	TotalHighScore   PlayerDataInt64Diff            `json:"total_high_score"`
	RecordStatistics PlayerDataRecordStatisticsDiff `json:"record_statistics"`
}
type PlayerDataStatistics struct {
	Overall      PlayerDataStatisticsGroup            `json:"overall"`
	ByDifficulty map[string]PlayerDataStatisticsGroup `json:"by_difficulty"`
}
type PlayerDataCounts struct {
	FullRecordsUpserted             int `json:"standard_records_upserted"`
	WorldsendRecordsUpserted        int `json:"worldsend_records_upserted"`
	FullRecordsSkipped              int `json:"standard_records_skipped"`
	WorldsendRecordsSkipped         int `json:"worldsend_records_skipped"`
	HonorsSkipped                   int `json:"honors_skipped"`
	FullRecordsActuallyChanged      int `json:"standard_records_actually_changed"`
	WorldsendRecordsActuallyChanged int `json:"worldsend_records_actually_changed"`
	CourseRecordsUpserted           int `json:"course_records_upserted"`
	CourseRecordsSkipped            int `json:"course_records_skipped"`
	CourseRecordsActuallyChanged    int `json:"course_records_actually_changed"`
}
type SkippedRecord struct {
	RecordType string `json:"record_type"`
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}
type PlayerDataRecordState struct {
	Score     int     `json:"score"`
	ClearLamp *string `json:"clear_lamp"`
	ComboLamp *string `json:"combo_lamp"`
	FullChain *string `json:"full_chain"`
	IsClear   *bool   `json:"is_clear,omitempty"`
}
type PlayerDataRecordChange struct {
	RecordType  string                 `json:"record_type"`
	ChangeType  string                 `json:"change_type"`
	Idx         string                 `json:"idx"`
	Diff        string                 `json:"diff,omitempty"`
	CourseClass string                 `json:"course_class,omitempty"`
	Before      *PlayerDataRecordState `json:"before"`
	After       PlayerDataRecordState  `json:"after"`
}
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

// PlayerLatestUpdateResult は保存済みの最新プレイヤーデータ登録結果です。
type PlayerLatestUpdateResult struct {
	SchemaVersion int                      `json:"schema_version"`
	PlayerID      int                      `json:"player_id"`
	AppVersion    string                   `json:"app_ver"`
	ImportedAt    time.Time                `json:"imported_at"`
	Profile       PlayerDataProfile        `json:"profile"`
	Summary       PlayerDataSummary        `json:"summary"`
	Statistics    PlayerDataStatistics     `json:"statistics"`
	Counts        PlayerDataCounts         `json:"counts"`
	Changes       []PlayerDataRecordChange `json:"changes"`
}
