package entity

import "time"

// TemporaryPlayerData は未ログイン時に一時保存するプレイヤーデータです。
type TemporaryPlayerData struct {
	Token     string
	IPAddress string
	Payload   []byte
	BodyHash  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// IsExpired は指定時刻時点で有効期限切れかを返します。
func (t *TemporaryPlayerData) IsExpired(now time.Time) bool {
	if t == nil {
		return true
	}
	return !now.Before(t.ExpiresAt)
}
