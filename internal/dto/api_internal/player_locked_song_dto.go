package api_internal

type PlayerLockedSongRequest struct {
	DisplayID string `json:"display_id" validate:"required"`
	IsUltima  bool   `json:"is_ultima"`
}

type PlayerLockedSongUnlockRequest struct {
	DisplayID string `param:"displayid" validate:"required"`
	IsUltima  bool   `query:"is_ultima"`
}

type PlayerLockedSongResponseItem struct {
	DisplayID string `json:"display_id"`
	Title     string `json:"title"`
	IsUltima  bool   `json:"is_ultima"`
}

type PlayerLockedSongsResponse struct {
	Items []*PlayerLockedSongResponseItem `json:"items"`
}
