package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayLogChartCandidateModel はプレイ履歴のジャケット照合結果を表す永続化モデルです。
type PlayLogChartCandidateModel struct {
	ID           int    `db:"id"`
	Title        string `db:"title"`
	Jacket       string `db:"jacket"`
	DifficultyID int    `db:"difficulty_id"`
}

// ToEntity は譜面候補モデルをドメインエンティティへ変換します。
func (m PlayLogChartCandidateModel) ToEntity() entity.PlayLogChartCandidate {
	return entity.PlayLogChartCandidate{
		ID:           m.ID,
		Title:        m.Title,
		Jacket:       m.Jacket,
		DifficultyID: m.DifficultyID,
	}
}

// PlayerPlayLogModel はプレイヤーのプレイ履歴を表す永続化モデルです。
type PlayerPlayLogModel struct {
	PlayerID         int       `db:"player_id"`
	PlayOrder        int       `db:"play_order"`
	ChartID          *int      `db:"chart_id"`
	WorldsendChartID *int      `db:"worldsend_chart_id"`
	Jacket           *string   `db:"jacket_img_id"`
	DifficultyID     *int      `db:"difficulty_id"`
	Score            int       `db:"score"`
	ClearLampID      int       `db:"clear_lamp_id"`
	ComboLampID      int       `db:"combo_lamp_id"`
	FullChainID      int       `db:"full_chain_id"`
	PlayedAt         time.Time `db:"played_at"`
}

// FromPlayerPlayLogEntity はプレイ履歴エンティティを保存モデルへ変換します。
// DATETIME はタイムゾーンを持たないため、保存前にUTCへ正規化します。
func FromPlayerPlayLogEntity(playerID, playOrder int, e entity.PlayerPlayLog) PlayerPlayLogModel {
	return PlayerPlayLogModel{
		PlayerID:         playerID,
		PlayOrder:        playOrder,
		ChartID:          e.ChartID,
		WorldsendChartID: e.WorldsendChartID,
		Jacket:           e.Jacket,
		DifficultyID:     e.DifficultyID,
		Score:            e.Score,
		ClearLampID:      e.ClearLampID,
		ComboLampID:      e.ComboLampID,
		FullChainID:      e.FullChainID,
		PlayedAt:         e.PlayedAt.UTC(),
	}
}
