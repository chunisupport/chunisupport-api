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

// NewAPIToken は新規APIトークンを生成し、永続化に必要な初期状態を設定します。
func NewAPIToken(userID int, name string, hashedToken string) *APIToken {
	return &APIToken{
		UserID:      userID,
		Name:        name,
		HashedToken: hashedToken,
		CreatedAt:   time.Now().UTC(),
	}
}
