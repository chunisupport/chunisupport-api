package api_internal

import "time"

// SongUpdatedAtDTO は公開中の楽曲データの updated_at と変更世代を返す DTO です。
// revision は楽曲・譜面の変更ごとに必ず進むため、フロントのキャッシュ判定に使います。
type SongUpdatedAtDTO struct {
	UpdatedAt *time.Time `json:"updated_at"`
	Revision  uint64     `json:"revision"`
}

// CourseUpdatedAtDTO はコースマスタの updated_at のみを返す DTO です。
type CourseUpdatedAtDTO struct {
	UpdatedAt *time.Time `json:"updated_at"`
}

// UserUpdatedAtDTO はユーザー関連データの updated_at のみを返す DTO です。
type UserUpdatedAtDTO struct {
	UpdatedAt *time.Time `json:"updated_at"`
}
