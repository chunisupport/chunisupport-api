package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/masterfingerprint"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
)

// PlayerModel はデータベース用のPlayerモデルです。
type PlayerModel struct {
	ID                            int                            `db:"id"`
	UserID                        int                            `db:"user_id"`
	Name                          string                         `db:"player_name"`
	Level                         int                            `db:"player_level"`
	OfficialRating                float64                        `db:"official_player_rating"`
	CalculatedRating              *float64                       `db:"calculated_player_rating"`
	NewAverageRating              *float64                       `db:"new_average_rating"`
	BestAverageRating             *float64                       `db:"best_average_rating"`
	ClassEmblemID                 *int                           `db:"class_emblem_id"`
	ClassEmblemBaseID             *int                           `db:"class_emblem_base_id"`
	PossessionID                  int                            `db:"possession_id"`
	TeamCode                      *string                        `db:"team_code"`
	TeamName                      *string                        `db:"team_name"`
	TeamColor                     *string                        `db:"team_color"`
	LastPlayedAt                  *time.Time                     `db:"last_played_at"`
	OverpowerValue                *float64                       `db:"overpower_value"`
	OfficialOverpower             float64                        `db:"official_overpower"`
	OfficialOverpowerPercent      *float64                       `db:"official_overpower_percent"`
	DataCollectedAt               *time.Time                     `db:"data_collected_at"`
	CreatedAt                     time.Time                      `db:"created_at"`
	UpdatedAt                     time.Time                      `db:"updated_at"`
	RecalculatedMasterFingerprint *masterfingerprint.Fingerprint `db:"recalculated_master_fingerprint"`
}

func (m *PlayerModel) ToEntity() (*entity.Player, error) {
	name, err := playername.NewPlayerName(m.Name)
	if err != nil {
		return nil, err
	}

	team, err := m.toPlayerTeam()
	if err != nil {
		return nil, err
	}

	return &entity.Player{
		ID:                            m.ID,
		UserID:                        m.UserID,
		Name:                          name,
		Level:                         m.Level,
		OfficialRating:                m.OfficialRating,
		CalculatedRating:              m.CalculatedRating,
		NewAverageRating:              m.NewAverageRating,
		BestAverageRating:             m.BestAverageRating,
		ClassEmblemID:                 m.ClassEmblemID,
		ClassEmblemBaseID:             m.ClassEmblemBaseID,
		PossessionID:                  m.PossessionID,
		Team:                          team,
		LastPlayedAt:                  m.LastPlayedAt,
		OverpowerValue:                m.OverpowerValue,
		OfficialOverpower:             m.OfficialOverpower,
		OfficialOverpowerPercent:      m.OfficialOverpowerPercent,
		DataCollectedAt:               m.DataCollectedAt,
		CreatedAt:                     m.CreatedAt,
		UpdatedAt:                     m.UpdatedAt,
		RecalculatedMasterFingerprint: m.RecalculatedMasterFingerprint,
	}, nil
}

// toPlayerTeam はチーム列から所属チームを復元します。チームコードがNULLの行はチーム未所属です。
func (m *PlayerModel) toPlayerTeam() (*entity.PlayerTeam, error) {
	if m.TeamCode == nil {
		return nil, nil
	}
	code, err := teamcode.NewTeamCode(*m.TeamCode)
	if err != nil {
		return nil, err
	}
	var name, color string
	if m.TeamName != nil {
		name = *m.TeamName
	}
	if m.TeamColor != nil {
		color = *m.TeamColor
	}
	return entity.NewPlayerTeam(code, name, color)
}

// PlayerTeamColumns は所属チームを players.team_code / team_name / team_color の値へ変換します。
// チーム未所属の場合はすべてNULLです。
func PlayerTeamColumns(team *entity.PlayerTeam) (code, name, color *string) {
	if team == nil {
		return nil, nil, nil
	}
	codeValue := team.Code().String()
	nameValue := team.Name()
	return &codeValue, &nameValue, team.Color()
}

func FromPlayerEntity(e *entity.Player) *PlayerModel {
	teamCode, teamName, teamColor := PlayerTeamColumns(e.Team)
	return &PlayerModel{
		ID:                            e.ID,
		UserID:                        e.UserID,
		Name:                          e.Name.String(),
		Level:                         e.Level,
		OfficialRating:                e.OfficialRating,
		CalculatedRating:              e.CalculatedRating,
		NewAverageRating:              e.NewAverageRating,
		BestAverageRating:             e.BestAverageRating,
		ClassEmblemID:                 e.ClassEmblemID,
		ClassEmblemBaseID:             e.ClassEmblemBaseID,
		PossessionID:                  e.PossessionID,
		TeamCode:                      teamCode,
		TeamName:                      teamName,
		TeamColor:                     teamColor,
		LastPlayedAt:                  e.LastPlayedAt,
		OverpowerValue:                e.OverpowerValue,
		OfficialOverpower:             e.OfficialOverpower,
		OfficialOverpowerPercent:      e.OfficialOverpowerPercent,
		DataCollectedAt:               e.DataCollectedAt,
		CreatedAt:                     e.CreatedAt,
		UpdatedAt:                     e.UpdatedAt,
		RecalculatedMasterFingerprint: e.RecalculatedMasterFingerprint,
	}
}
