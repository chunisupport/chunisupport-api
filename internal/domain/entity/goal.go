package entity

import "time"

// Goal はユーザーの目標を表す集約です。
type Goal struct {
	ID                int
	UserID            int
	Title             string
	AchievementTypeID int
	AchievementParams map[string]any
	Attributes        map[string]any
	Invert            bool
	CreatedAt         time.Time
}
