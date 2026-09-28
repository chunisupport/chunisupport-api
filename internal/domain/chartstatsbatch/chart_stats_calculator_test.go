package chartstatsbatch

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func float64Ptr(value float64) *float64 { return &value }

func TestChartStatsCalculator_ランプとランクを分類する(t *testing.T) {
	tests := []struct {
		name     string
		records  []ChartRecord
		expected entity.ChartStatsByRatingBand
	}{
		{
			name:    "AJかつ理論値はAJCとして数えAJには含めない",
			records: []ChartRecord{{ChartID: 42, Score: scoreMax, ClearLampID: clearLampIDClear, ComboLampID: comboLampIDAllJustice}},
			expected: entity.ChartStatsByRatingBand{
				ChartID: 42, PlayerCount: 1,
				Rank:         entity.ChartRankStats{Max: 1},
				Combo:        entity.ChartComboStats{AJC: 1},
				Clear:        entity.ChartClearStats{Clear: 1},
				AverageScore: float64Ptr(scoreMax), MedianScore: float64Ptr(scoreMax),
			},
		},
		{
			name:    "AJでない理論値はAJCに含めない",
			records: []ChartRecord{{ChartID: 42, Score: scoreMax, ClearLampID: clearLampIDClear, ComboLampID: comboLampIDFullCombo}},
			expected: entity.ChartStatsByRatingBand{
				ChartID: 42, PlayerCount: 1,
				Rank:         entity.ChartRankStats{Max: 1},
				Combo:        entity.ChartComboStats{FC: 1},
				Clear:        entity.ChartClearStats{Clear: 1},
				AverageScore: float64Ptr(scoreMax), MedianScore: float64Ptr(scoreMax),
			},
		},
		{
			name: "未クリアの記録は人数とランプに含めスコア統計から除外する",
			records: []ChartRecord{
				{ChartID: 42, Score: 100, ClearLampID: clearLampIDFailed, ComboLampID: comboLampIDNone},
				{ChartID: 42, Score: scoreSS, ClearLampID: clearLampIDClear, ComboLampID: comboLampIDFullCombo},
			},
			expected: entity.ChartStatsByRatingBand{
				ChartID: 42, PlayerCount: 2,
				Rank:         entity.ChartRankStats{SS: 1},
				Combo:        entity.ChartComboStats{None: 1, FC: 1},
				Clear:        entity.ChartClearStats{Failed: 1, Clear: 1},
				AverageScore: float64Ptr(scoreSS), MedianScore: float64Ptr(scoreSS),
			},
		},
		{
			name:    "すべて未クリアの場合は平均と中央値をnilにする",
			records: []ChartRecord{{ChartID: 42, Score: 100, ClearLampID: clearLampIDFailed, ComboLampID: comboLampIDNone}},
			expected: entity.ChartStatsByRatingBand{
				ChartID: 42, PlayerCount: 1,
				Combo: entity.ChartComboStats{None: 1},
				Clear: entity.ChartClearStats{Failed: 1},
			},
		},
		{
			name: "クリアランプとランクを境界値で排他的に分類する",
			records: []ChartRecord{
				{ChartID: 42, Score: scoreSSSPlus, ClearLampID: clearLampIDHard, ComboLampID: comboLampIDNone},
				{ChartID: 42, Score: scoreSSS, ClearLampID: clearLampIDBrave, ComboLampID: comboLampIDNone},
				{ChartID: 42, Score: scoreSSPlus, ClearLampID: clearLampIDAbsolute, ComboLampID: comboLampIDNone},
				{ChartID: 42, Score: scoreSPlus, ClearLampID: clearLampIDCatastrophy, ComboLampID: comboLampIDNone},
				{ChartID: 42, Score: scoreS, ClearLampID: clearLampIDClear, ComboLampID: comboLampIDNone},
				{ChartID: 42, Score: scoreS - 1, ClearLampID: clearLampIDClear, ComboLampID: comboLampIDNone},
			},
			expected: entity.ChartStatsByRatingBand{
				ChartID: 42, PlayerCount: 6,
				Rank:         entity.ChartRankStats{SSSP: 1, SSS: 1, SSP: 1, SP: 1, S: 1, AAAL: 1},
				Combo:        entity.ChartComboStats{None: 6},
				Clear:        entity.ChartClearStats{Clear: 2, Hard: 1, Brave: 1, Absolute: 1, Catastrophy: 1},
				AverageScore: float64Ptr(float64(scoreSSSPlus+scoreSSS+scoreSSPlus+scoreSPlus+scoreS+scoreS-1) / 6),
				MedianScore:  float64Ptr(float64(scoreSSPlus+scoreSPlus) / 2),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			calc := NewChartStatsCalculator([]*ratingband.RatingBand{{ID: 0}})

			// When
			for _, record := range tt.records {
				calc.Add(record)
			}
			results := calc.Results()

			// Then
			require.Len(t, results, 1)
			assert.Equal(t, tt.expected, *results[0])
		})
	}
}

func TestChartStatsCalculator_中央値(t *testing.T) {
	tests := []struct {
		name     string
		scores   []int
		expected float64
	}{
		{name: "1件の場合はその値", scores: []int{980000}, expected: 980000},
		{name: "奇数件の場合は並べ替えた中央の値", scores: []int{990000, 980000, 985000}, expected: 985000},
		{name: "偶数件の場合は中央2件の平均", scores: []int{990000, 980001}, expected: 985000.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			calc := NewChartStatsCalculator([]*ratingband.RatingBand{{ID: 0}})
			for _, score := range tt.scores {
				calc.Add(ChartRecord{ChartID: 1, Score: score, ClearLampID: clearLampIDClear, ComboLampID: comboLampIDNone})
			}

			// When
			results := calc.Results()

			// Then
			require.Len(t, results, 1)
			require.NotNil(t, results[0].MedianScore)
			assert.Equal(t, tt.expected, *results[0].MedianScore)
		})
	}
}

func TestChartStatsCalculator_譜面とレーティング帯ごとに集計する(t *testing.T) {
	// Given: ALL と 17.00～17.10 の帯
	lower, upper := 17.0, 17.1
	calc := NewChartStatsCalculator([]*ratingband.RatingBand{
		{ID: 0},
		{ID: 22, MinInclusive: &lower, MaxExclusive: &upper},
	})
	records := []ChartRecord{
		{ChartID: 1, BestAverageRating: 17.05, Score: scoreSS, ClearLampID: clearLampIDClear, ComboLampID: comboLampIDNone},
		{ChartID: 1, BestAverageRating: 16.5, Score: scoreS, ClearLampID: clearLampIDClear, ComboLampID: comboLampIDNone},
		{ChartID: 2, BestAverageRating: 16.5, Score: scoreS, ClearLampID: clearLampIDClear, ComboLampID: comboLampIDNone},
	}

	// When
	for _, record := range records {
		calc.Add(record)
	}
	results := calc.Results()

	// Then: 記録のない帯の行は作らない
	require.Len(t, results, 3)
	assert.Equal(t, [3]int{1, 0, 2}, [3]int{results[0].ChartID, results[0].RatingBandID, results[0].PlayerCount})
	assert.Equal(t, [3]int{1, 22, 1}, [3]int{results[1].ChartID, results[1].RatingBandID, results[1].PlayerCount})
	assert.Equal(t, [3]int{2, 0, 1}, [3]int{results[2].ChartID, results[2].RatingBandID, results[2].PlayerCount})
	assert.Equal(t, 3, calc.RecordCount())
}

func TestChartStatsCalculator_記録がない場合は空(t *testing.T) {
	// Given
	calc := NewChartStatsCalculator([]*ratingband.RatingBand{{ID: 0}})

	// When
	results := calc.Results()

	// Then
	assert.Empty(t, results)
	assert.Zero(t, calc.RecordCount())
}
