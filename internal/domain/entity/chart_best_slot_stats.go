package entity

// ChartBestSlotStatsByRatingBand は譜面とベスト枠平均レート帯ごとの採用統計です。
type ChartBestSlotStatsByRatingBand struct {
	ChartID              int
	RatingBandID         int
	BestPlayerCount      int
	EligiblePlayerCount  int
	BestPlayerPercentage *float64
}

// SingleChartBestSlotStats は単一譜面のレート帯別ベスト枠採用統計です。
type SingleChartBestSlotStats struct {
	SongID     string
	Difficulty string
	Stats      []*ChartBestSlotStatsByRatingBand
}
