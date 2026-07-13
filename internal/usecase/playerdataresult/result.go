// Package playerdataresult はプレイヤーデータ登録ユースケースの出力型を提供します。
package playerdataresult

import "time"

type Summary struct {
	Name             string
	Level            int
	Rating           *float64
	LastPlayedAt     *time.Time
	OverpowerValue   *float64
	OverpowerPercent *float64
}
type Profile struct {
	PlayerID          int
	Name              string
	Level             int
	Rating            *float64
	ClassEmblemID     *int
	ClassEmblemBaseID *int
	LastPlayedAt      *time.Time
	OverpowerValue    *float64
	OverpowerPercent  *float64
}
type Int64Diff struct {
	Before int64
	After  int64
	Delta  int64
}
type IntDiff struct {
	Before int
	After  int
	Delta  int
}
type RecordStatisticsDiff struct {
	AJ      IntDiff
	FC      IntDiff
	CLR     IntDiff
	FCH     IntDiff
	MAX     IntDiff
	SSSPlus IntDiff
	SSS     IntDiff
	SSPlus  IntDiff
	SS      IntDiff
	SPlus   IntDiff
	S       IntDiff
}
type StatisticsGroup struct {
	TotalHighScore   Int64Diff
	RecordStatistics RecordStatisticsDiff
}
type Statistics struct {
	Overall      StatisticsGroup
	ByDifficulty map[string]StatisticsGroup
}
type Counts struct {
	FullRecordsUpserted             int
	WorldsendRecordsUpserted        int
	FullRecordsSkipped              int
	WorldsendRecordsSkipped         int
	HonorsSkipped                   int
	FullRecordsActuallyChanged      int
	WorldsendRecordsActuallyChanged int
	CourseRecordsUpserted           int
	CourseRecordsSkipped            int
	CourseRecordsActuallyChanged    int
}
type SkippedRecord struct {
	RecordType string
	Reason     string
	Details    string
}
type RecordState struct {
	Score     int
	ClearLamp *string
	ComboLamp *string
	FullChain *string
	IsClear   *bool
}
type RecordChange struct {
	RecordType  string
	ChangeType  string
	Idx         string
	Diff        string
	CourseClass string
	Before      *RecordState
	After       RecordState
}
type Result struct {
	PlayerID       int
	AppVersion     string
	ImportedAt     time.Time
	Profile        Profile
	Summary        Summary
	Statistics     Statistics
	Counts         Counts
	Changes        []RecordChange
	SkippedRecords []SkippedRecord
}

type PlayerDataSummary = Summary
type PlayerDataProfile = Profile
type PlayerDataInt64Diff = Int64Diff
type PlayerDataIntDiff = IntDiff
type PlayerDataRecordStatisticsDiff = RecordStatisticsDiff
type PlayerDataStatisticsGroup = StatisticsGroup
type PlayerDataStatistics = Statistics
type PlayerDataCounts = Counts
type PlayerDataRecordState = RecordState
type PlayerDataRecordChange = RecordChange
type PlayerDataResult = Result
