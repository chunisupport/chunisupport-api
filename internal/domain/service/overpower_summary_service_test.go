package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcOverpowerSummary_楽曲ごとの最大値を合計する(t *testing.T) {
	// 手計算の根拠 (docs/overpower_calculation.md の1・2・3・6章)。
	// コンボランプIDは 3=ALL JUSTICE(+1.0)、2=FULL COMBO(+0.5)。S以上は0.005刻み切り捨て。
	// 楽曲1の高記録 (1008000/15.0/AJ): (15.0+2)×5 + (1008000-1007500)÷2500×3.75 + 1.0 = 85.0 + 0.75 + 1.0 = 86.75。
	// 楽曲1の低記録 (1007000/15.0/FC): SS+区間のため (15.0+1.5)×5 + (1007000-1005000)÷2500×2.5 + 0.5 = 82.5 + 2.0 + 0.5 = 85.0。
	// 楽曲2 (1009000/14.0/AJ): (14.0+2)×5 + (1009000-1007500)÷2500×3.75 + 1.0 = 80.0 + 2.25 + 1.0 = 83.25。
	// 楽曲ごとの最大値の合計は 86.75 + 83.25 = 170.0。分母は楽曲最大OPの合計で (15.0+3)×5 + (14.0+3)×5 = 90.0 + 85.0 = 175.0。
	// 割合は 170.0÷175.0×100 = 97.142857... を小数第5位で切り捨てて 97.14285。
	tests := []struct {
		name    string
		records []OverpowerRecord
	}{
		{
			name: "高い記録が先でも楽曲の最大値を採用する",
			records: []OverpowerRecord{
				{SongID: 1, Score: 1008000, ChartConst: 15.0, ComboLampID: 3},
				{SongID: 1, Score: 1007000, ChartConst: 15.0, ComboLampID: 2},
				{SongID: 2, Score: 1009000, ChartConst: 14.0, ComboLampID: 3},
			},
		},
		{
			name: "低い記録が先でも楽曲の最大値を採用する",
			records: []OverpowerRecord{
				{SongID: 1, Score: 1007000, ChartConst: 15.0, ComboLampID: 2},
				{SongID: 1, Score: 1008000, ChartConst: 15.0, ComboLampID: 3},
				{SongID: 2, Score: 1009000, ChartConst: 14.0, ComboLampID: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			value, percent := CalcOverpowerSummary(tt.records, 175.0)

			// Then
			assert.Equal(t, 170.0, value)
			assert.Equal(t, 97.14285, percent)
		})
	}
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
			name:              "小数点以下5桁で切り捨てる",
			overpowerValue:    2,
			maxOverpowerTotal: 3,
			expected:          66.66666,
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
