package api_internal

import "time"

// SongUpdatedAtDTO は楽曲関連データの updated_at のみを返す DTO です。
type SongUpdatedAtDTO struct {
	UpdatedAt *time.Time `json:"updated_at"`
}
