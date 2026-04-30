package entity

import (
	"time"
)

// APIToken は外部APIで利用する永続化トークンを表します。
type APIToken struct {
	ID          int64
	UserID      int
	Name        string
	HashedToken string
	CreatedAt   time.Time
}
