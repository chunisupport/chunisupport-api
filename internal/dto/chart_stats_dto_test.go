package dto

import (
	"encoding/json"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSingleChartStatsResponse_AJとAJCを排他的な件数として変換する(t *testing.T) {
	// Given
	stats := &entity.SingleChartStats{
		SongID: "0123456789abcdef",
		Stats: []*entity.ChartStatsByRatingBand{
			{
				RatingBandID: 0,
				Combo: entity.ChartComboStats{
					AJ:  12,
					AJC: 3,
				},
			},
		},
	}
	ratingBands := []*ratingband.RatingBand{{ID: 0, Label: "ALL"}}

	// When
	result := ToSingleChartStatsResponse(stats, ratingBands)

	// Then
	require.NotNil(t, result)
	require.Len(t, result.Stats, 1)
	assert.Equal(t, 12, result.Stats[0].Combo.AJ)
	assert.Equal(t, 3, result.Stats[0].Combo.AJC)
}

func TestToSingleChartStatsResponse_中央値を変換する(t *testing.T) {
	// Given
	medianScore := 1007123.5
	stats := &entity.SingleChartStats{
		SongID: "0123456789abcdef",
		Stats: []*entity.ChartStatsByRatingBand{
			{RatingBandID: 0, MedianScore: &medianScore},
			{RatingBandID: 1},
		},
	}
	ratingBands := []*ratingband.RatingBand{{ID: 0, Label: "ALL"}, {ID: 1, Label: "15.0"}}

	// When
	result := ToSingleChartStatsResponse(stats, ratingBands)

	// Then
	require.NotNil(t, result)
	require.Len(t, result.Stats, 2)
	assert.Equal(t, &medianScore, result.Stats[0].MedianScore)
	assert.Nil(t, result.Stats[1].MedianScore)
}

func TestToChartStatsResponse_中央値とNULLをJSONへ変換する(t *testing.T) {
	// Given
	medianScore := 1007123.5
	stats := &entity.SongChartStats{
		SongID: "0123456789abcdef",
		Charts: map[string][]*entity.ChartStatsByRatingBand{
			"MASTER": {
				{RatingBandID: 0, MedianScore: &medianScore},
				{RatingBandID: 1},
			},
		},
	}
	ratingBands := []*ratingband.RatingBand{{ID: 0, Label: "ALL"}, {ID: 1, Label: "15.0"}}

	// When
	result := ToChartStatsResponse(stats, ratingBands)
	encoded, err := json.Marshal(result)

	// Then
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Charts["MASTER"].Stats, 2)
	assert.Equal(t, &medianScore, result.Charts["MASTER"].Stats[0].MedianScore)
	assert.Nil(t, result.Charts["MASTER"].Stats[1].MedianScore)
	assert.Contains(t, string(encoded), `"median_score":null`)
}
