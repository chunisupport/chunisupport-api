package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcOverpowerSummary_楽曲ごとの最大値を合計する(t *testing.T) {
	records := []OverpowerRecord{
		{SongID: 1, Score: 1008000, ChartConst: 15.0, ComboLampID: 3},
		{SongID: 1, Score: 1007000, ChartConst: 15.0, ComboLampID: 2},
		{SongID: 2, Score: 1009000, ChartConst: 14.0, ComboLampID: 3},
	}
	maxTotal := CalcSongMaxOP(15.0) + CalcSongMaxOP(14.0)

	value, percent := CalcOverpowerSummary(records, maxTotal)

	wantValue := CalcSingleOverpower(1008000, 15.0, 3) + CalcSingleOverpower(1009000, 14.0, 3)
	wantPercent := roundToScale(wantValue/maxTotal*100, 4)
	assert.InDelta(t, wantValue, value, 0.0001)
	assert.InDelta(t, wantPercent, percent, 0.0001)
}

func TestCalcOverpowerPercent(t *testing.T) {
	tests := []struct {
		name              string
		overpowerValue    float64
		maxOverpowerTotal float64
		expected          float64
	}{
		{
			name:              "分母が0の場合は0",
			overpowerValue:    10,
			maxOverpowerTotal: 0,
			expected:          0,
		},
		{
			name:              "分母が負の場合は0",
			overpowerValue:    10,
			maxOverpowerTotal: -1,
			expected:          0,
		},
		{
			name:              "小数第4位に丸める",
			overpowerValue:    1,
			maxOverpowerTotal: 3,
			expected:          33.3333,
		},
		{
			name:              "100を超える場合は100に丸める",
			overpowerValue:    120,
			maxOverpowerTotal: 100,
			expected:          100,
		},
		{
			name:              "負値は0に丸める",
			overpowerValue:    -10,
			maxOverpowerTotal: 100,
			expected:          0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := CalcOverpowerPercent(tt.overpowerValue, tt.maxOverpowerTotal)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
