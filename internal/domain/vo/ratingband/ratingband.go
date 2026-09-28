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

// Contains は指定したベスト枠平均レーティングがこの帯に含まれるかを返します。
// 下限は包含、上限は除外とし、下限・上限のない帯（ALL など）は該当側を無制限として扱います。
func (b RatingBand) Contains(rating float64) bool {
	if b.MinInclusive != nil && rating < *b.MinInclusive {
		return false
	}
	return b.MaxExclusive == nil || rating < *b.MaxExclusive
}
