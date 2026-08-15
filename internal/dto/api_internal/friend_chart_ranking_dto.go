package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// FriendChartRankingSongDTO はフレンドランキング対象楽曲の概要です。
type FriendChartRankingSongDTO struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

// FriendChartRankingChartDTO はフレンドランキング対象譜面の概要です。
type FriendChartRankingChartDTO struct {
	Difficulty     string                       `json:"difficulty"`
	Const          *chartconstant.ChartConstant `json:"const,omitempty"`
	IsConstUnknown *bool                        `json:"is_const_unknown,omitempty"`
	LevelStar      *int                         `json:"level_star,omitempty"`
	Attribute      *string                      `json:"attribute,omitempty"`
	IsWorldsend    bool                         `json:"is_worldsend"`
}

// FriendChartRankingEntryDTO は譜面単位フレンドランキングの1件です。
type FriendChartRankingEntryDTO struct {
	Rank             int       `json:"rank"`
	UserID           int       `json:"user_id"`
	Username         string    `json:"username"`
	PlayerName       string    `json:"player_name"`
	Score            uint32    `json:"score"`
	Rating           *float64  `json:"rating,omitempty"`
	Overpower        *float64  `json:"overpower,omitempty"`
	OverpowerPercent *float64  `json:"overpower_percent,omitempty"`
	ClearLamp        *string   `json:"clear_lamp"`
	ComboLamp        *string   `json:"combo_lamp"`
	FullChain        *string   `json:"full_chain"`
	UpdatedAt        time.Time `json:"updated_at"`
	IsSelf           bool      `json:"is_self"`
}

// FriendChartRankingResponse は譜面単位フレンドランキングのレスポンスです。
type FriendChartRankingResponse struct {
	Song    FriendChartRankingSongDTO    `json:"song"`
	Chart   FriendChartRankingChartDTO   `json:"chart"`
	Ranking []FriendChartRankingEntryDTO `json:"ranking"`
	MyRank  *int                         `json:"my_rank"`
	Total   int                          `json:"total"`
}
