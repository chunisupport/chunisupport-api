// Package service はドメイン層のサービスを提供します。
// ここにはエンティティや値オブジェクトに属さない計算ロジックなどが含まれます。
package service

import (
	"cmp"
	"math"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
)

// CalcSingleRating は指定されたスコアと譜面定数から単曲レーティングを計算します。
// スコアが低い場合は0を返し、SSS+（1,009,000点）で譜面定数+2.15が上限となります。
//
// 計算式（Wikiより）:
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
//	BBB  (800,000～):   (譜面定数 - 5.0) / 2 + (score - 800,000) / (2000 / (譜面定数 - 5)) * 0.01
//	C    (500,000～):   (score - 500,000) / (6000 / (譜面定数 - 5)) * 0.01
//	D    (～500,000):   0
func CalcSingleRating(score uint32, chartConst float64) float64 {
	var rating float64

	switch {
	case score >= 1009000:
		// SSS+: 譜面定数 + 2.15（上限）
		rating = chartConst + 2.15
	case score >= 1007500:
		// SSS: 譜面定数 + 2.0、100点毎に+0.01
		rating = chartConst + 2.0 + float64(score-1007500)/100*0.01
	case score >= 1005000:
		// SS+: 譜面定数 + 1.5、50点毎に+0.01
		rating = chartConst + 1.5 + float64(score-1005000)/50*0.01
	case score >= 1000000:
		// SS: 譜面定数 + 1.0、100点毎に+0.01
		rating = chartConst + 1.0 + float64(score-1000000)/100*0.01
	case score >= 990000:
		// S+: 譜面定数 + 0.6、250点毎に+0.01
		rating = chartConst + 0.6 + float64(score-990000)/250*0.01
	case score >= 975000:
		// S: 譜面定数、2500点毎に+0.1（= 250点毎に+0.01）
		rating = chartConst + float64(score-975000)/2500*0.1
	case score >= 950000:
		// AAA: 譜面定数 - 1.67、150点毎に+0.01
		rating = chartConst - 1.67 + float64(score-950000)/150*0.01
	case score >= 925000:
		// AA: 譜面定数 - 3.34、150点毎に+0.01
		rating = chartConst - 3.34 + float64(score-925000)/150*0.01
	case score >= 900000:
		// A: 譜面定数 - 5.0、150点毎に+0.01
		rating = chartConst - 5.0 + float64(score-900000)/150*0.01
	case score >= 800000:
		// BBB: (譜面定数 - 5.0) / 2 から線形増加
		if chartConst > 5.0 {
			rating = (chartConst-5.0)/2 + float64(score-800000)/(2000/(chartConst-5))*0.01
		} else {
			rating = 0
		}
	case score >= 500000:
		// C: 0 から (譜面定数 - 5.0) / 2 まで線形増加
		if chartConst > 5.0 {
			rating = float64(score-500000) / (6000 / (chartConst - 5)) * 0.01
		} else {
			rating = 0
		}
	default:
		// D: 0
		rating = 0
	}

	return truncN(max(rating, 0), 2)
}

// roundN は数値を小数点以下n桁で四捨五入します。
func roundN(num float64, n int) float64 {
	factor := math.Pow(10, float64(n))
	return math.Round(num*factor) / factor
}

// truncN は数値を小数点以下n桁で切り捨てます。
// 浮動小数点の丸め誤差を吸収するため、適切なepsilonを加算します。
func truncN(num float64, n int) float64 {
	factor := math.Pow(10, float64(n))
	// 浮動小数点誤差を吸収（例: 17.149999999999 -> 17.15）
	// epsilonは切り捨てに影響しない程度に小さく、誤差吸収に十分な値
	const epsilon = 0.0000001
	return math.Floor(num*factor+epsilon) / factor
}

// CalcSingleOverpower は指定されたスコア、譜面定数、コンボランプから単曲 OVER POWER を計算します。
// コンボランプによる補正:
//   - comboLampID == 2 (FULL COMBO): +0.5
//   - comboLampID == 3 (ALL JUSTICE): +1.0
//   - スコア == 1,010,000 (AJC/理論値): +1.25
//
// 計算式:
//
//	S以上 (975,000～1,007,500):  レーティング値 × 5 + 補正1
//	SSS以上 (1,007,501～):       (譜面定数 + 2) × 5 + 補正1 + 補正2
//	  補正2 = (スコア - 1,007,500) × 0.0015 （最大3.75）
//	AJC (1,010,000):             (譜面定数 + 3) × 5
//
// 精度:
//   - S以上: 0.005単位（小数点以下3桁目を切り捨て）
//   - S未満: 0.05単位（小数点以下2桁目を切り捨て）
func CalcSingleOverpower(score uint32, chartConst float64, comboLampID int) float64 {
	var overPower float64

	switch {
	case score >= 1007500:
		// SSS以上: (譜面定数+2)×5 + 補正2
		overPower = (chartConst+2)*5 + float64(score-1007500)/2500*3.75
	case score >= 1005000:
		// SS+
		overPower = (chartConst+1.5)*5 + float64(score-1005000)/2500*2.5
	case score >= 1000000:
		// SS
		overPower = (chartConst+1)*5 + float64(score-1000000)/5000*2.5
	case score >= 975000:
		// S～S+
		overPower = chartConst*5 + float64(score-975000)/25000*5
	case score >= 900000:
		// A～AAA
		overPower = (chartConst-5)*5 + float64(score-900000)/75000*25
	case score >= 800000:
		// BBB
		overPower = (chartConst-5)/2*5 + float64(score-800000)/100000*(chartConst-5)*5/2
	case score >= 500000:
		// C
		overPower = float64(score-500000) / 300000 * (chartConst - 5) * 5 / 2
	default:
		overPower = 0
	}

	// コンボランプ補正
	if score == constants.TheoreticalScore {
		// AJC（理論値）: +1.25
		overPower += 1.25
	} else {
		switch comboLampID {
		case comboLampAllJustice: // ALL JUSTICE
			overPower += 1.0
		case comboLampFullCombo: // FULL COMBO
			overPower += 0.5
		}
	}

	// 精度調整: S以上は0.005単位、S未満は0.05単位
	if score >= 975000 {
		overPower = math.Floor(roundN(overPower*200, 2)) / 200 // 0.005単位
	} else {
		overPower = math.Floor(roundN(overPower*20, 3)) / 20 // 0.05単位
	}

	return max(overPower, 0)
}

// CalcSongMaxOP は楽曲の最大譜面定数から、理論値(AJC)を取った際のOPを返します。
// maxChartConst はドメインサービスの AggregateSongCharts で算出された値を受け取ります。
func CalcSongMaxOP(maxChartConst float64) float64 {
	if maxChartConst <= 0 {
		return 0
	}

	return CalcSingleOverpower(constants.TheoreticalScore, maxChartConst, comboLampAllJustice)
}

// RatingRecord はレーティング計算に必要な単曲の情報を保持します。
type RatingRecord struct {
	Score      uint32  // スコア
	ChartConst float64 // 譜面定数
	IsNew      bool    // 新曲枠に属するか
}

// RatingStats はプレイヤーのレーティング統計情報を保持します。
type RatingStats struct {
	PlayerRating float64 // プレイヤーレーティング
	BestAverage  float64 // ベスト枠平均
	NewAverage   float64 // 新曲枠平均
}

// CalcRatingStats はレコードリストからプレイヤーレーティング統計を一括計算します。
func CalcRatingStats(records []RatingRecord) RatingStats {
	type ratedRecord struct {
		rating float64
		isNew  bool
	}

	// 1. 全レコードの単曲レーティングを計算
	rated := make([]ratedRecord, 0, len(records))
	for _, rec := range records {
		rating := CalcSingleRating(rec.Score, rec.ChartConst)
		rated = append(rated, ratedRecord{
			rating: rating,
			isNew:  rec.IsNew,
		})
	}

	// 2. ベスト枠: 全レコードから上位30曲
	// スコア順にソート（降順）
	slices.SortFunc(rated, func(a, b ratedRecord) int {
		return cmp.Compare(b.rating, a.rating)
	})

	bestSum := 0.0
	bestCount := min(30, len(rated))
	for i := range bestCount {
		bestSum += rated[i].rating
	}

	bestAvg := 0.0
	if bestCount > 0 {
		bestAvg = truncN(bestSum/float64(bestCount), 2)
	}

	// 3. 新曲枠: 新曲のみを抽出して上位20曲
	newRatings := make([]float64, 0, len(rated))
	for _, r := range rated {
		if r.isNew {
			newRatings = append(newRatings, r.rating)
		}
	}
	// 既にソートされているが、抽出後に再度ソートは不要（元の順序が保存されているため）
	// ただし、念のためソートしておく（安全策）
	slices.SortFunc(newRatings, func(a, b float64) int {
		return cmp.Compare(b, a)
	})

	newSum := 0.0
	newCount := min(20, len(newRatings))
	for i := range newCount {
		newSum += newRatings[i]
	}

	newAvg := 0.0
	if newCount > 0 {
		newAvg = truncN(newSum/float64(newCount), 2)
	}

	// 4. プレイヤーレーティング: (ベスト枠合計 + 新曲枠合計) / 50
	playerRating := 0.0
	// 50枠に満たない場合でも、計算式は常に50で割る（CHUNITHMの仕様）
	// 厳密には、ベスト枠+新曲枠の合計を50で割る
	totalSum := bestSum + newSum
	if bestCount+newCount > 0 {
		playerRating = truncN(totalSum/50.0, 2)
	}

	return RatingStats{
		PlayerRating: playerRating,
		BestAverage:  bestAvg,
		NewAverage:   newAvg,
	}
}
