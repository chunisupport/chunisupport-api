package models

import (
	"encoding/json"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// GoalModel はデータベース用のGoalモデルです。
type GoalModel struct {
	ID                int             `db:"id"`
	UserID            int             `db:"user_id"`
	Title             string          `db:"title"`
	AchievementTypeID int             `db:"achievement_type_id"`
	AchievementParams json.RawMessage `db:"achievement_params"`
	Attributes        json.RawMessage `db:"attributes"`
	Invert            bool            `db:"invert"`
	CreatedAt         time.Time       `db:"created_at"`
}

func (m *GoalModel) ToEntity() (*entity.Goal, error) {
	params := map[string]any{}
	if len(m.AchievementParams) > 0 {
		if err := json.Unmarshal(m.AchievementParams, &params); err != nil {
			return nil, err
		}
	}
	attrs := map[string]any{}
	if len(m.Attributes) > 0 {
		if err := json.Unmarshal(m.Attributes, &attrs); err != nil {
			return nil, err
		}
	}
	return &entity.Goal{
		ID:                m.ID,
		UserID:            m.UserID,
		Title:             m.Title,
		AchievementTypeID: m.AchievementTypeID,
		AchievementParams: params,
		Attributes:        attrs,
		Invert:            m.Invert,
		CreatedAt:         m.CreatedAt,
	}, nil
}

func FromGoalEntity(e *entity.Goal) (*GoalModel, error) {
	paramsBytes, err := json.Marshal(e.AchievementParams)
	if err != nil {
		return nil, err
	}
	attrsBytes, err := json.Marshal(e.Attributes)
	if err != nil {
		return nil, err
	}
	return &GoalModel{
		ID:                e.ID,
		UserID:            e.UserID,
		Title:             e.Title,
		AchievementTypeID: e.AchievementTypeID,
		AchievementParams: paramsBytes,
		Attributes:        attrsBytes,
		Invert:            e.Invert,
		CreatedAt:         e.CreatedAt,
	}, nil
}
