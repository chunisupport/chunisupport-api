package models

import (
	"strconv"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// SongModel はデータベース用のSongモデルです。
type SongModel struct {
	ID          int        `db:"id"`
	DisplayID   string     `db:"display_id"`
	Title       string     `db:"title"`
	Reading     *string    `db:"reading"`
	Artist      string     `db:"artist"`
	GenreID     *int       `db:"genre_id"`
	BPM         *int       `db:"bpm"`
	ReleasedAt  *time.Time `db:"released_at"`
	OfficialIdx string     `db:"official_idx"`
	Jacket      *string    `db:"jacket"`
	IsWorldsend bool       `db:"is_worldsend"`
	IsNew       bool       `db:"is_new"`
	IsDeleted   bool       `db:"is_deleted"`
	UpdatedAt   *time.Time `db:"updated_at"`
}

// ToEntity はSongModelをentity.Songに変換します。
func (m *SongModel) ToEntity() *entity.Song {
	song := entity.NewSong()
	song.ID = m.ID
	song.DisplayID = m.DisplayID
	song.Title = m.Title
	song.Reading = m.Reading
	song.Artist = m.Artist
	song.GenreID = m.GenreID
	song.BPM = m.BPM
	song.ReleasedAt = m.ReleasedAt
	song.OfficialIdx = m.OfficialIdx
	song.Jacket = m.Jacket
	song.IsWorldsend = m.IsWorldsend
	song.IsNew = m.IsNew
	song.IsDeleted = m.IsDeleted
	song.UpdatedAt = m.UpdatedAt
	return song
}

// FromSongEntity はentity.SongをSongModelに変換します。
func FromSongEntity(e *entity.Song) *SongModel {
	return &SongModel{
		ID:          e.ID,
		DisplayID:   e.DisplayID,
		Title:       e.Title,
		Reading:     e.Reading,
		Artist:      e.Artist,
		GenreID:     e.GenreID,
		BPM:         e.BPM,
		ReleasedAt:  e.ReleasedAt,
		OfficialIdx: e.OfficialIdx,
		Jacket:      e.Jacket,
		IsWorldsend: e.IsWorldsend,
		IsNew:       e.IsNew,
		IsDeleted:   e.IsDeleted,
		UpdatedAt:   e.UpdatedAt,
	}
}

// ChartModel はデータベース用のChartモデルです。
type ChartModel struct {
	ID             int        `db:"id"`
	SongID         int        `db:"song_id"`
	DifficultyID   int        `db:"difficulty_id"`
	Const          float64    `db:"const"`
	IsConstUnknown bool       `db:"is_const_unknown"`
	Notes          *int       `db:"notes"`
	NotesDesigner  *string    `db:"notes_designer"`
	UpdatedAt      *time.Time `db:"updated_at"`
}

func (m *ChartModel) ToEntity() (*entity.Chart, error) {
	chartConst, err := chartconstant.NewChartConstant(m.Const)
	if err != nil {
		return nil, err
	}

	var n *notes.Notes
	if m.Notes != nil {
		notesVal, err := notes.NewNotes(*m.Notes)
		if err != nil {
			return nil, err
		}
		n = &notesVal
	}

	return &entity.Chart{
		ID:             m.ID,
		SongID:         m.SongID,
		DifficultyID:   m.DifficultyID,
		Const:          chartConst,
		IsConstUnknown: m.IsConstUnknown,
		Notes:          n,
		NotesDesigner:  m.NotesDesigner,
		UpdatedAt:      m.UpdatedAt,
	}, nil
}

// FromChartEntity はentity.ChartをChartModelに変換します。
func FromChartEntity(e *entity.Chart) *ChartModel {
	var notesVal *int
	if e.Notes != nil {
		val, _ := e.Notes.Value()
		if val != nil {
			intVal := int(val.(int64))
			notesVal = &intVal
		}
	}

	constVal, _ := e.Const.Value()
	float64Const := 0.0
	if constVal != nil {
		if str, ok := constVal.(string); ok {
			// Value()は文字列として返すのでパースが必要
			if val, err := strconv.ParseFloat(str, 64); err == nil {
				float64Const = val
			}
		}
	}

	return &ChartModel{
		ID:             e.ID,
		SongID:         e.SongID,
		DifficultyID:   e.DifficultyID,
		Const:          float64Const,
		IsConstUnknown: e.IsConstUnknown,
		Notes:          notesVal,
		NotesDesigner:  e.NotesDesigner,
		UpdatedAt:      e.UpdatedAt,
	}
}
