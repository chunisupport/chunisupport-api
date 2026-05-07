package api_internal

import "strconv"

type strictBool bool

func (b *strictBool) UnmarshalParam(param string) error {
	parsed, err := strconv.ParseBool(param)
	if err != nil {
		return err
	}
	*b = strictBool(parsed)
	return nil
}

type PlayerLockedSongRequest struct {
	DisplayID string `json:"display_id" validate:"required"`
	IsUltima  bool   `json:"is_ultima"`
}

type PlayerLockedSongUnlockRequest struct {
	DisplayID string     `param:"displayid" validate:"required"`
	IsUltima  strictBool `query:"is_ultima"`
}

type PlayerLockedSongResponseItem struct {
	DisplayID string `json:"display_id"`
	Title     string `json:"title"`
	IsUltima  bool   `json:"is_ultima"`
}

type PlayerLockedSongsResponse struct {
	Items []*PlayerLockedSongResponseItem `json:"items"`
}
