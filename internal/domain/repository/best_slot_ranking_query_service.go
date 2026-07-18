package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// BestSlotRankingCursor はランキングのキーセットページング位置です。
type BestSlotRankingCursor struct {
	RatingBand           string
	BestPlayerPercentage float64
	BestPlayerCount      int
	SongDisplayID        string
	Difficulty           string
}

// BestSlotRankingItem はベスト枠採用率ランキングの読み取りモデルです。
type BestSlotRankingItem struct {
	Rank                 int
	SongDisplayID        string
	SongTitle            string
	Difficulty           string
	ChartConst           chartconstant.ChartConstant
	IsConstUnknown       bool
	BestPlayerCount      int
	BestPlayerPercentage float64
	AverageScore         *float64
}

// BestSlotRankingPage はランキング1ページ分の読み取り結果です。
type BestSlotRankingPage struct {
	EligiblePlayerCount int
	Items               []BestSlotRankingItem
	NextCursor          *BestSlotRankingCursor
}

// BestSlotRankingQueryService は統計DBと楽曲DBを横断したランキング読み取りを扱います。
type BestSlotRankingQueryService interface {
	List(ctx context.Context, ratingBandID int, cursor *BestSlotRankingCursor, limit int) (*BestSlotRankingPage, error)
}
