package api_internal

import "time"

// UserUpdatedAtDTO はプレイヤーデータの updated_at のみを返す DTO です。
type UserUpdatedAtDTO struct {
	UpdatedAt time.Time `json:"updated_at"`
}
