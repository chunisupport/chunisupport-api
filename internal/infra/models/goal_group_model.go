package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// GoalGroupModel はデータベース用のGoalGroupモデルです。
type GoalGroupModel struct {
	ID        uint32    `db:"id"`
	UserID    int       `db:"user_id"`
	Name      string    `db:"name"`
	SortOrder uint16    `db:"sort_order"`
	CreatedAt time.Time `db:"created_at"`
}

// ToEntity は永続化モデルを検証済みのドメインエンティティへ変換します。
func (m *GoalGroupModel) ToEntity() (*entity.GoalGroup, error) {
	group, err := entity.NewGoalGroup(m.UserID, m.Name)
	if err != nil {
		return nil, err
	}
	group.ID = m.ID
	group.SortOrder = m.SortOrder
	group.CreatedAt = m.CreatedAt
	return group, nil
}

// FromGoalGroupEntity はドメインエンティティを永続化モデルへ変換します。
func FromGoalGroupEntity(group *entity.GoalGroup) *GoalGroupModel {
	return &GoalGroupModel{
		ID:        group.ID,
		UserID:    group.UserID,
		Name:      group.Name.String(),
		SortOrder: group.SortOrder,
		CreatedAt: group.CreatedAt,
	}
}
