package chartstatsbatch

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
)

func TestCourseStatsCalculator_クリア状態とランプを分類しスコアを集計する(t *testing.T) {
	tests := []struct {
		name string
		// Given
		records []CourseRecord
		// Then
		expected entity.CourseStatsByRatingBand
	}{
		{
			name:    "クリア済みの記録は人数とスコア統計の両方に含める",
			records: []CourseRecord{{CourseID: 42, Score: 2_900_000, IsClear: true, ComboLampID: comboLampIDNone}},
			expected: entity.CourseStatsByRatingBand{
				CourseID: 42, PlayerCount: 1,
				Combo:        entity.CourseComboStats{None: 1},
				Clear:        entity.CourseClearStats{Clear: 1},
				AverageScore: new(float64(2_900_000)), MedianScore: new(float64(2_900_000)),
			},
		},
		{
			name: "未クリアの記録は人数とランプに含めスコア統計から除外する",
			records: []CourseRecord{
				{CourseID: 42, Score: 1_000_000, IsClear: false, ComboLampID: comboLampIDNone},
				{CourseID: 42, Score: 3_000_000, IsClear: true, ComboLampID: comboLampIDFullCombo},
			},
			expected: entity.CourseStatsByRatingBand{
				CourseID: 42, PlayerCount: 2,
				Combo:        entity.CourseComboStats{None: 1, FC: 1},
				Clear:        entity.CourseClearStats{Failed: 1, Clear: 1},
				AverageScore: new(float64(3_000_000)), MedianScore: new(float64(3_000_000)),
			},
		},
		{
			name:    "すべて未クリアの場合は平均と中央値をnilにする",
			records: []CourseRecord{{CourseID: 42, Score: 1_000_000, IsClear: false, ComboLampID: comboLampIDNone}},
			expected: entity.CourseStatsByRatingBand{
				CourseID: 42, PlayerCount: 1,
				Combo: entity.CourseComboStats{None: 1},
				Clear: entity.CourseClearStats{Failed: 1},
			},
		},
		{
			name: "コンボランプを排他的に分類し理論値もAJに数える",
			records: []CourseRecord{
				{CourseID: 42, Score: 2_800_000, IsClear: true, ComboLampID: comboLampIDNone},
				{CourseID: 42, Score: 2_900_000, IsClear: true, ComboLampID: comboLampIDFullCombo},
				{CourseID: 42, Score: 3_020_000, IsClear: true, ComboLampID: comboLampIDAllJustice},
				{CourseID: 42, Score: 3_030_000, IsClear: true, ComboLampID: comboLampIDAllJustice},
			},
			expected: entity.CourseStatsByRatingBand{
				CourseID: 42, PlayerCount: 4,
				Combo:        entity.CourseComboStats{None: 1, FC: 1, AJ: 2},
				Clear:        entity.CourseClearStats{Clear: 4},
				AverageScore: new(float64(2_800_000+2_900_000+3_020_000+3_030_000) / 4),
				MedianScore:  new(float64(2_900_000+3_020_000) / 2),
			},
		},
		{
			name: "奇数件の中央値は昇順に並べた中央の値にする",
			records: []CourseRecord{
				{CourseID: 42, Score: 3_000_000, IsClear: true, ComboLampID: comboLampIDNone},
				{CourseID: 42, Score: 2_000_000, IsClear: true, ComboLampID: comboLampIDNone},
				{CourseID: 42, Score: 2_700_000, IsClear: true, ComboLampID: comboLampIDNone},
			},
			expected: entity.CourseStatsByRatingBand{
				CourseID: 42, PlayerCount: 3,
				Combo:        entity.CourseComboStats{None: 3},
				Clear:        entity.CourseClearStats{Clear: 3},
				AverageScore: new(float64(7_700_000) / 3), MedianScore: new(float64(2_700_000)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			calculator := NewCourseStatsCalculator([]*ratingband.RatingBand{{ID: 0}})

			// When
			for _, record := range tt.records {
				calculator.Add(record)
			}
			results := calculator.Results()

			// Then
			assert.Equal(t, []*entity.CourseStatsByRatingBand{&tt.expected}, results)
			assert.Equal(t, len(tt.records), calculator.RecordCount())
		})
	}
}

func TestCourseStatsCalculator_レーティング帯とコースごとに集計する(t *testing.T) {
	// Given: ALL と、下限包含・上限除外の2つの帯
	lower, boundary, upper := 16.0, 17.0, 18.0
	calculator := NewCourseStatsCalculator([]*ratingband.RatingBand{
		{ID: 0},
		{ID: 1, MinInclusive: &lower, MaxExclusive: &boundary},
		{ID: 2, MinInclusive: &boundary, MaxExclusive: &upper},
	})

	// When: コースID順に記録を渡す。コース8には帯1のプレイヤーがいない。
	for _, record := range []CourseRecord{
		{CourseID: 7, BestAverageRating: 16.99, Score: 2_800_000, IsClear: true, ComboLampID: comboLampIDNone},
		{CourseID: 7, BestAverageRating: 17.0, Score: 3_000_000, IsClear: true, ComboLampID: comboLampIDFullCombo},
		{CourseID: 8, BestAverageRating: 17.5, Score: 1_500_000, IsClear: false, ComboLampID: comboLampIDNone},
	} {
		calculator.Add(record)
	}
	results := calculator.Results()

	// Then: コースの出現順、同じコース内は帯の順に並び、記録のない帯の行は作らない。
	assert.Equal(t, []*entity.CourseStatsByRatingBand{
		{
			CourseID: 7, RatingBandID: 0, PlayerCount: 2,
			Combo: entity.CourseComboStats{None: 1, FC: 1}, Clear: entity.CourseClearStats{Clear: 2},
			AverageScore: new(float64(2_900_000)), MedianScore: new(float64(2_900_000)),
		},
		{
			CourseID: 7, RatingBandID: 1, PlayerCount: 1,
			Combo: entity.CourseComboStats{None: 1}, Clear: entity.CourseClearStats{Clear: 1},
			AverageScore: new(float64(2_800_000)), MedianScore: new(float64(2_800_000)),
		},
		{
			CourseID: 7, RatingBandID: 2, PlayerCount: 1,
			Combo: entity.CourseComboStats{FC: 1}, Clear: entity.CourseClearStats{Clear: 1},
			AverageScore: new(float64(3_000_000)), MedianScore: new(float64(3_000_000)),
		},
		{CourseID: 8, RatingBandID: 0, PlayerCount: 1, Combo: entity.CourseComboStats{None: 1}, Clear: entity.CourseClearStats{Failed: 1}},
		{CourseID: 8, RatingBandID: 2, PlayerCount: 1, Combo: entity.CourseComboStats{None: 1}, Clear: entity.CourseClearStats{Failed: 1}},
	}, results)
	assert.Equal(t, 3, calculator.RecordCount())
}
