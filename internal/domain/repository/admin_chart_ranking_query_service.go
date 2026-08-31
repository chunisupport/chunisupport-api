package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

type AdminChartRankingChart struct {
	SongDisplayID  string
	SongTitle      string
	SongArtist     string
	Difficulty     string
	Const          chartconstant.ChartConstant
	IsConstUnknown bool
	LevelStar      *int
	Attribute      *string
	IsWorldsend    bool
}

type AdminChartRankingRecord struct {
	Username   string
	PlayerName string
	Score      uint32
	ClearLamp  string
	ComboLamp  string
	FullChain  string
	UpdatedAt  time.Time
}

type AdminChartRankingData struct {
	Chart   *AdminChartRankingChart
	Records []*AdminChartRankingRecord
	Total   int
}

type AdminChartRankingQueryService interface {
	GetStandard(ctx context.Context, displayID string, difficulty string, limit int) (*AdminChartRankingData, error)
	GetWorldsend(ctx context.Context, displayID string, limit int) (*AdminChartRankingData, error)
}
