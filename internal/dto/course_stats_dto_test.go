package dto

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToCourseStatsResponse(t *testing.T) {
	bands := []*ratingband.RatingBand{{ID: 0, Label: "ALL"}, {ID: 22, Label: "17.0"}}
	average, median := 2950000.5, 2900000.0
	tests := []struct {
		name string
		// Given
		stats *entity.CourseStats
		// Then
		wantJSON string
	}{
		{
			name: "先頭要素をALLとしてコース用の統計形式へ変換する",
			stats: &entity.CourseStats{CourseDisplayID: "0123456789abcdef", Stats: []*entity.CourseStatsByRatingBand{
				{
					CourseID: 501, RatingBandID: 0, PlayerCount: 9,
					Combo: entity.CourseComboStats{None: 4, FC: 3, AJ: 2}, Clear: entity.CourseClearStats{Failed: 1, Clear: 8},
					AverageScore: &average, MedianScore: &median,
				},
				{CourseID: 501, RatingBandID: 22, PlayerCount: 1, Combo: entity.CourseComboStats{None: 1}, Clear: entity.CourseClearStats{Failed: 1}},
			}},
			wantJSON: `{"course_id":"0123456789abcdef","stats":[
				{"rating_band":"ALL","combo":{"none":4,"fc":3,"aj":2},"clear":{"failed":1,"clear":8},"average_score":2950000.5,"median_score":2900000,"player_count":9},
				{"rating_band":"17.0","combo":{"none":1,"fc":0,"aj":0},"clear":{"failed":1,"clear":0},"average_score":null,"median_score":null,"player_count":1}
			]}`,
		},
		{
			name:     "未集計は空配列で返す",
			stats:    &entity.CourseStats{CourseDisplayID: "0123456789abcdef"},
			wantJSON: `{"course_id":"0123456789abcdef","stats":[]}`,
		},
		{
			name: "マスタにないレーティング帯はIDを文字列で返す",
			stats: &entity.CourseStats{CourseDisplayID: "0123456789abcdef", Stats: []*entity.CourseStatsByRatingBand{
				{CourseID: 501, RatingBandID: 99},
			}},
			wantJSON: `{"course_id":"0123456789abcdef","stats":[
				{"rating_band":"99","combo":{"none":0,"fc":0,"aj":0},"clear":{"failed":0,"clear":0},"average_score":null,"median_score":null,"player_count":0}
			]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			encoded, err := json.Marshal(ToCourseStatsResponse(context.Background(), tt.stats, bands))

			// Then
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(encoded))
		})
	}
}
