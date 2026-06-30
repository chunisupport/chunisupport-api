package service

import (
	"math"
)

// OverpowerRecord はOVER POWER集計に必要な単曲情報です。
type OverpowerRecord struct {
	SongID      int
	Score       uint32
	ChartConst  float64
	ComboLampID int
}

// CalcOverpowerSummary は楽曲ごとの最高OVER POWERを合算し、値と割合を返します。
func CalcOverpowerSummary(records []OverpowerRecord, maxOverpowerTotal float64) (float64, float64) {
	bestBySongID := make(map[int]float64, len(records))
	for _, record := range records {
		overpower := CalcSingleOverpower(record.Score, record.ChartConst, record.ComboLampID)
		if best, exists := bestBySongID[record.SongID]; !exists || overpower > best {
			bestBySongID[record.SongID] = overpower
		}
	}

	totalOverpower := 0.0
	for _, overpower := range bestBySongID {
		totalOverpower += overpower
	}

	value := max(roundToScale(totalOverpower, 3), 0.0)
	percent := CalcOverpowerPercent(totalOverpower, maxOverpowerTotal)

	return value, percent
}

// CalcOverpowerPercent は保存済みOVER POWER値と現在の最大OP合計から達成割合を計算します。
func CalcOverpowerPercent(overpowerValue float64, maxOverpowerTotal float64) float64 {
	if maxOverpowerTotal <= 0 {
		return 0.0
	}

	return min(max(truncN(overpowerValue/maxOverpowerTotal*100, 4), 0.0), 100.0)
}

func roundToScale(value float64, scale int) float64 {
	factor := math.Pow10(scale)
	return math.Round(value*factor) / factor
}
