package service

import (
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

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
//   - IsMaxOPUnknown: MASTER または ULTIMA の譜面に is_const_unknown=true が
//     1件でも含まれれば true。EXPERT以下のunknownは判定対象外。
func AggregateSongCharts(charts []*entity.Chart, difficultyNamesByID map[int]string) SongAggregation {
	var maxConst float64
	isMaxOPUnknown := false

	for _, c := range charts {
		constVal := float64(c.Const)
		if constVal > maxConst {
			maxConst = constVal
		}

		difficultyName, ok := difficultyNamesByID[c.DifficultyID]
		if !ok {
			continue
		}

		if c.IsConstUnknown && isMasterOrUltimaDifficulty(difficultyName) {
			isMaxOPUnknown = true
		}
	}

	return SongAggregation{
		MaxChartConst:  maxConst,
		IsMaxOPUnknown: isMaxOPUnknown,
	}
}

func isMasterOrUltimaDifficulty(difficultyName string) bool {
	name := strings.ToUpper(difficultyName)
	return name == difficultyNameMaster || name == difficultyNameUltima
}

// ApplyAggregation は楽曲エンティティの譜面リストから集約結果を計算し、
// MaxChartConst と IsMaxOPUnknown をエンティティに適用します。
func ApplyAggregation(song *entity.Song, difficultyNamesByID map[int]string) {
	agg := AggregateSongCharts(song.Charts, difficultyNamesByID)
	song.MaxChartConst = agg.MaxChartConst
	song.IsMaxOPUnknown = agg.IsMaxOPUnknown
}
