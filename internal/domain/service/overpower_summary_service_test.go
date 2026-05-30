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
