package entity

import "time"

// Goal はユーザーが設定する目標を表します。
type Goal struct {
	ID                uint32
	UserID            int
	GroupID           *uint32
	Title             string
	AchievementTypeID int
	AchievementType   string
	AchievementParams []byte
	Attributes        []byte
	InvertValue       bool
	InvertPercentage  bool
	SortOrder         uint16
	CreatedAt         time.Time
}
