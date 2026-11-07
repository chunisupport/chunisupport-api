package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// AdminChartRankingResult は管理用の譜面単位ランキングの取得結果です。
// Total は上位件数に絞る前の全プレイ人数です。
type AdminChartRankingResult struct {
	Song    ChartRankingSong
	Chart   ChartRankingChart
	Ranking []ChartRankingEntry
	Total   int
}

type AdminChartRankingUsecase interface {
	GetStandard(ctx context.Context, displayID string, difficulty string) (*AdminChartRankingResult, error)
	GetWorldsend(ctx context.Context, displayID string) (*AdminChartRankingResult, error)
}

type adminChartRankingUsecase struct {
	rankingRepo repository.AdminChartRankingQueryService
}

func NewAdminChartRankingUsecase(rankingRepo repository.AdminChartRankingQueryService) AdminChartRankingUsecase {
	return &adminChartRankingUsecase{rankingRepo: rankingRepo}
}

func (u *adminChartRankingUsecase) GetStandard(ctx context.Context, displayID string, difficulty string) (*AdminChartRankingResult, error) {
	data, err := u.rankingRepo.GetStandard(ctx, displayID, difficulty, info.AdminChartRankingLimit)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, ErrChartNotFound
	}

	return buildAdminChartRankingResult(data), nil
}

func (u *adminChartRankingUsecase) GetWorldsend(ctx context.Context, displayID string) (*AdminChartRankingResult, error) {
	data, err := u.rankingRepo.GetWorldsend(ctx, displayID, info.AdminChartRankingLimit)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, ErrChartNotFound
	}

	return buildAdminChartRankingResult(data), nil
}

func buildAdminChartRankingResult(data *repository.AdminChartRankingData) *AdminChartRankingResult {
	result := &AdminChartRankingResult{
		Song:    newChartRankingSong(data.Chart),
		Chart:   newChartRankingChart(data.Chart),
		Ranking: make([]ChartRankingEntry, 0, len(data.Records)),
		Total:   data.Total,
	}

	var ranker scoreRanker
	for _, record := range data.Records {
		result.Ranking = append(result.Ranking, newChartRankingEntry(result.Chart, ranker.Next(record.Score), record))
	}

	return result
}
