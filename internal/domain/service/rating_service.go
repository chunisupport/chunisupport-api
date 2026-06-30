// Package service はドメイン層のサービスを提供します。
package service

import (
	"cmp"
	"math"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
)

const (
	ratingScale          = int64(100)
	aggregateRatingScale = int64(10_000)
	overpowerScale       = int64(1_000)
	percentScale         = int64(10_000)
)

// CalcSingleRating は指定されたスコアと譜面定数から単曲レーティングを計算します。
// スコアが低い場合は0を返し、SSS+（1,009,000点）で譜面定数+2.15が上限となります。
//
// 計算式:
//
//	SSS+ (1,009,000～): 譜面定数 + 2.15
//	SSS  (1,007,500～): 譜面定数 + 2.0 + (score - 1,007,500) / 100 * 0.01
//	SS+  (1,005,000～): 譜面定数 + 1.5 + (score - 1,005,000) / 50 * 0.01
//	SS   (1,000,000～): 譜面定数 + 1.0 + (score - 1,000,000) / 100 * 0.01
//	S+   (990,000～):   譜面定数 + 0.6 + (score - 990,000) / 250 * 0.01
//	S    (975,000～):   譜面定数 + (score - 975,000) / 2,500 * 0.1
//	AAA  (950,000～):   譜面定数 - 1.67 + (score - 950,000) / 150 * 0.01
//	AA   (925,000～):   譜面定数 - 3.34 + (score - 925,000) / 150 * 0.01
//	A    (900,000～):   譜面定数 - 5.0 + (score - 900,000) / 150 * 0.01
//	BBB  (800,000～):   (譜面定数 - 5.0) / 2 + (score - 800,000) / (2,000 / (譜面定数 - 5)) * 0.01
//	C    (500,000～):   (score - 500,000) / (6,000 / (譜面定数 - 5)) * 0.01
//	D    (～500,000):   0
//
// 計算中は0.01単位の整数を使用し、仕様上の切り捨てを整数除算で行います。
func CalcSingleRating(score uint32, chartConst float64) float64 {
	return float64(calcSingleRatingHundredths(score, chartConst)) / float64(ratingScale)
}

func calcSingleRatingHundredths(score uint32, chartConst float64) int64 {
	constTenths := chartConstTenths(chartConst)
	base := constTenths * 10
	var rating int64

	switch {
	case score >= 1_009_000:
		// SSS+: 譜面定数 + 2.15（上限）
		rating = base + 215
	case score >= 1_007_500:
		// SSS: 譜面定数 + 2.0、100点毎に+0.01
		rating = base + 200 + int64(score-1_007_500)/100
	case score >= 1_005_000:
		// SS+: 譜面定数 + 1.5、50点毎に+0.01
		rating = base + 150 + int64(score-1_005_000)/50
	case score >= 1_000_000:
		// SS: 譜面定数 + 1.0、100点毎に+0.01
		rating = base + 100 + int64(score-1_000_000)/100
	case score >= 990_000:
		// S+: 譜面定数 + 0.6、250点毎に+0.01
		rating = base + 60 + int64(score-990_000)/250
	case score >= 975_000:
		// S: 譜面定数、250点毎に+0.01
		rating = base + int64(score-975_000)/250
	case score >= 950_000:
		// AAA: 譜面定数 - 1.67、150点毎に+0.01
		rating = base - 167 + int64(score-950_000)/150
	case score >= 925_000:
		// AA: 譜面定数 - 3.34、150点毎に+0.01
		rating = base - 334 + int64(score-925_000)/150
	case score >= 900_000:
		// A: 譜面定数 - 5.0、150点毎に+0.01
		rating = base - 500 + int64(score-900_000)/150
	case score >= 800_000:
		// BBB: (譜面定数 - 5.0) / 2 から線形増加
		diff := constTenths - 50
		if diff > 0 {
			rating = diff*5 + int64(score-800_000)*diff/20_000
		}
	case score >= 500_000:
		// C: 0から(譜面定数 - 5.0) / 2まで線形増加
		diff := constTenths - 50
		if diff > 0 {
			rating = int64(score-500_000) * diff / 60_000
		}
	}

	return max(rating, 0)
}

// CalcSingleOverpower は指定されたスコア、譜面定数、コンボランプから単曲OVER POWERを計算します。
//
// 計算式:
//
//	SSS以上 (1,007,500～): (譜面定数 + 2) × 5 + (score - 1,007,500) / 2,500 × 3.75
//	SS+     (1,005,000～): (譜面定数 + 1.5) × 5 + (score - 1,005,000) / 2,500 × 2.5
//	SS      (1,000,000～): (譜面定数 + 1) × 5 + (score - 1,000,000) / 5,000 × 2.5
//	S～S+   (975,000～):   譜面定数 × 5 + (score - 975,000) / 25,000 × 5
//	A～AAA  (900,000～):   (譜面定数 - 5) × 5 + (score - 900,000) / 75,000 × 25
//
// コンボランプ補正:
//   - FULL COMBO: +0.5
//   - ALL JUSTICE: +1.0
//   - 理論値（1,010,000点）: +1.25
//
// 計算中は0.001単位の整数を使用し、S以上は0.005、S未満は0.05単位で切り捨てます。
func CalcSingleOverpower(score uint32, chartConst float64, comboLampID int) float64 {
	return float64(calcSingleOverpowerThousandths(score, chartConst, comboLampID)) / float64(overpowerScale)
}

func calcSingleOverpowerThousandths(score uint32, chartConst float64, comboLampID int) int64 {
	constTenths := chartConstTenths(chartConst)
	var overpower int64

	switch {
	case score >= 1_007_500:
		// SSS以上: (譜面定数 + 2) × 5 + スコア補正
		overpower = (constTenths+20)*500 + int64(score-1_007_500)*3/2
	case score >= 1_005_000:
		// SS+: (譜面定数 + 1.5) × 5 + スコア補正
		overpower = (constTenths+15)*500 + int64(score-1_005_000)
	case score >= 1_000_000:
		// SS: (譜面定数 + 1) × 5 + スコア補正
		overpower = (constTenths+10)*500 + int64(score-1_000_000)/2
	case score >= 975_000:
		// S～S+: 譜面定数 × 5 + スコア補正
		overpower = constTenths*500 + int64(score-975_000)/5
	case score >= 900_000:
		// A～AAA: (譜面定数 - 5) × 5 + スコア補正
		overpower = (constTenths-50)*500 + int64(score-900_000)/3
	case score >= 800_000:
		// BBB: (譜面定数 - 5) / 2 × 5から線形増加
		diff := constTenths - 50
		if diff > 0 {
			overpower = diff*250 + int64(score-800_000)*diff/400
		}
	case score >= 500_000:
		// C: 0から(譜面定数 - 5) / 2 × 5まで線形増加
		diff := constTenths - 50
		if diff > 0 {
			overpower = int64(score-500_000) * diff / 1_200
		}
	}

	if score == constants.TheoreticalScore {
		// 理論値ではコンボランプ補正の代わりに+1.25する
		overpower += 1_250
	} else {
		switch comboLampID {
		case comboLampAllJustice:
			// ALL JUSTICE: +1.0
			overpower += 1_000
		case comboLampFullCombo:
			// FULL COMBO: +0.5
			overpower += 500
		}
	}

	// S以上は0.005単位、S未満は0.05単位で切り捨てる
	unit := int64(50)
	if score >= 975_000 {
		unit = 5
	}
	return max(overpower/unit*unit, 0)
}

// CalcSongMaxOP は理論値を取った際の楽曲最大OVER POWERを返します。
func CalcSongMaxOP(maxChartConst float64) float64 {
	if maxChartConst <= 0 {
		return 0
	}
	return CalcSingleOverpower(constants.TheoreticalScore, maxChartConst, comboLampAllJustice)
}

// CalcSingleOverpowerPercent は譜面別理論値に対する達成割合を小数点以下4桁で返します。
func CalcSingleOverpowerPercent(score uint32, chartConst float64, comboLampID int) float64 {
	if chartConstTenths(chartConst) <= 0 {
		return 0
	}
	maxOverpower := calcSingleOverpowerThousandths(constants.TheoreticalScore, chartConst, comboLampAllJustice)
	if maxOverpower <= 0 {
		return 0
	}

	overpower := calcSingleOverpowerThousandths(score, chartConst, comboLampID)
	scaledPercent := min(overpower*100*percentScale/maxOverpower, 100*percentScale)
	return float64(max(scaledPercent, 0)) / float64(percentScale)
}

// RatingRecord はレーティング計算に必要な単曲の情報を保持します。
type RatingRecord struct {
	Score      uint32
	ChartConst float64
	IsNew      bool
}

// RatingStats はプレイヤーのレーティング統計情報を保持します。
// DB・APIとの互換性を維持するため公開値はfloat64ですが、集計は整数で行います。
type RatingStats struct {
	PlayerRating float64
	BestAverage  float64
	NewAverage   float64
}

// CalcRatingStats はレコードリストからプレイヤーレーティング統計を一括計算します。
func CalcRatingStats(records []RatingRecord) RatingStats {
	// 1. 単曲レーティングをBEST枠とNEW枠に分けて計算する
	bestRatings := make([]int64, 0, len(records))
	newRatings := make([]int64, 0, len(records))
	for _, rec := range records {
		rating := calcSingleRatingHundredths(rec.Score, rec.ChartConst)
		if rec.IsNew {
			newRatings = append(newRatings, rating)
		} else {
			bestRatings = append(bestRatings, rating)
		}
	}

	// 2. BEST系レコードから単曲レーティング上位30曲を選ぶ
	slices.SortFunc(bestRatings, func(a, b int64) int { return cmp.Compare(b, a) })
	bestCount := min(30, len(bestRatings))
	bestSum := sumRatings(bestRatings[:bestCount])

	// 3. NEW系レコードから単曲レーティング上位20曲を選ぶ
	slices.SortFunc(newRatings, func(a, b int64) int { return cmp.Compare(b, a) })
	newCount := min(20, len(newRatings))
	newSum := sumRatings(newRatings[:newCount])

	// 4. 各平均と、50枠固定のプレイヤーレーティングを小数点以下4桁で切り捨てる
	return RatingStats{
		PlayerRating: scaledAverage(bestSum+newSum, playerRatingSlotCount),
		BestAverage:  scaledAverage(bestSum, bestCount),
		NewAverage:   scaledAverage(newSum, newCount),
	}
}

func sumRatings(ratings []int64) int64 {
	var sum int64
	for _, rating := range ratings {
		sum += rating
	}
	return sum
}

func scaledAverage(sum int64, count int) float64 {
	if count == 0 {
		return 0
	}
	scaled := sum * aggregateRatingScale / ratingScale / int64(count)
	return float64(scaled) / float64(aggregateRatingScale)
}

func chartConstTenths(value float64) int64 {
	return int64(math.Round(value * 10))
}

// roundN は入出力境界で扱う小数を指定桁数に丸めます。
func roundN(num float64, n int) float64 {
	factor := math.Pow10(n)
	return math.Round(num*factor) / factor
}

// truncN は入出力境界で扱う小数を指定桁数で切り捨てます。
func truncN(num float64, n int) float64 {
	factor := math.Pow10(n)
	return math.Floor(num*factor+1e-7) / factor
}
