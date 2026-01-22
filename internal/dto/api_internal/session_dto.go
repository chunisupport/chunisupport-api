package api_internal

// SessionCountDTO はセッション数を表すDTOです。
type SessionCountDTO struct {
	// Count は現在有効なセッション数
	Count int `json:"count"`
}
