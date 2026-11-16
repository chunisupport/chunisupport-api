package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// PlayerRecordHistoryModel は通常譜面履歴の永続化モデルです。
type PlayerRecordHistoryModel struct {
	PlayerID    int       `db:"player_id"`
	ChartID     int       `db:"chart_id"`
	Score       int       `db:"score"`
	ClearLampID int       `db:"clear_lamp_id"`
	ComboLampID int       `db:"combo_lamp_id"`
	FullChainID int       `db:"full_chain_id"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// PlayerRecordHistoryModelFromEntity は通常譜面の履歴状態を永続化モデルへ変換します。
func PlayerRecordHistoryModelFromEntity(row repository.PlayerRecordHistory) PlayerRecordHistoryModel {
	return PlayerRecordHistoryModel{
		PlayerID: row.PlayerID, ChartID: row.ChartID, Score: row.State.Score,
		ClearLampID: row.State.ClearLampID, ComboLampID: row.State.ComboLampID,
		FullChainID: row.State.FullChainID, UpdatedAt: row.State.UpdatedAt,
	}
}

// PlayerWorldsendRecordHistoryModel はWORLD'S END譜面履歴の永続化モデルです。
type PlayerWorldsendRecordHistoryModel struct {
	PlayerID         int       `db:"player_id"`
	WorldsendChartID int       `db:"worldsend_chart_id"`
	Score            int       `db:"score"`
	ClearLampID      int       `db:"clear_lamp_id"`
	ComboLampID      int       `db:"combo_lamp_id"`
	FullChainID      int       `db:"full_chain_id"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// PlayerWorldsendRecordHistoryModelFromEntity はWORLD'S END履歴状態を永続化モデルへ変換します。
func PlayerWorldsendRecordHistoryModelFromEntity(row repository.PlayerWorldsendRecordHistory) PlayerWorldsendRecordHistoryModel {
	return PlayerWorldsendRecordHistoryModel{
		PlayerID: row.PlayerID, WorldsendChartID: row.WorldsendChartID, Score: row.State.Score,
		ClearLampID: row.State.ClearLampID, ComboLampID: row.State.ComboLampID,
		FullChainID: row.State.FullChainID, UpdatedAt: row.State.UpdatedAt,
	}
}

// PlayerCourseRecordHistoryModel はコース記録履歴の永続化モデルです。
type PlayerCourseRecordHistoryModel struct {
	PlayerID    int       `db:"player_id"`
	CourseID    int       `db:"course_id"`
	Score       int       `db:"score"`
	IsClear     bool      `db:"is_clear"`
	ComboLampID int       `db:"combo_lamp_id"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// PlayerCourseRecordHistoryModelFromEntity はコース記録の履歴状態を永続化モデルへ変換します。
func PlayerCourseRecordHistoryModelFromEntity(row repository.PlayerCourseRecordHistory) PlayerCourseRecordHistoryModel {
	return PlayerCourseRecordHistoryModel{
		PlayerID: row.PlayerID, CourseID: row.CourseID, Score: row.State.Score,
		IsClear: row.State.IsClear, ComboLampID: row.State.ComboLampID, UpdatedAt: row.State.UpdatedAt,
	}
}

// CourseScoreHistoryTimelineModel は現行値を含むコース履歴タイムラインの読み取りモデルです。
type CourseScoreHistoryTimelineModel struct {
	Score       int       `db:"score"`
	IsClear     bool      `db:"is_clear"`
	ComboLampID int       `db:"combo_lamp_id"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// ToEntity は読み取りモデルをドメインエンティティへ変換します。
func (m CourseScoreHistoryTimelineModel) ToEntity() entity.CourseScoreHistoryEntry {
	return entity.CourseScoreHistoryEntry{
		Score: m.Score, IsClear: m.IsClear, ComboLampID: m.ComboLampID, UpdatedAt: m.UpdatedAt,
	}
}

// ScoreHistoryTimelineModel は現行値を含むタイムラインの読み取りモデルです。
type ScoreHistoryTimelineModel struct {
	Score       int       `db:"score"`
	ClearLampID int       `db:"clear_lamp_id"`
	ComboLampID int       `db:"combo_lamp_id"`
	FullChainID int       `db:"full_chain_id"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// ToEntity は読み取りモデルをドメインエンティティへ変換します。
func (m ScoreHistoryTimelineModel) ToEntity() entity.ScoreHistoryEntry {
	return entity.ScoreHistoryEntry{
		Score: m.Score, ClearLampID: m.ClearLampID, ComboLampID: m.ComboLampID,
		FullChainID: m.FullChainID, UpdatedAt: m.UpdatedAt,
	}
}
