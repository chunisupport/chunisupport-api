package api_internal

// CourseRankingResponse はコースの自己ベストランキングを返します。
type CourseRankingResponse struct {
	Course  CourseRankingCourseDTO  `json:"course"`
	Ranking []CourseRankingEntryDTO `json:"ranking"`
	Total   int                     `json:"total"`
}

type CourseRankingCourseDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Class string `json:"class"`
}

type CourseRankingEntryDTO struct {
	CourseScoreHistoryEntry
	Rank       int    `json:"rank"`
	Username   string `json:"username"`
	PlayerName string `json:"player_name"`
	IsSelf     *bool  `json:"is_self,omitempty"`
}
