package entity

import "time"

// Session はユーザーセッション情報を表すエンティティです。
type Session struct {
	ID        string
	UserID    int
	ExpiresAt time.Time
	CreatedAt time.Time
}
