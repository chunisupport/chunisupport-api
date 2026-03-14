package models

import (
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// PlayerDataSongModel はプレイヤーデータ登録用の楽曲マスタモデルです。
type PlayerDataSongModel struct {
	ID          int        `db:"id"`
	DisplayID   string     `db:"display_id"`
	Title       string     `db:"title"`
	Artist      string     `db:"artist"`
	GenreID     *int       `db:"genre_id"`
	BPM         *int       `db:"bpm"`
	ReleasedAt  *time.Time `db:"released_at"`
	OfficialIdx string     `db:"official_idx"`
	Jacket      *string    `db:"jacket"`
	IsDeleted   bool       `db:"is_deleted"`
}

// ToEntity は PlayerDataSongModel を entity.PlayerDataSong に変換します。
func (m *PlayerDataSongModel) ToEntity() *entity.PlayerDataSong {
	return &entity.PlayerDataSong{
		ID:          m.ID,
		DisplayID:   m.DisplayID,
		Title:       m.Title,
		Artist:      m.Artist,
		GenreID:     m.GenreID,
		BPM:         m.BPM,
		ReleasedAt:  m.ReleasedAt,
		OfficialIdx: m.OfficialIdx,
		Jacket:      m.Jacket,
		IsDeleted:   m.IsDeleted,
	}
}

func FromPlayerDataSongEntity(e *entity.PlayerDataSong) *PlayerDataSongModel {
	return &PlayerDataSongModel{
		ID:          e.ID,
		DisplayID:   e.DisplayID,
		Title:       e.Title,
		Artist:      e.Artist,
		GenreID:     e.GenreID,
		BPM:         e.BPM,
		ReleasedAt:  e.ReleasedAt,
		OfficialIdx: e.OfficialIdx,
		Jacket:      e.Jacket,
		IsDeleted:   e.IsDeleted,
	}
}

// PlayerDataChartModel はプレイヤーデータ登録用の譜面マスタモデルです。
type PlayerDataChartModel struct {
	ID             int          `db:"id"`
	SongID         int          `db:"song_id"`
	DifficultyID   int          `db:"difficulty_id"`
	Const          float64      `db:"const"`
	IsConstUnknown bool         `db:"is_const_unknown"`
	Notes          *notes.Notes `db:"notes"`
}

// ToEntity は PlayerDataChartModel を entity.PlayerDataChart に変換します。
func (m *PlayerDataChartModel) ToEntity() (*entity.PlayerDataChart, error) {
	chartConst, err := chartconstant.NewChartConstant(m.Const)
	if err != nil {
		return nil, fmt.Errorf("invalid chart constant (chart_id=%d): %w", m.ID, err)
	}

	return &entity.PlayerDataChart{
		ID:             m.ID,
		SongID:         m.SongID,
		DifficultyID:   m.DifficultyID,
		Const:          chartConst,
		IsConstUnknown: m.IsConstUnknown,
		Notes:          m.Notes,
	}, nil
}

func FromPlayerDataChartEntity(e *entity.PlayerDataChart) *PlayerDataChartModel {
	return &PlayerDataChartModel{
		ID:             e.ID,
		SongID:         e.SongID,
		DifficultyID:   e.DifficultyID,
		Const:          float64(e.Const),
		IsConstUnknown: e.IsConstUnknown,
		Notes:          e.Notes,
	}
}

// PlayerDataWorldsendChartModel はプレイヤーデータ登録用のWORLD'S END譜面モデルです。
type PlayerDataWorldsendChartModel struct {
	ID     int `db:"id"`
	SongID int `db:"song_id"`
}

// ToEntity は PlayerDataWorldsendChartModel を entity.PlayerDataWorldsendChart に変換します。
func (m *PlayerDataWorldsendChartModel) ToEntity() *entity.PlayerDataWorldsendChart {
	return &entity.PlayerDataWorldsendChart{
		ID:     m.ID,
		SongID: m.SongID,
	}
}

func FromPlayerDataWorldsendChartEntity(e *entity.PlayerDataWorldsendChart) *PlayerDataWorldsendChartModel {
	return &PlayerDataWorldsendChartModel{
		ID:     e.ID,
		SongID: e.SongID,
	}
}
