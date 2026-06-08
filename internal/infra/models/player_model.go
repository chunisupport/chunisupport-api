package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
)

// PlayerModel はデータベース用のPlayerモデルです。
type PlayerModel struct {
	ID                int        `db:"id"`
	UserID            int        `db:"user_id"`
	Name              string     `db:"player_name"`
	Level             int        `db:"player_level"`
	OfficialRating    *float64   `db:"official_player_rating"`
	CalculatedRating  *float64   `db:"calculated_player_rating"`
	NewAverageRating  *float64   `db:"new_average_rating"`
	BestAverageRating *float64   `db:"best_average_rating"`
	ClassEmblemID     *int       `db:"class_emblem_id"`
	ClassEmblemBaseID *int       `db:"class_emblem_base_id"`
	LastPlayedAt      *time.Time `db:"last_played_at"`
	OverpowerValue    *float64   `db:"overpower_value"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

func (m *PlayerModel) ToEntity() (*entity.Player, error) {
	name, err := playername.NewPlayerName(m.Name)
	if err != nil {
		return nil, err
	}

	return &entity.Player{
		ID:                m.ID,
		UserID:            m.UserID,
		Name:              name,
		Level:             m.Level,
		OfficialRating:    m.OfficialRating,
		CalculatedRating:  m.CalculatedRating,
		NewAverageRating:  m.NewAverageRating,
		BestAverageRating: m.BestAverageRating,
		ClassEmblemID:     m.ClassEmblemID,
		ClassEmblemBaseID: m.ClassEmblemBaseID,
		LastPlayedAt:      m.LastPlayedAt,
		OverpowerValue:    m.OverpowerValue,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}, nil
}

func FromPlayerEntity(e *entity.Player) *PlayerModel {
	return &PlayerModel{
		ID:                e.ID,
		UserID:            e.UserID,
		Name:              e.Name.String(),
		Level:             e.Level,
		OfficialRating:    e.OfficialRating,
		CalculatedRating:  e.CalculatedRating,
		NewAverageRating:  e.NewAverageRating,
		BestAverageRating: e.BestAverageRating,
		ClassEmblemID:     e.ClassEmblemID,
		ClassEmblemBaseID: e.ClassEmblemBaseID,
		LastPlayedAt:      e.LastPlayedAt,
		OverpowerValue:    e.OverpowerValue,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}
