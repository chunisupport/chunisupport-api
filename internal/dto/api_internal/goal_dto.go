package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// GoalDTO は目標レスポンスです。
type GoalDTO struct {
	ID                int            `json:"id"`
	Title             string         `json:"title"`
	AchievementType   string         `json:"achievement_type"`
	AchievementParams map[string]any `json:"achievement_params"`
	Attributes        map[string]any `json:"attributes"`
	Invert            bool           `json:"invert"`
	CreatedAt         string         `json:"created_at"`
}

type UpsertGoalRequestDTO struct {
	Title             string         `json:"title"`
	AchievementType   string         `json:"achievement_type"`
	AchievementParams map[string]any `json:"achievement_params"`
	Attributes        map[string]any `json:"attributes"`
	Invert            bool           `json:"invert"`
}

type GoalListResponseDTO struct {
	Goals []*GoalDTO `json:"goals"`
}

func ToGoalDTO(goal *entity.Goal, achievementTypesByID map[int]string) *GoalDTO {
	return &GoalDTO{
		ID:                goal.ID,
		Title:             goal.Title,
		AchievementType:   achievementTypesByID[goal.AchievementTypeID],
		AchievementParams: goal.AchievementParams,
		Attributes:        goal.Attributes,
		Invert:            goal.Invert,
		CreatedAt:         goal.CreatedAt.Format(time.RFC3339),
	}
}

func ToGoalDTOs(goals []*entity.Goal, achievementTypesByID map[int]string) []*GoalDTO {
	result := make([]*GoalDTO, 0, len(goals))
	for _, goal := range goals {
		result = append(result, ToGoalDTO(goal, achievementTypesByID))
	}
	return result
}
