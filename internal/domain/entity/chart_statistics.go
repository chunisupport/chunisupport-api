package entity

import "time"

// ChartStatistics は譜面の統計情報を表すエンティティです。
// レーティング帯別にランク・ランプの人数を集計します。
// 対象: 譜面定数10.0以上の譜面のみ
type ChartStatistics struct {
	ChartID    int // 譜面ID
	RatingTier int // レーティング帯（10倍した整数: 150-176, 177は17.7+を表す）

	// ランク別人数
	RankS       int // Sランク人数
	RankSPlus   int // S+ランク人数
	RankSS      int // SSランク人数
	RankSSPlus  int // SS+ランク人数
	RankSSS     int // SSSランク人数
	RankSSSPlus int // SSS+ランク人数

	// ランプ別人数
	LampAJ    int // ALL JUSTICE人数
	LampFC    int // FULL COMBO人数
	LampOther int // その他ランプ人数

	// メタデータ
	TotalCount int       // 合計人数（検算用）
	UpdatedAt  time.Time // 更新日時
}

// IsValidRatingTier はレーティング帯が有効な値かを判定します。
// 有効値: 150-176, 177(17.7+)
func (cs *ChartStatistics) IsValidRatingTier() bool {
	return cs.RatingTier >= 150 && cs.RatingTier <= 177
}

// GetRatingTierString はレーティング帯の文字列表現を返します。
// 例: 150 -> "15.0", 177 -> "17.7+"
func (cs *ChartStatistics) GetRatingTierString() string {
	if cs.RatingTier == 177 {
		return "17.7+"
	}
	return formatRatingTier(cs.RatingTier)
}

// formatRatingTier はレーティング帯の整数値を文字列に変換します。
// 例: 150 -> "15.0", 156 -> "15.6"
func formatRatingTier(tier int) string {
	major := tier / 10
	minor := tier % 10
	return string(rune('0'+major/10)) + string(rune('0'+major%10)) + "." + string(rune('0'+minor))
}
