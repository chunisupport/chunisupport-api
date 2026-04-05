package service

import "github.com/chunisupport/chunisupport-api/internal/domain/entity"

// MaxOPのunknown判定対象になる難易度名
const (
	difficultyNameMaster = "MASTER"
	difficultyNameUltima = "ULTIMA"
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
//   - IsMaxOPUnknown: MASTERまたはULTIMAの譜面に is_const_unknown=true が
//     1件でも含まれれば true。EXPERT以下のunknownは判定対象外。
func AggregateSongCharts(charts []*entity.Chart, difficultyNamesByID map[int]string) SongAggregation {
	var maxConst float64
	isMaxOPUnknown := false

	for _, c := range charts {
		constVal := float64(c.Const)
		if constVal > maxConst {
			maxConst = constVal
		}

		// MASTER/ULTIMA の is_const_unknown をチェック
		difficultyName, exists := difficultyNamesByID[c.DifficultyID]
		if exists && c.IsConstUnknown && (difficultyName == difficultyNameMaster || difficultyName == difficultyNameUltima) {
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
func ApplyAggregation(song *entity.Song, difficultyNamesByID map[int]string) {
	agg := AggregateSongCharts(song.Charts, difficultyNamesByID)
	song.MaxChartConst = agg.MaxChartConst
	song.IsMaxOPUnknown = agg.IsMaxOPUnknown
}
