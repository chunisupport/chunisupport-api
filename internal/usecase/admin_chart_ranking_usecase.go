package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

type AdminChartRankingSong struct {
	ID     string
	Title  string
	Artist string
}

type AdminChartRankingChart struct {
	Difficulty     string
	Const          chartconstant.ChartConstant
	IsConstUnknown bool
	LevelStar      *int
	Attribute      *string
	IsWorldsend    bool
}

type AdminChartRankingEntry struct {
	Rank             int
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
}

type AdminChartRankingResult struct {
	Song    AdminChartRankingSong
	Chart   AdminChartRankingChart
	Ranking []AdminChartRankingEntry
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
	chart := data.Chart
	result := &AdminChartRankingResult{
		Song: AdminChartRankingSong{
			ID:     chart.SongDisplayID,
			Title:  chart.SongTitle,
			Artist: chart.SongArtist,
		},
		Chart: AdminChartRankingChart{
			Difficulty:     chart.Difficulty,
			Const:          chart.Const,
			IsConstUnknown: chart.IsConstUnknown,
			LevelStar:      chart.LevelStar,
			Attribute:      chart.Attribute,
			IsWorldsend:    chart.IsWorldsend,
		},
		Ranking: make([]AdminChartRankingEntry, 0, len(data.Records)),
		Total:   data.Total,
	}

	previousScore := uint32(0)
	currentRank := 0
	for i, row := range data.Records {
		if i == 0 || row.Score != previousScore {
			currentRank = i + 1
			previousScore = row.Score
		}

		rating := 0.0
		overpower := 0.0
		overpowerPercent := 0.0
		if !chart.IsWorldsend {
			rating = service.CalcSingleRating(row.Score, chart.Const.Float64())
			overpower = service.CalcSingleOverpower(row.Score, chart.Const.Float64(), comboLampID(row.ComboLamp))
			overpowerPercent = service.CalcSingleOverpowerPercent(row.Score, chart.Const.Float64(), comboLampID(row.ComboLamp))
		}

		result.Ranking = append(result.Ranking, AdminChartRankingEntry{
			Rank:             currentRank,
			Username:         row.Username,
			PlayerName:       row.PlayerName,
			Score:            row.Score,
			Rating:           rating,
			Overpower:        overpower,
			OverpowerPercent: overpowerPercent,
			ClearLamp:        rankingLampNamePtr(row.ClearLamp),
			ComboLamp:        rankingLampNamePtr(row.ComboLamp),
			FullChain:        rankingLampNamePtr(row.FullChain),
			UpdatedAt:        row.UpdatedAt,
		})
	}

	return result
}
