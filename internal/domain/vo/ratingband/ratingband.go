package ratingband

// RatingBand はレーティング帯の値オブジェクトです。
// 中身が同一であれば同値として扱います。
type RatingBand struct {
	ID           int
	Label        string
	MinInclusive *float64
	MaxExclusive *float64
	SortOrder    int
}
