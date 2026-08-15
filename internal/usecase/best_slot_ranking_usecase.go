package usecase

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// BestSlotRankingEntry はベスト枠採用率ランキングの1件です。
type BestSlotRankingEntry struct {
	Rank                 int
	SongID               string
	Title                string
	Difficulty           string
	ChartConst           chartconstant.ChartConstant
	IsConstUnknown       bool
	BestPlayerCount      int
	BestPlayerPercentage float64
	AverageScore         *float64
}

// BestSlotRankingResult は指定レート帯のベスト枠採用率ランキングです。
type BestSlotRankingResult struct {
	RatingBand          string
	EligiblePlayerCount int
	Ranking             []BestSlotRankingEntry
	NextCursor          *repository.BestSlotRankingCursor
}

// BestSlotRankingUsecase はベスト枠平均レート帯別ランキングを提供します。
type BestSlotRankingUsecase interface {
	Get(ctx context.Context, ratingBand string, cursor *repository.BestSlotRankingCursor, limit int) (*BestSlotRankingResult, error)
}

type bestSlotRankingUsecase struct {
	query          repository.BestSlotRankingQueryService
	masterProvider repository.ChartStatsMasterProvider
}

// NewBestSlotRankingUsecase はベスト枠採用率ランキングのユースケースを生成します。
func NewBestSlotRankingUsecase(query repository.BestSlotRankingQueryService, masterProvider repository.ChartStatsMasterProvider) BestSlotRankingUsecase {
	return &bestSlotRankingUsecase{
		query:          query,
		masterProvider: masterProvider,
	}
}

// Get は公開ラベルを内部のレート帯へ解決してランキングを取得します。
func (u *bestSlotRankingUsecase) Get(ctx context.Context, ratingBand string, cursor *repository.BestSlotRankingCursor, limit int) (*BestSlotRankingResult, error) {
	if cursor != nil && cursor.RatingBand != ratingBand {
		return nil, fmt.Errorf("%w: cursor rating band does not match", ErrInvalidBestSlotRankingCursor)
	}
	var ratingBandID *int
	for _, band := range u.masterProvider.RatingBands() {
		if band.Label == ratingBand {
			id := band.ID
			ratingBandID = &id
			break
		}
	}
	if ratingBandID == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRatingBand, ratingBand)
	}

	page, err := u.query.List(ctx, *ratingBandID, cursor, limit)
	if err != nil {
		return nil, err
	}
	result := &BestSlotRankingResult{
		RatingBand:          ratingBand,
		EligiblePlayerCount: page.EligiblePlayerCount,
		Ranking:             make([]BestSlotRankingEntry, 0, len(page.Items)),
		NextCursor:          page.NextCursor,
	}
	if result.NextCursor != nil {
		result.NextCursor.RatingBand = ratingBand
	}
	for _, item := range page.Items {
		result.Ranking = append(result.Ranking, BestSlotRankingEntry{
			Rank:                 item.Rank,
			SongID:               item.SongDisplayID,
			Title:                item.SongTitle,
			Difficulty:           item.Difficulty,
			ChartConst:           item.ChartConst,
			IsConstUnknown:       item.IsConstUnknown,
			BestPlayerCount:      item.BestPlayerCount,
			BestPlayerPercentage: item.BestPlayerPercentage,
			AverageScore:         item.AverageScore,
		})
	}
	return result, nil
}
