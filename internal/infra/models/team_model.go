package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
)

// TeamModel はデータベース用のチームモデルです。
type TeamModel struct {
	ID              int       `db:"id"`
	Code            string    `db:"code"`
	Name            string    `db:"name"`
	Color           *string   `db:"color"`
	DataCollectedAt time.Time `db:"data_collected_at"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// ToEntity はDBの値を検証してチームエンティティへ変換します。
func (m *TeamModel) ToEntity() (*entity.Team, error) {
	code, err := teamcode.NewTeamCode(m.Code)
	if err != nil {
		return nil, err
	}
	return &entity.Team{
		ID:              m.ID,
		Code:            code,
		Name:            m.Name,
		Color:           m.Color,
		DataCollectedAt: m.DataCollectedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}, nil
}
