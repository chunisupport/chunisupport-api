package dto

import (
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
