package entity

// RatingBand はレーティング帯のマスタ情報です。
type RatingBand struct {
	ID           int
	Label        string
	MinInclusive *float64
	MaxExclusive *float64
	SortOrder    int
}

// ChartRankStats はランク別の人数統計です。
type ChartRankStats struct {
	AAAL int
	S    int
	SP   int
	SS   int
	SSP  int
	SSS  int
	SSSP int
	Max  int
}

// ChartComboStats はコンボランプ別の人数統計です。
type ChartComboStats struct {
	None int
	FC   int
	AJ   int
}

// ChartStatsByRatingBand は譜面×レーティング帯の統計です。
type ChartStatsByRatingBand struct {
	ChartID      int
	RatingBandID int
	Rank         ChartRankStats
	Combo        ChartComboStats
	Clear        map[string]int
	AverageScore *float64 // レート帯別平均スコア（レコードが0件の場合はnil）
	PlayerCount  int      // レート帯別プレイヤー数
}

// SongChartStats は楽曲の譜面統計レスポンス用エンティティです。
type SongChartStats struct {
	SongID string
	Charts map[string][]*ChartStatsByRatingBand
}
