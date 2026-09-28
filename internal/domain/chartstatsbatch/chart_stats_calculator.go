package chartstatsbatch

import (
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

// ChartStatsCalculator は記録を譜面×レーティング帯の統計へ集計します。
// 中央値を求めるにはクリア済みスコアをすべて保持する必要があるため、記録は譜面ID順に渡してもらい、
// 譜面が切り替わった時点で前の譜面の集計を確定して、保持するスコアを1譜面分に抑えます。
type ChartStatsCalculator struct {
	ratingBands    []*ratingband.RatingBand
	currentChartID int
	// current は現在の譜面の集計で、ratingBands と同じ添字を使います。記録のない帯は nil です。
	current     []*chartStatsAccumulator
	results     []*entity.ChartStatsByRatingBand
	recordCount int
}

// NewChartStatsCalculator は指定したレーティング帯で集計する ChartStatsCalculator を生成します。
func NewChartStatsCalculator(ratingBands []*ratingband.RatingBand) *ChartStatsCalculator {
	return &ChartStatsCalculator{ratingBands: ratingBands}
}

// Add は記録を集計に加えます。記録はプレイヤーのレーティングを含むすべての帯（ALL を含む）に加算します。
func (c *ChartStatsCalculator) Add(record ChartRecord) {
	if c.current != nil && record.ChartID != c.currentChartID {
		c.flush()
	}
	if c.current == nil {
		c.current = make([]*chartStatsAccumulator, len(c.ratingBands))
		c.currentChartID = record.ChartID
	}
	for i, band := range c.ratingBands {
		if !band.Contains(record.BestAverageRating) {
			continue
		}
		if c.current[i] == nil {
			c.current[i] = &chartStatsAccumulator{
				stats: entity.ChartStatsByRatingBand{ChartID: record.ChartID, RatingBandID: band.ID},
			}
		}
		c.current[i].add(record)
	}
	c.recordCount++
}

// Results は集計結果を譜面の出現順、同じ譜面内ではレーティング帯の順に返します。記録のない帯の行は含みません。
func (c *ChartStatsCalculator) Results() []*entity.ChartStatsByRatingBand {
	c.flush()
	return c.results
}

// RecordCount は集計に加えた記録の件数です。
func (c *ChartStatsCalculator) RecordCount() int {
	return c.recordCount
}

func (c *ChartStatsCalculator) flush() {
	for _, accumulator := range c.current {
		if accumulator != nil {
			c.results = append(c.results, accumulator.result())
		}
	}
	c.current = nil
}

// chartStatsAccumulator は譜面×レーティング帯1件分の集計途中の値です。
type chartStatsAccumulator struct {
	stats             entity.ChartStatsByRatingBand
	clearedScoreTotal int64
	// clearedScores は中央値の計算用です。件数が多いためスコアの値域に収まる int32 で保持します。
	clearedScores []int32
}

// add は記録を加算します。未クリアの記録は人数とランプには含め、スコアに関する統計からだけ除外します。
func (a *chartStatsAccumulator) add(record ChartRecord) {
	a.stats.PlayerCount++
	a.addClearLamp(record.ClearLampID)
	a.addComboLamp(record.ComboLampID, record.Score)
	if record.ClearLampID != clearLampIDFailed {
		a.addClearedScore(record.Score)
	}
}

func (a *chartStatsAccumulator) addClearLamp(clearLampID int) {
	switch clearLampID {
	case clearLampIDFailed:
		a.stats.Clear.Failed++
	case clearLampIDClear:
		a.stats.Clear.Clear++
	case clearLampIDHard:
		a.stats.Clear.Hard++
	case clearLampIDBrave:
		a.stats.Clear.Brave++
	case clearLampIDAbsolute:
		a.stats.Clear.Absolute++
	case clearLampIDCatastrophy:
		a.stats.Clear.Catastrophy++
	}
}

// addComboLamp はコンボランプを加算します。AJ のうち理論値は AJC として数え、AJ とは重複させません。
func (a *chartStatsAccumulator) addComboLamp(comboLampID int, score int) {
	switch comboLampID {
	case comboLampIDNone:
		a.stats.Combo.None++
	case comboLampIDFullCombo:
		a.stats.Combo.FC++
	case comboLampIDAllJustice:
		if score == scoreMax {
			a.stats.Combo.AJC++
		} else {
			a.stats.Combo.AJ++
		}
	}
}

func (a *chartStatsAccumulator) addClearedScore(score int) {
	a.clearedScoreTotal += int64(score)
	a.clearedScores = append(a.clearedScores, int32(score))

	rank := &a.stats.Rank
	switch {
	case score >= scoreMax:
		rank.Max++
	case score >= scoreSSSPlus:
		rank.SSSP++
	case score >= scoreSSS:
		rank.SSS++
	case score >= scoreSSPlus:
		rank.SSP++
	case score >= scoreSS:
		rank.SS++
	case score >= scoreSPlus:
		rank.SP++
	case score >= scoreS:
		rank.S++
	default:
		rank.AAAL++
	}
}

// result は平均・中央値を確定した統計を返します。クリア済み記録がない場合、平均・中央値は nil です。
func (a *chartStatsAccumulator) result() *entity.ChartStatsByRatingBand {
	stats := a.stats
	count := len(a.clearedScores)
	if count == 0 {
		return &stats
	}

	average := float64(a.clearedScoreTotal) / float64(count)
	slices.Sort(a.clearedScores)
	median := float64(a.clearedScores[count/2])
	if count%2 == 0 {
		median = (float64(a.clearedScores[count/2-1]) + median) / 2
	}
	stats.AverageScore = &average
	stats.MedianScore = &median
	return &stats
}
