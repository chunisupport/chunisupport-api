package service

import "github.com/chunisupport/chunisupport-api/internal/domain/entity"

// SongAggregation は楽曲の譜面集約結果を保持します。
type SongAggregation struct {
	MaxChartConst        float64
	IsMaxOPUnknown       bool
	OpTargetDifficultyID int
}

// AggregateSongCharts は譜面リストから楽曲の最大譜面定数とMAXOP確度を計算します。
//
// 判定ルール:
//   - MaxChartConst: 全譜面のうち最大の定数値
//   - OpTargetDifficultyID: 理論値OVER POWERが最大となる譜面の難易度ID。
//     定数が同値の場合は難易度IDが大きい譜面を採用する。
//   - IsMaxOPUnknown: MASTER(4)またはULTIMA(5)の譜面に is_const_unknown=true が
//     1件でも含まれれば true。EXPERT以下のunknownは判定対象外。
func AggregateSongCharts(charts []*entity.Chart) SongAggregation {
	song := &entity.Song{Charts: charts}
	song.RecalculateChartAggregation()

	return SongAggregation{
		MaxChartConst:        song.MaxChartConst,
		IsMaxOPUnknown:       song.IsMaxOPUnknown,
		OpTargetDifficultyID: song.OpTargetDifficultyID,
	}
}

// ApplyAggregation は楽曲エンティティの譜面リストから集約結果を計算し、
// MaxChartConst、IsMaxOPUnknown、OpTargetDifficultyID をエンティティに適用します。
func ApplyAggregation(song *entity.Song) {
	song.RecalculateChartAggregation()
}
