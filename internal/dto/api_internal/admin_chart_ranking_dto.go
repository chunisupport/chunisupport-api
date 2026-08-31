package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

type AdminChartRankingSongDTO struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

type AdminChartRankingChartDTO struct {
	Difficulty     string                       `json:"difficulty"`
	Const          *chartconstant.ChartConstant `json:"const,omitempty"`
	IsConstUnknown *bool                        `json:"is_const_unknown,omitempty"`
	LevelStar      *int                         `json:"level_star,omitempty"`
	Attribute      *string                      `json:"attribute,omitempty"`
	IsWorldsend    bool                         `json:"is_worldsend"`
}

type AdminChartRankingEntryDTO struct {
	Rank             int       `json:"rank"`
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
}

type AdminChartRankingResponse struct {
	Song    AdminChartRankingSongDTO    `json:"song"`
	Chart   AdminChartRankingChartDTO   `json:"chart"`
	Ranking []AdminChartRankingEntryDTO `json:"ranking"`
	Total   int                         `json:"total"`
}
