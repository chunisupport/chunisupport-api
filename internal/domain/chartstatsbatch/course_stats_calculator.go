package chartstatsbatch

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

// CourseStatsCalculator は記録をコース×レーティング帯の統計へ集計します。
// 譜面統計と同じく、記録はコースID順に渡してもらい、コースが切り替わった時点で前のコースの集計を確定して、
// 中央値のために保持するスコアを1コース分に抑えます。
type CourseStatsCalculator struct {
	ratingBands     []*ratingband.RatingBand
	currentCourseID int
	// current は ratingBands と同じ添字で帯ごとの集計を保持し、記録のない帯は nil です。
	current     []*courseStatsAccumulator
	results     []*entity.CourseStatsByRatingBand
	recordCount int
}

// NewCourseStatsCalculator は指定したレーティング帯で集計する CourseStatsCalculator を生成します。
func NewCourseStatsCalculator(ratingBands []*ratingband.RatingBand) *CourseStatsCalculator {
	return &CourseStatsCalculator{ratingBands: ratingBands}
}

// Add は記録を集計に加えます。記録はプレイヤーのレーティングを含むすべての帯（ALL を含む）に加算します。
func (c *CourseStatsCalculator) Add(record CourseRecord) {
	if c.current != nil && record.CourseID != c.currentCourseID {
		c.flush()
	}
	if c.current == nil {
		c.current = make([]*courseStatsAccumulator, len(c.ratingBands))
		c.currentCourseID = record.CourseID
	}
	for i, band := range c.ratingBands {
		if !band.Contains(record.BestAverageRating) {
			continue
		}
		if c.current[i] == nil {
			c.current[i] = &courseStatsAccumulator{
				stats: entity.CourseStatsByRatingBand{CourseID: record.CourseID, RatingBandID: band.ID},
			}
		}
		c.current[i].add(record)
	}
	c.recordCount++
}

// Results は集計結果をコースの出現順、同じコース内ではレーティング帯の順に返します。記録のない帯の行は含みません。
func (c *CourseStatsCalculator) Results() []*entity.CourseStatsByRatingBand {
	c.flush()
	return c.results
}

// RecordCount は集計に加えた記録の件数です。
func (c *CourseStatsCalculator) RecordCount() int {
	return c.recordCount
}

func (c *CourseStatsCalculator) flush() {
	for _, accumulator := range c.current {
		if accumulator != nil {
			c.results = append(c.results, accumulator.result())
		}
	}
	c.current = nil
}

// courseStatsAccumulator はコース×レーティング帯1件分の集計途中の値です。
type courseStatsAccumulator struct {
	stats             entity.CourseStatsByRatingBand
	clearedScoreTotal int64
	// 合計と件数だけでは中央値を求められないため、個々のクリア済みスコアを保持します。
	clearedScores []int
}

// add は記録を加算します。未クリアの記録は人数とランプには含め、スコアに関する統計からだけ除外します。
// 未クリアのコースは途中で終了した分のスコアしか持たず、完走した記録と混ぜると平均・中央値が実力を表さなくなるためです。
func (a *courseStatsAccumulator) add(record CourseRecord) {
	a.stats.PlayerCount++
	switch record.ComboLampID {
	case comboLampIDNone:
		a.stats.Combo.None++
	case comboLampIDFullCombo:
		a.stats.Combo.FC++
	case comboLampIDAllJustice:
		a.stats.Combo.AJ++
	}
	if !record.IsClear {
		a.stats.Clear.Failed++
		return
	}
	a.stats.Clear.Clear++
	a.clearedScoreTotal += int64(record.Score)
	a.clearedScores = append(a.clearedScores, record.Score)
}

// result は平均・中央値を確定した統計を返します。クリア済み記録がない場合、平均・中央値は nil です。
func (a *courseStatsAccumulator) result() *entity.CourseStatsByRatingBand {
	stats := a.stats
	stats.AverageScore, stats.MedianScore = averageAndMedian(a.clearedScoreTotal, a.clearedScores)
	return &stats
}
