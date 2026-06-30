package service

import "math"

// OverpowerRecord はOVER POWER集計に必要な単曲情報です。
type OverpowerRecord struct {
	SongID      int
	Score       uint32
	ChartConst  float64
	ComboLampID int
}

// CalcOverpowerSummary は楽曲ごとの最高OVER POWERを合算し、値と割合を返します。
func CalcOverpowerSummary(records []OverpowerRecord, maxOverpowerTotal float64) (float64, float64) {
	bestBySongID := make(map[int]int64, len(records))
	for _, record := range records {
		overpower := calcSingleOverpowerThousandths(record.Score, record.ChartConst, record.ComboLampID)
		if best, exists := bestBySongID[record.SongID]; !exists || overpower > best {
			bestBySongID[record.SongID] = overpower
		}
	}

	var totalOverpower int64
	for _, overpower := range bestBySongID {
		totalOverpower += overpower
	}

	value := float64(max(totalOverpower, 0)) / float64(overpowerScale)
	percent := calcOverpowerPercent(
		totalOverpower,
		int64(math.Round(maxOverpowerTotal*float64(overpowerScale))),
	)

	return value, percent
}

// CalcOverpowerPercent は保存済みOVER POWER値と現在の最大OP合計から達成割合を計算します。
func CalcOverpowerPercent(overpowerValue float64, maxOverpowerTotal float64) float64 {
	return calcOverpowerPercent(
		int64(math.Round(overpowerValue*float64(overpowerScale))),
		int64(math.Round(maxOverpowerTotal*float64(overpowerScale))),
	)
}

func calcOverpowerPercent(overpowerValue, maxOverpowerTotal int64) float64 {
	if maxOverpowerTotal <= 0 {
		return 0.0
	}

	scaled := min(max(overpowerValue*100*percentScale/maxOverpowerTotal, 0), 100*percentScale)
	return float64(scaled) / float64(percentScale)
}
