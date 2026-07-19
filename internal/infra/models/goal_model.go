package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// GoalModel はデータベース用のGoalモデルです。
type GoalModel struct {
	ID                uint32    `db:"id"`
	UserID            int       `db:"user_id"`
	GroupID           *uint32   `db:"group_id"`
	Title             string    `db:"title"`
	AchievementTypeID int       `db:"achievement_type_id"`
	AchievementParams []byte    `db:"achievement_params"`
	Attributes        []byte    `db:"attributes"`
	Invert            bool      `db:"invert"`
	SortOrder         uint16    `db:"sort_order"`
	CreatedAt         time.Time `db:"created_at"`
}

func (m *GoalModel) ToEntity() *entity.Goal {
	return &entity.Goal{
		ID:                m.ID,
		UserID:            m.UserID,
		GroupID:           m.GroupID,
		Title:             m.Title,
		AchievementTypeID: m.AchievementTypeID,
		AchievementParams: m.AchievementParams,
		Attributes:        m.Attributes,
		Invert:            m.Invert,
		SortOrder:         m.SortOrder,
		CreatedAt:         m.CreatedAt,
	}
}

func FromGoalEntity(e *entity.Goal) *GoalModel {
	return &GoalModel{
		ID:                e.ID,
		UserID:            e.UserID,
		GroupID:           e.GroupID,
		Title:             e.Title,
		AchievementTypeID: e.AchievementTypeID,
		AchievementParams: e.AchievementParams,
		Attributes:        e.Attributes,
		Invert:            e.Invert,
		SortOrder:         e.SortOrder,
		CreatedAt:         e.CreatedAt,
	}
}
