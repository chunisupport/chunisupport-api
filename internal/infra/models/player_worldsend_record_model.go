package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

// PlayerWorldsendRecordModel はデータベース用のプレイヤー WORLD'S END レコードモデルです。
type PlayerWorldsendRecordModel struct {
	PlayerID         int       `db:"player_id"`
	WorldsendChartID int       `db:"worldsend_chart_id"`
	Score            uint32    `db:"score"`
	ClearLampID      int       `db:"clear_lamp_id"`
	ComboLampID      int       `db:"combo_lamp_id"`
	FullChainID      int       `db:"full_chain_id"`
	UpdatedAt        time.Time `db:"updated_at"`
}

func (m *PlayerWorldsendRecordModel) ToEntity() (*entity.PlayerWorldsendRecord, error) {
	s, err := score.NewScore(m.Score)
	if err != nil {
		return nil, err
	}

	return &entity.PlayerWorldsendRecord{
		PlayerID:         m.PlayerID,
		WorldsendChartID: m.WorldsendChartID,
		Score:            s,
		ClearLampID:      m.ClearLampID,
		ComboLampID:      m.ComboLampID,
		FullChainID:      m.FullChainID,
		UpdatedAt:        m.UpdatedAt,
	}, nil
}

func FromPlayerWorldsendRecordEntity(e *entity.PlayerWorldsendRecord) *PlayerWorldsendRecordModel {
	scoreVal, _ := e.Score.Value()
	return &PlayerWorldsendRecordModel{
		PlayerID:         e.PlayerID,
		WorldsendChartID: e.WorldsendChartID,
		Score:            uint32(scoreVal.(int64)), // #nosec G115 -- Score value is guaranteed to be within uint32 range by domain VO
		ClearLampID:      e.ClearLampID,
		ComboLampID:      e.ComboLampID,
		FullChainID:      e.FullChainID,
		UpdatedAt:        e.UpdatedAt,
	}
}
