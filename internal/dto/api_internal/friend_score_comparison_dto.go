package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// FriendScoreComparisonUserDTO は比較対象ユーザーの公開識別子です。
type FriendScoreComparisonUserDTO struct {
	Username   string `json:"username"`
	PlayerName string `json:"player_name"`
}

// FriendScoreComparisonSummaryDTO は指定難易度の勝敗とプレイ状態の集計です。
type FriendScoreComparisonSummaryDTO struct {
	TotalCharts      int `json:"total_charts"`
	SelfWins         int `json:"self_wins"`
	Draws            int `json:"draws"`
	FriendWins       int `json:"friend_wins"`
	SelfPlayed       int `json:"self_played"`
	FriendPlayed     int `json:"friend_played"`
	BothPlayed       int `json:"both_played"`
	SelfOnlyPlayed   int `json:"self_only_played"`
	FriendOnlyPlayed int `json:"friend_only_played"`
	BothUnplayed     int `json:"both_unplayed"`
}

// FriendScoreComparisonRecordDTO は未プレイを正規化した1人分の現在レコードです。
type FriendScoreComparisonRecordDTO struct {
	IsPlayed  bool       `json:"is_played"`
	Score     uint32     `json:"score"`
	ClearLamp *string    `json:"clear_lamp"`
	ComboLamp *string    `json:"combo_lamp"`
	FullChain *string    `json:"full_chain"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// FriendScoreComparisonSongDTO は比較対象楽曲の公開概要です。
type FriendScoreComparisonSongDTO struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

// FriendScoreComparisonChartDTO は比較対象譜面の公開概要です。
type FriendScoreComparisonChartDTO struct {
	Const          chartconstant.ChartConstant `json:"const"`
	IsConstUnknown bool                        `json:"is_const_unknown"`
}

// FriendScoreComparisonItemDTO は1譜面の比較結果です。
type FriendScoreComparisonItemDTO struct {
	Song            FriendScoreComparisonSongDTO   `json:"song"`
	Chart           FriendScoreComparisonChartDTO  `json:"chart"`
	Self            FriendScoreComparisonRecordDTO `json:"self"`
	Friend          FriendScoreComparisonRecordDTO `json:"friend"`
	ScoreDifference int                            `json:"score_difference"`
	Result          string                         `json:"result"`
}

// FriendScoreComparisonResponse はフレンドスコア比較のレスポンスです。
type FriendScoreComparisonResponse struct {
	Difficulty string                          `json:"difficulty"`
	Self       FriendScoreComparisonUserDTO    `json:"self"`
	Friend     FriendScoreComparisonUserDTO    `json:"friend"`
	Summary    FriendScoreComparisonSummaryDTO `json:"summary"`
	Items      []FriendScoreComparisonItemDTO  `json:"items"`
}
