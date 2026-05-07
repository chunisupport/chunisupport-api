package service

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// SongAggregation は楽曲の譜面集約結果を保持します。
type SongAggregation struct {
	MaxChartConst  float64
	IsMaxOPUnknown bool
}

// AggregateSongCharts は譜面リストから楽曲の最大譜面定数とMAXOP確度を計算します。
//
// 判定ルール:
//   - MaxChartConst: 全譜面のうち最大の定数値
//   - IsMaxOPUnknown: MASTER(4)またはULTIMA(5)の譜面に is_const_unknown=true が
//     1件でも含まれれば true。EXPERT以下のunknownは判定対象外。
func AggregateSongCharts(charts []*entity.Chart) SongAggregation {
	var maxConst float64
	isMaxOPUnknown := false

	for _, c := range charts {
		constVal := float64(c.Const)
		if constVal > maxConst {
			maxConst = constVal
		}

		// MASTER/ULTIMA の is_const_unknown をチェック
		if (c.DifficultyID == constants.DifficultyIDMaster || c.DifficultyID == constants.DifficultyIDUltima) && c.IsConstUnknown {
			isMaxOPUnknown = true
		}
	}

	return SongAggregation{
		MaxChartConst:  maxConst,
		IsMaxOPUnknown: isMaxOPUnknown,
	}
}

// ApplyAggregation は楽曲エンティティの譜面リストから集約結果を計算し、
// MaxChartConst と IsMaxOPUnknown をエンティティに適用します。
func ApplyAggregation(song *entity.Song) {
	agg := AggregateSongCharts(song.Charts)
	song.MaxChartConst = agg.MaxChartConst
	song.IsMaxOPUnknown = agg.IsMaxOPUnknown
}
