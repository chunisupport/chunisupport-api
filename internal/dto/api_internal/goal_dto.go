package api_internal

import "time"

// GoalRequest は目標作成・更新リクエストです。
type GoalRequest struct {
	Title             string         `json:"title" validate:"required"`
	AchievementType   string         `json:"achievement_type" validate:"required"`
	AchievementParams map[string]any `json:"achievement_params" validate:"required"`
	Attributes        map[string]any `json:"attributes"`
	Invert            bool           `json:"invert"`
}

// GoalResponse は目標レスポンスです。
type GoalResponse struct {
	ID                uint32         `json:"id"`
	Title             string         `json:"title"`
	AchievementType   string         `json:"achievement_type"`
	AchievementParams map[string]any `json:"achievement_params"`
	Attributes        map[string]any `json:"attributes"`
	Invert            bool           `json:"invert"`
	SortOrder         uint16         `json:"sort_order"`
	CreatedAt         time.Time      `json:"created_at"`
}

// GoalOrderRequest は目標の並び順更新リクエストです。
type GoalOrderRequest struct {
	GoalIDs []uint32 `json:"goal_ids"`
}

// GoalsResponse は目標一覧レスポンスです。
type GoalsResponse struct {
	Goals []*GoalResponse `json:"goals"`
}
