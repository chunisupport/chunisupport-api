package chartstatsbatch

import "github.com/chunisupport/chunisupport-api/internal/domain/scorepercentile"

// ScorePercentileCalculator は譜面・コースごとの全プレイ記録（未クリア含む）から上位%境界データを集計します。
// 記録は対象ID順に渡してもらい、IDが切り替わった時点で前の対象の境界データを確定してメモリ消費を1対象分に抑えます。
type ScorePercentileCalculator struct {
	hasCurrent      bool
	currentTargetID int
	currentScores   []int
	results         []scorepercentile.Entry
}

// NewScorePercentileCalculator は ScorePercentileCalculator を生成します。
func NewScorePercentileCalculator() *ScorePercentileCalculator {
	return &ScorePercentileCalculator{}
}

// Add は対象IDとスコアを集計に加えます。未クリア記録も除外せずそのまま含めます。
func (c *ScorePercentileCalculator) Add(targetID int, score int) {
	if c.hasCurrent && targetID != c.currentTargetID {
		c.flush()
	}
	if !c.hasCurrent {
		c.hasCurrent = true
		c.currentTargetID = targetID
	}
	c.currentScores = append(c.currentScores, score)
}

// Results は集計結果を出現順に返します。
func (c *ScorePercentileCalculator) Results() []scorepercentile.Entry {
	c.flush()
	return c.results
}

func (c *ScorePercentileCalculator) flush() {
	if !c.hasCurrent || len(c.currentScores) == 0 {
		c.hasCurrent = false
		c.currentScores = nil
		return
	}
	thresholds := scorepercentile.BuildThresholds(c.currentScores)
	if len(thresholds) > 0 {
		c.results = append(c.results, scorepercentile.Entry{
			TargetID:   c.currentTargetID,
			Thresholds: thresholds,
		})
	}
	c.hasCurrent = false
	c.currentScores = c.currentScores[:0]
}
