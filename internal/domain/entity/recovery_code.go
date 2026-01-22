package entity

import "time"

// RecoveryCode はリカバリーコードを表すエンティティです。
type RecoveryCode struct {
	ID        uint32
	UserID    int
	CodeHash  []byte
	CreatedAt time.Time
}
