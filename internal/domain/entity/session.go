package entity

import "time"

// Session はユーザーセッション情報を表すエンティティです。
type Session struct {
	ID        string
	UserID    int
	ExpiresAt time.Time
	CreatedAt time.Time
}

// IsExpired は指定された時刻においてセッションが有効期限切れかどうかを判定します。
// 引数で現在時刻を受け取ることでテスタビリティを確保します。
func (s *Session) IsExpired(now time.Time) bool {
	return s.ExpiresAt.Before(now)
}
