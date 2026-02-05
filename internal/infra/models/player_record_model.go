package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

// PlayerRecordModel はデータベース用のPlayerRecordモデルです。
type PlayerRecordModel struct {
	PlayerID    int       `db:"player_id"`
	ChartID     int       `db:"chart_id"`
	Score       uint32    `db:"score"`
	ClearLampID int       `db:"clear_lamp_id"`
	ComboLampID int       `db:"combo_lamp_id"`
	FullChainID int       `db:"full_chain_id"`
	SlotID      int       `db:"slot_id"`
	SlotOrder   *int      `db:"slot_order"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (m *PlayerRecordModel) ToEntity() (*entity.PlayerRecord, error) {
	s, err := score.NewScore(m.Score)
	if err != nil {
		return nil, err
	}

	return &entity.PlayerRecord{
		PlayerID:    m.PlayerID,
		ChartID:     m.ChartID,
		Score:       s,
		ClearLampID: m.ClearLampID,
		ComboLampID: m.ComboLampID,
		FullChainID: m.FullChainID,
		SlotID:      m.SlotID,
		SlotOrder:   m.SlotOrder,
		UpdatedAt:   m.UpdatedAt,
	}, nil
}

// FromPlayerRecordEntity はentity.PlayerRecordをPlayerRecordModelに変換します。
func FromPlayerRecordEntity(e *entity.PlayerRecord) *PlayerRecordModel {
	scoreVal, _ := e.Score.Value()
	return &PlayerRecordModel{
		PlayerID:    e.PlayerID,
		ChartID:     e.ChartID,
		Score:       uint32(scoreVal.(int64)), // #nosec G115 -- Score value is guaranteed to be within uint32 range by domain VO
		ClearLampID: e.ClearLampID,
		ComboLampID: e.ComboLampID,
		FullChainID: e.FullChainID,
		SlotID:      e.SlotID,
		SlotOrder:   e.SlotOrder,
		UpdatedAt:   e.UpdatedAt,
	}
}
