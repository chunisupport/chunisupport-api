package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// FriendChartRankingEntry は譜面単位フレンドランキングの1件です。
// 共通項目に加え、ユーザーIDと自分かどうかを持ちます。
type FriendChartRankingEntry struct {
	ChartRankingEntry
	UserID int
	IsSelf bool
}

// FriendChartRankingResult は譜面単位フレンドランキングの取得結果です。
type FriendChartRankingResult struct {
	Song    ChartRankingSong
	Chart   ChartRankingChart
	Ranking []FriendChartRankingEntry
	MyRank  *int
	Total   int
}

// FriendChartRankingUsecase は譜面単位フレンドランキング取得を提供します。
type FriendChartRankingUsecase interface {
	GetStandard(ctx context.Context, userID int, displayID string, difficulty string) (*FriendChartRankingResult, error)
	GetWorldsend(ctx context.Context, userID int, displayID string) (*FriendChartRankingResult, error)
}

type friendChartRankingUsecase struct {
	exec        repository.Executor
	rankingRepo repository.FriendChartRankingQueryService
}

func NewFriendChartRankingUsecase(exec repository.Executor, rankingRepo repository.FriendChartRankingQueryService) FriendChartRankingUsecase {
	return &friendChartRankingUsecase{exec: exec, rankingRepo: rankingRepo}
}

func (u *friendChartRankingUsecase) GetStandard(ctx context.Context, userID int, displayID string, difficulty string) (*FriendChartRankingResult, error) {
	chart, err := u.rankingRepo.FindChart(ctx, u.exec, displayID, difficulty)
	if err != nil {
		return nil, err
	}
	if chart == nil {
		return nil, ErrChartNotFound
	}

	rows, err := u.rankingRepo.ListRecords(ctx, u.exec, userID, chart.ChartID)
	if err != nil {
		return nil, err
	}

	return buildFriendChartRankingResult(userID, chart, rows), nil
}

func (u *friendChartRankingUsecase) GetWorldsend(ctx context.Context, userID int, displayID string) (*FriendChartRankingResult, error) {
	chart, err := u.rankingRepo.FindWorldsendChart(ctx, u.exec, displayID)
	if err != nil {
		return nil, err
	}
	if chart == nil {
		return nil, ErrChartNotFound
	}

	rows, err := u.rankingRepo.ListWorldsendRecords(ctx, u.exec, userID, chart.ChartID)
	if err != nil {
		return nil, err
	}

	return buildFriendChartRankingResult(userID, chart, rows), nil
}

func buildFriendChartRankingResult(userID int, chart *repository.FriendChartRankingChart, rows []*repository.FriendChartRankingRecord) *FriendChartRankingResult {
	result := &FriendChartRankingResult{
		Song:    newChartRankingSong(&chart.ChartRankingChart),
		Chart:   newChartRankingChart(&chart.ChartRankingChart),
		Ranking: make([]FriendChartRankingEntry, 0, len(rows)),
		Total:   len(rows),
	}

	var ranker scoreRanker
	for _, row := range rows {
		entry := FriendChartRankingEntry{
			ChartRankingEntry: newChartRankingEntry(result.Chart, ranker.Next(row.Score), &row.ChartRankingRecord),
			UserID:            row.UserID,
			IsSelf:            row.UserID == userID,
		}
		if entry.IsSelf {
			rank := entry.Rank
			result.MyRank = &rank
		}
		result.Ranking = append(result.Ranking, entry)
	}

	return result
}
