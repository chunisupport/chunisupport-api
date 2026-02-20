package entity

import "time"

// Goal はユーザーが設定する目標を表します。
type Goal struct {
	ID                uint32
	UserID            int
	Title             string
	AchievementTypeID int
	AchievementType   string
	AchievementParams []byte
	Attributes        []byte
	Invert            bool
	CreatedAt         time.Time
}
