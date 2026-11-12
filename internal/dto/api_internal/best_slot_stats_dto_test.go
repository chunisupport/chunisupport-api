package api_internal

import (
	"context"
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
	response := ToSingleChartBestSlotStatsResponse(context.Background(), stats, []*ratingband.RatingBand{{ID: 22, Label: "17.0"}})
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
	tests := []struct {
		name string
		// Given: 入力値
		input float64
		// Then: 期待する丸め結果
		expected float64
	}{
		{
			name:     "四捨五入で切り上がる値",
			input:    25.123456,
			expected: 25.1235,
		},
		{
			name:     "5桁目が4の場合は切り捨てる",
			input:    25.12344,
			expected: 25.1234,
		},
	}

	// 注意: 25.12345のような十進ちょうどの半端境界はfloat64で正確に表せず、
	// 本番のmath.Roundでは切り上がりを保証できないため本件では扱わない。

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			percentage := tt.input
			singleStats := &entity.SingleChartBestSlotStats{
				SongID: "0000000000000001",
				Stats: []*entity.ChartBestSlotStatsByRatingBand{{
					RatingBandID: 22, BestPlayerCount: 10, EligiblePlayerCount: 40, BestPlayerPercentage: &percentage,
				}},
			}
			ranking := &usecase.BestSlotRankingResult{
				RatingBand:          "17.0",
				EligiblePlayerCount: 40,
				Ranking: []usecase.BestSlotRankingEntry{{
					Rank: 1, SongID: "0000000000000001", Title: "楽曲名", Difficulty: "MASTER",
					BestPlayerCount: 10, BestPlayerPercentage: tt.input,
				}},
			}
			bands := []*ratingband.RatingBand{{ID: 22, Label: "17.0"}}

			// When
			singleResponse := ToSingleChartBestSlotStatsResponse(context.Background(), singleStats, bands)
			rankingResponse := ToBestSlotRankingResponse(ranking, nil)

			// Then
			require.Len(t, singleResponse.Stats, 1)
			require.Len(t, rankingResponse.Ranking, 1)
			require.NotNil(t, singleResponse.Stats[0].BestPlayerPercentage)
			assert.Equal(t, tt.expected, *singleResponse.Stats[0].BestPlayerPercentage)
			assert.Equal(t, tt.expected, rankingResponse.Ranking[0].BestPlayerPercentage)
		})
	}
}

func TestToSingleChartBestSlotStatsResponse_採用率のnullとゼロを区別する(t *testing.T) {
	// Given: 集計対象0人はnil、対象がいて採用0人はゼロをDTOへ直接渡す
	zero := 0.0
	stats := &entity.SingleChartBestSlotStats{
		SongID: "0000000000000001",
		Stats: []*entity.ChartBestSlotStatsByRatingBand{
			{RatingBandID: 22, BestPlayerCount: 0, EligiblePlayerCount: 0, BestPlayerPercentage: nil},
			{RatingBandID: 23, BestPlayerCount: 0, EligiblePlayerCount: 40, BestPlayerPercentage: &zero},
		},
	}
	bands := []*ratingband.RatingBand{{ID: 22, Label: "17.0"}, {ID: 23, Label: "17.1"}}

	// When
	response := ToSingleChartBestSlotStatsResponse(context.Background(), stats, bands)
	encoded, err := json.Marshal(response)

	// Then
	require.NoError(t, err)
	require.Len(t, response.Stats, 2)
	assert.Nil(t, response.Stats[0].BestPlayerPercentage)
	require.NotNil(t, response.Stats[1].BestPlayerPercentage)
	assert.Equal(t, 0.0, *response.Stats[1].BestPlayerPercentage)
	assert.JSONEq(t, `{
		"song_id":"0000000000000001",
		"stats":[{
			"rating_band":"17.0",
			"best_player_count":0,
			"eligible_player_count":0,
			"best_player_percentage":null
		},{
			"rating_band":"17.1",
			"best_player_count":0,
			"eligible_player_count":40,
			"best_player_percentage":0
		}]
	}`, string(encoded))
}

func TestToBestSlotRankingResponse_平均スコアをJSONへ変換する(t *testing.T) {
	// Given
	averageScore := 1007500.5
	ranking := &usecase.BestSlotRankingResult{
		Ranking: []usecase.BestSlotRankingEntry{
			{AverageScore: &averageScore},
			{AverageScore: nil},
		},
	}

	// When
	response := ToBestSlotRankingResponse(ranking, nil)
	encoded, err := json.Marshal(response.Ranking)

	// Then
	require.NoError(t, err)
	require.NotNil(t, response.Ranking[0].AverageScore)
	assert.Equal(t, averageScore, *response.Ranking[0].AverageScore)
	assert.Nil(t, response.Ranking[1].AverageScore)
	assert.Contains(t, string(encoded), `"average_score":1007500.5`)
	assert.Contains(t, string(encoded), `"average_score":null`)
}
