package api_internal

import (
	"encoding/json"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSingleChartBestSlotStatsResponse_レート帯IDを公開しない(t *testing.T) {
	// Given
	percentage := 25.0
	stats := &entity.SingleChartBestSlotStats{
		SongID: "0000000000000001",
		Stats: []*entity.ChartBestSlotStatsByRatingBand{{
			RatingBandID: 22, BestPlayerCount: 10, EligiblePlayerCount: 40, BestPlayerPercentage: &percentage,
		}},
	}

	// When
	response := ToSingleChartBestSlotStatsResponse(stats, []*ratingband.RatingBand{{ID: 22, Label: "17.0"}})
	encoded, err := json.Marshal(response)

	// Then
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"song_id":"0000000000000001",
		"stats":[{
			"rating_band":"17.0",
			"best_player_count":10,
			"eligible_player_count":40,
			"best_player_percentage":25
		}]
	}`, string(encoded))
	assert.NotContains(t, string(encoded), "rating_band_id")
}

func TestBestSlotStatsResponse_採用率を小数点以下4桁へ丸める(t *testing.T) {
	// Given
	percentage := 25.123456
	singleStats := &entity.SingleChartBestSlotStats{
		Stats: []*entity.ChartBestSlotStatsByRatingBand{{BestPlayerPercentage: &percentage}},
	}
	ranking := &usecase.BestSlotRankingResult{
		Ranking: []usecase.BestSlotRankingEntry{{BestPlayerPercentage: percentage}},
	}

	// When
	singleResponse := ToSingleChartBestSlotStatsResponse(singleStats, nil)
	rankingResponse := ToBestSlotRankingResponse(ranking, nil)

	// Then
	require.NotNil(t, singleResponse.Stats[0].BestPlayerPercentage)
	assert.Equal(t, 25.1235, *singleResponse.Stats[0].BestPlayerPercentage)
	assert.Equal(t, 25.1235, rankingResponse.Ranking[0].BestPlayerPercentage)
}
