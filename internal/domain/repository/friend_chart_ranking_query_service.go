package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// FriendChartRankingChart はフレンドランキング対象の通常譜面情報です。
type FriendChartRankingChart struct {
	SongDisplayID   string
	SongTitle       string
	SongArtist      string
	Difficulty      string
	Const           chartconstant.ChartConstant
	IsConstUnknown  bool
	ChartID         int
	ChartDifficulty int
}

// FriendChartRankingRecord は譜面単位フレンドランキングの1件です。
type FriendChartRankingRecord struct {
	UserID           int
	Username         string
	PlayerName       string
	Score            uint32
	Rating           float64
	Overpower        float64
	OverpowerPercent float64
	ClearLamp        string
	ComboLamp        string
	FullChain        string
	UpdatedAt        time.Time
}

// FriendChartRankingQueryService は譜面単位フレンドランキングの読み取りを扱います。
type FriendChartRankingQueryService interface {
	FindChart(ctx context.Context, exec Executor, displayID string, difficulty string) (*FriendChartRankingChart, error)
	ListRecords(ctx context.Context, exec Executor, userID int, chartID int) ([]*FriendChartRankingRecord, error)
}
