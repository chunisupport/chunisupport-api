package api_internal

import "time"

type PlayerFavoriteSongRequest struct {
	DisplayID string `json:"display_id" validate:"required"`
}

type PlayerFavoriteSongResponseItem struct {
	DisplayID   string    `json:"display_id"`
	Title       string    `json:"title"`
	Jacket      *string   `json:"jacket"`
	FavoritedAt time.Time `json:"favorited_at"`
}

type PlayerFavoriteSongsResponse struct {
	Items []*PlayerFavoriteSongResponseItem `json:"items"`
}
