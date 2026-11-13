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

func TestAggregateOfficialRating_両枠とも空のときはゼロを返す(t *testing.T) {
	// Given: ベスト枠も新曲枠も空（データ連携直後の状態）
	// When: 公式レーティングを集計する
	got := AggregateOfficialRating(nil, nil)

	// Then: ゼロ除算にならず三値すべて0になる
	assert.Equal(t, RatingStats{PlayerRating: 0, BestAverage: 0, NewAverage: 0}, got)
}

func TestBuildRatingSlots_候補枠は上限件数で切り詰めて単曲レート順に残す(t *testing.T) {
	// Given: 本枠2件（SSS+・定数15.0 → 単曲17.15）と候補相当4件（SS・定数15.1〜15.4 → 単曲16.10〜16.40）
	// 候補相当のSSS+仮定レート（定数+2.15 = 17.25〜17.55）は本枠下限17.15を上回る
	records := []RatingSlotRecord{
		{ChartID: 1, Score: 1_009_000, ChartConst: 15.0, OfficialIndex: 1},
		{ChartID: 2, Score: 1_009_000, ChartConst: 15.0, OfficialIndex: 2},
		{ChartID: 3, Score: 1_000_000, ChartConst: 15.4, OfficialIndex: 3},
		{ChartID: 4, Score: 1_000_000, ChartConst: 15.3, OfficialIndex: 4},
		{ChartID: 5, Score: 1_000_000, ChartConst: 15.2, OfficialIndex: 5},
		{ChartID: 6, Score: 1_000_000, ChartConst: 15.1, OfficialIndex: 6},
	}

	// When: 本枠2・候補枠2で選出する
	result := BuildRatingSlots(records, 2, 2)

	// Then: 本枠は単曲レート上位2件、候補枠は単曲レート順の上位2件に切り詰められる
	require.Len(t, result.Main, 2)
	assert.Equal(t, []int{1, 2}, []int{result.Main[0].ChartID, result.Main[1].ChartID})
	require.Len(t, result.Candidates, 2)
	assert.Equal(t, []int{3, 4}, []int{result.Candidates[0].ChartID, result.Candidates[1].ChartID})
}

func TestBuildRatingSlots_逆順入力でも単曲レート降順に並ぶ(t *testing.T) {
	// Given: 単曲レート昇順（SSS+・定数14.6〜15.0 → 単曲16.75〜17.15）の入力
	records := []RatingSlotRecord{
		{ChartID: 1, Score: 1_009_000, ChartConst: 14.6, OfficialIndex: 1},
		{ChartID: 2, Score: 1_009_000, ChartConst: 14.7, OfficialIndex: 2},
		{ChartID: 3, Score: 1_009_000, ChartConst: 14.8, OfficialIndex: 3},
		{ChartID: 4, Score: 1_009_000, ChartConst: 14.9, OfficialIndex: 4},
		{ChartID: 5, Score: 1_009_000, ChartConst: 15.0, OfficialIndex: 5},
	}

	// When: 全件を本枠として選出する
	result := BuildRatingSlots(records, 5, 10)

	// Then: 単曲レート降順に並び替わる
	require.Len(t, result.Main, 5)
	assert.Equal(t, []int{5, 4, 3, 2, 1}, []int{
		result.Main[0].ChartID,
		result.Main[1].ChartID,
		result.Main[2].ChartID,
		result.Main[3].ChartID,
		result.Main[4].ChartID,
	})
	assert.Empty(t, result.Candidates)
}

func TestBuildRatingSlots_本枠不足時は全件本枠で候補なし(t *testing.T) {
	result := BuildRatingSlots([]RatingSlotRecord{{ChartID: 1}}, 30, 10)
	assert.Len(t, result.Main, 1)
	assert.Empty(t, result.Candidates)
}
