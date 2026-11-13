package entity

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSong_Clone_複製を変更しても元の楽曲と譜面は変わらない(t *testing.T) {
	// Given
	constant, err := chartconstant.NewChartConstant(13.5)
	require.NoError(t, err)
	n, err := notes.NewNotes(1000)
	require.NoError(t, err)
	updatedAt := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	original := &Song{
		ID:            1,
		DisplayID:     "song1",
		Title:         "曲",
		WikiPageTitle: new("wiki"),
		Reading:       new("きょく"),
		GenreID:       new(1),
		BPM:           new(180),
		ReleasedAt:    new(updatedAt),
		Jacket:        new("jacket.png"),
		UpdatedAt:     new(updatedAt),
		Charts: []*Chart{{
			ID:            10,
			SongID:        1,
			DifficultyID:  DifficultyIDMaster,
			Const:         constant,
			Notes:         new(n),
			NotesDesigner: new("作者"),
			UpdatedAt:     new(updatedAt),
		}},
	}
	want := original.Clone()

	// When
	cloned := original.Clone()
	cloned.Title = "変更"
	*cloned.WikiPageTitle = "変更"
	*cloned.Reading = "へんこう"
	*cloned.GenreID = 2
	*cloned.BPM = 1
	*cloned.ReleasedAt = time.Time{}
	*cloned.Jacket = "changed.png"
	*cloned.UpdatedAt = time.Time{}
	cloned.Charts[0].IsConstUnknown = true
	*cloned.Charts[0].Notes = 1
	*cloned.Charts[0].NotesDesigner = "変更"
	*cloned.Charts[0].UpdatedAt = time.Time{}
	cloned.Charts = append(cloned.Charts, &Chart{ID: 11})

	// Then
	assert.Equal(t, want, original)
}

func TestSong_Clone_nilはnilを返す(t *testing.T) {
	// Given
	var song *Song

	// When / Then
	assert.Nil(t, song.Clone())
}

func TestWorldsendSongWithChart_Clone_複製を変更しても元の値は変わらない(t *testing.T) {
	// Given
	star, err := levelstar.NewLevelStar(3)
	require.NoError(t, err)
	n, err := notes.NewNotes(800)
	require.NoError(t, err)
	original := &WorldsendSongWithChart{
		Song: &Song{ID: 2, DisplayID: "we2", Title: "WE", Charts: []*Chart{}},
		Chart: &WorldsendChart{
			ID:            20,
			SongID:        2,
			LevelStar:     new(star),
			Attribute:     new("狂"),
			Notes:         new(n),
			NotesDesigner: new("作者"),
		},
	}
	want := original.Clone()

	// When
	cloned := original.Clone()
	cloned.Song.Title = "変更"
	*cloned.Chart.LevelStar = 1
	*cloned.Chart.Attribute = "光"
	*cloned.Chart.Notes = 1
	*cloned.Chart.NotesDesigner = "変更"

	// Then
	assert.Equal(t, want, original)
}
