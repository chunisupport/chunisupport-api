package api_internal

type DataTransferCountsResponse struct {
	Records                  int `json:"records"`
	RecordHistories          int `json:"record_histories"`
	WorldsendRecords         int `json:"worldsend_records"`
	WorldsendRecordHistories int `json:"worldsend_record_histories"`
	MetricHistories          int `json:"metric_histories"`
	CourseRecords            int `json:"course_records"`
	Honors                   int `json:"honors"`
	FavoriteSongs            int `json:"favorite_songs"`
	LockedSongs              int `json:"locked_songs"`
	GoalGroups               int `json:"goal_groups"`
	Goals                    int `json:"goals"`
	RecordFilters            int `json:"record_filters"`
}

type DataTransferValidationResponse struct {
	Importable               bool                       `json:"importable"`
	PlayerName               string                     `json:"player_name"`
	Counts                   DataTransferCountsResponse `json:"counts"`
	Blockers                 []string                   `json:"blockers"`
	UnresolvedReferences     []string                   `json:"unresolved_references"`
	UnresolvedReferenceCount int                        `json:"unresolved_reference_count"`
}

type DataTransferImportResponse struct {
	PlayerID int                        `json:"player_id"`
	Counts   DataTransferCountsResponse `json:"counts"`
}
