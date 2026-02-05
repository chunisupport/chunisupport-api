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

// ChartClearStats はクリアランプ別の人数統計です。
type ChartClearStats struct {
	Failed      int
	Clear       int
	Hard        int
	Brave       int
	Absolute    int
	Catastrophy int
}

// ChartStatsByRatingBand は譜面×レーティング帯の統計です。
type ChartStatsByRatingBand struct {
	ChartID      int
	RatingBandID int
	Rank         ChartRankStats
	Combo        ChartComboStats
	Clear        ChartClearStats
	AverageScore *float64 // レート帯別平均スコア（レコードが0件の場合はnil）
	PlayerCount  int      // レート帯別プレイヤー数
}

// SongChartStats は楽曲の譜面統計レスポンス用エンティティです。
//
// Charts のキーは難易度名（"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"）
// または "WORLD'S END" です。難易度IDではなく名前を使用する理由は、
// WORLD'S END譜面には通常の難易度IDが存在しないためです。
// 値のスライスはRatingBandのSortOrder順にソートされている必要があります。
type SongChartStats struct {
	SongID string
	Charts map[string][]*ChartStatsByRatingBand
}

// SingleChartStats は単一難易度の譜面統計レスポンス用エンティティです。
// 難易度別APIで使用されます。
type SingleChartStats struct {
	SongID     string
	Difficulty string
	Stats      []*ChartStatsByRatingBand
}
