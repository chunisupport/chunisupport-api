package service

import (
	"cmp"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
)

// RatingSlotRecord はRating枠選出に必要な純粋なドメイン入力です。
type RatingSlotRecord struct {
	ChartID       int
	Score         uint32
	ChartConst    float64
	OfficialIndex uint64
}

// RatingSlotResult は本枠と候補枠の選出結果です。
type RatingSlotResult struct {
	Main       []RatingSlotRecord
	Candidates []RatingSlotRecord
}

// BuildRatingSlots は単一の排他的プールから本枠と候補枠を選出します。
func BuildRatingSlots(records []RatingSlotRecord, mainLimit, candidateLimit int) RatingSlotResult {
	sorted := slices.Clone(records)
	slices.SortFunc(sorted, compareRatingSlotRecord)
	mainCount := min(mainLimit, len(sorted))
	result := RatingSlotResult{Main: slices.Clone(sorted[:mainCount])}
	if mainCount < mainLimit {
		return result
	}

	candidates := make([]RatingSlotRecord, 0, len(sorted)-mainCount)
	mainThresholdRating := calcSlotSortRatingHundredths(sorted[mainCount-1].Score, sorted[mainCount-1].ChartConst)
	for _, record := range sorted[mainCount:] {
		if record.Score >= constants.SSSPlusScore {
			continue
		}
		assumedRating := calcSingleRatingHundredths(constants.SSSPlusScore, record.ChartConst)
		if assumedRating > mainThresholdRating {
			candidates = append(candidates, record)
		}
	}
	slices.SortFunc(candidates, compareRatingSlotRecord)
	result.Candidates = slices.Clone(candidates[:min(candidateLimit, len(candidates))])
	return result
}

// FindCandidatesAboveMainThreshold は本枠下限の単曲レートを上回る候補枠レコードを返します。
// 公式の候補枠は本枠に入らなかった譜面から選ばれるため、該当があれば定数マスターの誤りなどを示します。
// 公式順位と同じ基準で比べるため並び替え用の単曲レートを使い、同値は正常として扱います。
func FindCandidatesAboveMainThreshold(main, candidates []RatingSlotRecord) []RatingSlotRecord {
	if len(main) == 0 {
		return nil
	}
	threshold := calcSlotSortRatingHundredths(main[0].Score, main[0].ChartConst)
	for _, record := range main[1:] {
		threshold = min(threshold, calcSlotSortRatingHundredths(record.Score, record.ChartConst))
	}
	var exceeded []RatingSlotRecord
	for _, record := range candidates {
		if calcSlotSortRatingHundredths(record.Score, record.ChartConst) > threshold {
			exceeded = append(exceeded, record)
		}
	}
	return exceeded
}

func compareRatingSlotRecord(a, b RatingSlotRecord) int {
	if ratingComparison := cmp.Compare(calcSlotSortRatingHundredths(b.Score, b.ChartConst), calcSlotSortRatingHundredths(a.Score, a.ChartConst)); ratingComparison != 0 {
		return ratingComparison
	}
	if constantComparison := cmp.Compare(chartConstTenths(b.ChartConst), chartConstTenths(a.ChartConst)); constantComparison != 0 {
		return constantComparison
	}
	// SSS+（1,009,000点）以上はスコア差を無視して公式ID順で並べるため、上限で丸めて比較する。
	if scoreComparison := cmp.Compare(min(a.Score, constants.SSSPlusScore), min(b.Score, constants.SSSPlusScore)); scoreComparison != 0 {
		return scoreComparison
	}
	return cmp.Compare(a.OfficialIndex, b.OfficialIndex)
}

// calcSlotSortRatingHundredths は枠の並び替えに使う単曲レーティング（0.01単位の整数）を計算します。
// 公式枠の実測順位に合わせるため、以下の2点だけ通常計算から補正します。
//   - ちょうど1,008,999点は SSS+ と同じ「譜面定数 + 2.15」扱いとする
//   - 900,000～974,999点はランク帯ごとの途中切り捨てを行わず、
//     譜面定数 + (score - 975,000) / 15,000 を最後に小数第2位で切り捨てる
func calcSlotSortRatingHundredths(score uint32, chartConst float64) int64 {
	constTenths := chartConstTenths(chartConst)
	base := constTenths * 10

	switch {
	case score == constants.SSSPlusScore-1:
		return max(base+215, 0)
	case score >= 900_000 && score < 975_000:
		// (score - 975,000) / 150 の床関数は (score - 900,000) / 150 - 500 と等価で、
		// 非負の整数除算のみで正確に切り捨てられる。
		return max(base+int64(score-900_000)/150-500, 0)
	default:
		return calcSingleRatingHundredths(score, chartConst)
	}
}

// AggregateOfficialRating は公式本枠だけを使ってRating三値を集計します。
func AggregateOfficialRating(best, newRecords []RatingSlotRecord) RatingStats {
	bestRatings := make([]int64, 0, len(best))
	for _, record := range best {
		bestRatings = append(bestRatings, calcSingleRatingHundredths(record.Score, record.ChartConst))
	}
	newRatings := make([]int64, 0, len(newRecords))
	for _, record := range newRecords {
		newRatings = append(newRatings, calcSingleRatingHundredths(record.Score, record.ChartConst))
	}
	bestSum := sumRatings(bestRatings)
	newSum := sumRatings(newRatings)
	return RatingStats{
		PlayerRating: scaledAverage(bestSum+newSum, playerRatingSlotCount),
		BestAverage:  scaledAverage(bestSum, len(bestRatings)),
		NewAverage:   scaledAverage(newSum, len(newRatings)),
	}
}
