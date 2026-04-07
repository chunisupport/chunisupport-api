package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindByDisplayID_ScansLevelStarValueObject(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES (101, 1, 4, '狂', 1200, '譜面作者A')
	`)
	require.NoError(t, err)

	repo := &worldsendChartRepository{db: db}

	got, err := repo.FindByDisplayID(context.Background(), db, "WE001")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Chart)
	require.NotNil(t, got.Chart.LevelStar)
	assert.Equal(t, 4, got.Chart.LevelStar.Int())
	require.NotNil(t, got.Chart.NotesDesigner)
	assert.Equal(t, "譜面作者A", *got.Chart.NotesDesigner)
}

func TestFindByDisplayID_ScansNilLevelStarAsNil(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES (101, 1, NULL, '狂', 1200, NULL)
	`)
	require.NoError(t, err)

	repo := &worldsendChartRepository{db: db}

	got, err := repo.FindByDisplayID(context.Background(), db, "WE001")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Chart)
	assert.Nil(t, got.Chart.LevelStar)
}
