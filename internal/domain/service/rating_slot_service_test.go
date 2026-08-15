package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRatingSlots_本枠と候補枠を決定的に選出する(t *testing.T) {
	records := make([]RatingSlotRecord, 0, 33)
	for i := 1; i <= 33; i++ {
		score := uint32(1_000_000)
		if i <= 30 {
			score = 1_008_999
		}
		records = append(records, RatingSlotRecord{
			ChartID:       i,
			Score:         score,
			ChartConst:    15,
			OfficialIndex: uint64(i),
		})
	}

	result := BuildRatingSlots(records, 30, 10)

	require.Len(t, result.Main, 30)
	assert.Equal(t, 1, result.Main[0].ChartID)
	assert.Equal(t, 30, result.Main[29].ChartID)
	require.Len(t, result.Candidates, 3)
	assert.Equal(t, []int{31, 32, 33}, []int{
		result.Candidates[0].ChartID,
		result.Candidates[1].ChartID,
		result.Candidates[2].ChartID,
	})
}

func TestBuildRatingSlots_SSSPlus済みと仮定しても本枠に入れない譜面は候補外(t *testing.T) {
	records := make([]RatingSlotRecord, 0, 32)
	for i := 1; i <= 30; i++ {
		records = append(records, RatingSlotRecord{ChartID: i, Score: 1_009_000, ChartConst: 15, OfficialIndex: uint64(i)})
	}
	records = append(records,
		RatingSlotRecord{ChartID: 31, Score: 1_008_999, ChartConst: 10, OfficialIndex: 31},
		RatingSlotRecord{ChartID: 32, Score: 1_009_000, ChartConst: 16, OfficialIndex: 32},
	)

	result := BuildRatingSlots(records, 30, 10)

	assert.Empty(t, result.Candidates)
}

func TestBuildRatingSlots_SSSPlus仮定レートが本枠下限と同値なら候補外(t *testing.T) {
	records := make([]RatingSlotRecord, 0, 31)
	for i := 1; i <= 30; i++ {
		records = append(records, RatingSlotRecord{
			ChartID:       i,
			Score:         1_009_000,
			ChartConst:    15.3,
			OfficialIndex: uint64(i),
		})
	}
	records = append(records, RatingSlotRecord{
		ChartID:       31,
		Score:         1_008_999,
		ChartConst:    15.3,
		OfficialIndex: 31,
	})

	result := BuildRatingSlots(records, 30, 10)

	assert.Empty(t, result.Candidates)
}

func TestBuildRatingSlots_SSSPlus仮定レートが本枠下限を上回れば候補になる(t *testing.T) {
	records := make([]RatingSlotRecord, 0, 31)
	for i := 1; i <= 30; i++ {
		records = append(records, RatingSlotRecord{
			ChartID:       i,
			Score:         1_009_000,
			ChartConst:    15.3,
			OfficialIndex: uint64(i),
		})
	}
	records = append(records, RatingSlotRecord{
		ChartID:       31,
		Score:         1_000_000,
		ChartConst:    15.4,
		OfficialIndex: 31,
	})

	result := BuildRatingSlots(records, 30, 10)

	require.Len(t, result.Candidates, 1)
	assert.Equal(t, 31, result.Candidates[0].ChartID)
}

func TestAggregateOfficialRating_候補を含めず公式本枠だけを集計する(t *testing.T) {
	stats := AggregateOfficialRating(
		[]RatingSlotRecord{{Score: 1_009_000, ChartConst: 15}},
		[]RatingSlotRecord{{Score: 1_009_000, ChartConst: 14}},
	)

	assert.Equal(t, 0.666, stats.PlayerRating)
	assert.Equal(t, 17.15, stats.BestAverage)
	assert.Equal(t, 16.15, stats.NewAverage)
}

func TestBuildRatingSlots_レート定数スコアが同じ時は公式IDを数値昇順で並べる(t *testing.T) {
	records := []RatingSlotRecord{
		{ChartID: 1, Score: 1_009_000, ChartConst: 15, OfficialIndex: 10},
		{ChartID: 2, Score: 1_009_000, ChartConst: 15, OfficialIndex: 2},
	}
	result := BuildRatingSlots(records, 2, 10)
	assert.Equal(t, []int{2, 1}, []int{result.Main[0].ChartID, result.Main[1].ChartID})
}

func TestBuildRatingSlots_レートと定数が同じ時はスコア昇順で並ぶ(t *testing.T) {
	records := []RatingSlotRecord{
		{ChartID: 1, Score: 1_010_000, ChartConst: 15, OfficialIndex: 1},
		{ChartID: 2, Score: 1_009_000, ChartConst: 15, OfficialIndex: 2},
	}

	result := BuildRatingSlots(records, 2, 10)

	assert.Equal(t, []int{2, 1}, []int{result.Main[0].ChartID, result.Main[1].ChartID})
}

func TestBuildRatingSlots_本枠不足時は全件本枠で候補なし(t *testing.T) {
	result := BuildRatingSlots([]RatingSlotRecord{{ChartID: 1}}, 30, 10)
	assert.Len(t, result.Main, 1)
	assert.Empty(t, result.Candidates)
}
