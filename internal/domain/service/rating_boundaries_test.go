package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcSingleOverpower_区間と切り捨て単位の境界(t *testing.T) {
	// 補正なし・定数15での仕様値。期待値は計算関数から生成しない。
	for _, tt := range []struct {
		name  string
		score uint32
		want  float64
	}{
		{"SSS直前", 1_007_499, 84.995}, {"SSS一致", 1_007_500, 85},
		{"SSSの傾きと丸め", 1_007_504, 85.005},
		{"SSプラス直前", 1_004_999, 82.495}, {"SSプラス一致", 1_005_000, 82.5},
		{"SSプラスの最小増分直前", 1_005_004, 82.5}, {"SSプラスの最小増分一致", 1_005_005, 82.505},
		{"SS直前", 999_999, 79.995}, {"SS一致", 1_000_000, 80},
		{"SSの最小増分直前", 1_000_009, 80}, {"SSの最小増分一致", 1_000_010, 80.005},
		{"S直前は0.05単位", 974_999, 74.95}, {"S一致は0.005単位", 975_000, 75},
		{"Sの最小増分直前", 975_024, 75}, {"Sの最小増分一致", 975_025, 75.005},
		{"A直前", 899_999, 49.95}, {"A一致", 900_000, 50},
		{"Aの最小増分直前", 900_149, 50}, {"Aの最小増分一致", 900_150, 50.05},
		{"BBB直前", 799_999, 24.95}, {"BBB一致", 800_000, 25},
		{"BBBの最小増分直前", 800_199, 25}, {"BBBの最小増分一致", 800_200, 25.05},
		{"C直前", 499_999, 0}, {"C一致", 500_000, 0},
		{"Cの最小増分直前", 500_599, 0}, {"Cの最小増分一致", 500_600, 0.05},
	} {
		t.Run(tt.name, func(t *testing.T) { assert.Equal(t, tt.want, CalcSingleOverpower(tt.score, 15, 1)) })
	}
}

func TestAggregateOfficialRating_本枠平均を小数第4位で切り捨てる(t *testing.T) {
	got := AggregateOfficialRating(
		[]RatingSlotRecord{{Score: 1_009_000, ChartConst: 15}, {Score: 1_009_000, ChartConst: 15}, {Score: 1_009_000, ChartConst: 15.1}},
		[]RatingSlotRecord{{Score: 1_009_000, ChartConst: 14}, {Score: 1_009_000, ChartConst: 14.1}, {Score: 1_009_000, ChartConst: 14.1}},
	)
	assert.Equal(t, RatingStats{BestAverage: 17.1833, NewAverage: 16.2166, PlayerRating: 2.004}, got)
}
