package api_internal

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
	ID                int64          `json:"id"`
	Title             string         `json:"title"`
	AchievementType   string         `json:"achievement_type"`
	AchievementParams map[string]any `json:"achievement_params"`
	Attributes        map[string]any `json:"attributes"`
	Invert            bool           `json:"invert"`
	CreatedAt         string         `json:"created_at"`
}

// GoalsResponse は目標一覧レスポンスです。
type GoalsResponse struct {
	Goals []*GoalResponse `json:"goals"`
}
