package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

const (
	rankingComboLampNoneID       = 1
	rankingComboLampFullComboID  = 2
	rankingComboLampAllJusticeID = 3
)

// FriendChartRankingSong はランキング対象楽曲の概要です。
type FriendChartRankingSong struct {
	ID     string
	Title  string
	Artist string
}

// FriendChartRankingChart はランキング対象譜面の概要です。
type FriendChartRankingChart struct {
	Difficulty     string
	Const          chartconstant.ChartConstant
	IsConstUnknown bool
}

// FriendChartRankingEntry は譜面単位フレンドランキングの1件です。
type FriendChartRankingEntry struct {
	Rank             int
	UserID           int
	Username         string
	PlayerName       string
	Score            uint32
	Rating           float64
	Overpower        float64
	OverpowerPercent float64
	ClearLamp        *string
	ComboLamp        *string
	FullChain        *string
	UpdatedAt        time.Time
	IsSelf           bool
}

// FriendChartRankingResult は譜面単位フレンドランキングの取得結果です。
type FriendChartRankingResult struct {
	Song    FriendChartRankingSong
	Chart   FriendChartRankingChart
	Ranking []FriendChartRankingEntry
	MyRank  *int
	Total   int
}

// FriendChartRankingUsecase は譜面単位フレンドランキング取得を提供します。
type FriendChartRankingUsecase interface {
	GetStandard(ctx context.Context, userID int, displayID string, difficulty string) (*FriendChartRankingResult, error)
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

	result := &FriendChartRankingResult{
		Song: FriendChartRankingSong{
			ID:     chart.SongDisplayID,
			Title:  chart.SongTitle,
			Artist: chart.SongArtist,
		},
		Chart: FriendChartRankingChart{
			Difficulty:     chart.Difficulty,
			Const:          chart.Const,
			IsConstUnknown: chart.IsConstUnknown,
		},
		Ranking: make([]FriendChartRankingEntry, 0, len(rows)),
		Total:   len(rows),
	}

	previousScore := uint32(0)
	currentRank := 0
	for i, row := range rows {
		if i == 0 || row.Score != previousScore {
			currentRank = i + 1
			previousScore = row.Score
		}
		isSelf := row.UserID == userID
		if isSelf {
			rank := currentRank
			result.MyRank = &rank
		}
		result.Ranking = append(result.Ranking, FriendChartRankingEntry{
			Rank:             currentRank,
			UserID:           row.UserID,
			Username:         row.Username,
			PlayerName:       row.PlayerName,
			Score:            row.Score,
			Rating:           service.CalcSingleRating(row.Score, chart.Const.Float64()),
			Overpower:        service.CalcSingleOverpower(row.Score, chart.Const.Float64(), comboLampID(row.ComboLamp)),
			OverpowerPercent: service.CalcSingleOverpowerPercent(row.Score, chart.Const.Float64(), comboLampID(row.ComboLamp)),
			ClearLamp:        rankingLampNamePtr(row.ClearLamp),
			ComboLamp:        rankingLampNamePtr(row.ComboLamp),
			FullChain:        rankingLampNamePtr(row.FullChain),
			UpdatedAt:        row.UpdatedAt,
			IsSelf:           isSelf,
		})
	}

	return result, nil
}

func rankingLampNamePtr(name string) *string {
	if name == "" || name == "NONE" || name == "none" {
		return nil
	}
	return &name
}

func comboLampID(name string) int {
	switch name {
	case "FULL COMBO":
		return rankingComboLampFullComboID
	case "ALL JUSTICE":
		return rankingComboLampAllJusticeID
	default:
		return rankingComboLampNoneID
	}
}
