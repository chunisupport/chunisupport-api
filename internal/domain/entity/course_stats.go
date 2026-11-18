package entity

// CourseComboStats はコースのコンボランプ別の人数統計です。
// コースの理論値は ALL JUSTICE に含め、譜面統計の AJC のような区分は持ちません。
type CourseComboStats struct {
	None int
	FC   int
	AJ   int
}

// CourseClearStats はコースのクリア状態別の人数統計です。
type CourseClearStats struct {
	Failed int
	Clear  int
}

// CourseStatsByRatingBand はコース×レーティング帯の統計です。
// コースには公式のランク区分がないため、スコアランク分布は持ちません。
type CourseStatsByRatingBand struct {
	CourseID     int
	RatingBandID int
	Combo        CourseComboStats
	Clear        CourseClearStats
	AverageScore *float64 // クリア済み記録の平均スコア（クリア済み記録が0件の場合はnil）
	MedianScore  *float64 // クリア済み記録の中央スコア（クリア済み記録が0件の場合はnil）
	PlayerCount  int      // プレイ済み人数
}

// CourseStats はコース統計レスポンス用エンティティです。
// Stats はレーティング帯の表示順に並びます。
type CourseStats struct {
	CourseDisplayID string
	Stats           []*CourseStatsByRatingBand
}
