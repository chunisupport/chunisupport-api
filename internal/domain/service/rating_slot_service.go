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
	mainThresholdRating := calcSingleRatingHundredths(sorted[mainCount-1].Score, sorted[mainCount-1].ChartConst)
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

func compareRatingSlotRecord(a, b RatingSlotRecord) int {
	if ratingComparison := cmp.Compare(calcSingleRatingHundredths(b.Score, b.ChartConst), calcSingleRatingHundredths(a.Score, a.ChartConst)); ratingComparison != 0 {
		return ratingComparison
	}
	if constantComparison := cmp.Compare(chartConstTenths(b.ChartConst), chartConstTenths(a.ChartConst)); constantComparison != 0 {
		return constantComparison
	}
	if scoreComparison := cmp.Compare(a.Score, b.Score); scoreComparison != 0 {
		return scoreComparison
	}
	return cmp.Compare(a.OfficialIndex, b.OfficialIndex)
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
