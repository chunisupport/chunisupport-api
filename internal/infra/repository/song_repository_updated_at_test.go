package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindLatestUpdatedAt_ReturnsMaxAcrossSongRelatedTables(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS worldsend_charts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			song_id INTEGER NOT NULL,
			level_star INTEGER,
			attribute TEXT,
			notes INTEGER,
			notes_designer TEXT,
			updated_at TEXT,
			FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
		)
	`)
	require.NoError(t, err)

	songUpdatedAt := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	chartUpdatedAt := time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC)
	worldsendChartUpdatedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)

	_, err = db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted, updated_at)
		VALUES
			(1, 'DISPLAY001', 'Song 1', 'Artist 1', 1, 180, NULL, 'IDX001', NULL, 0, 0, ?),
			(2, 'WORLD001', 'World Song', 'Artist W', 1, 200, NULL, 'IDX002', NULL, 1, 0, ?)
	`, songUpdatedAt, songUpdatedAt.Add(-time.Hour))
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer, updated_at)
		VALUES
			(1, 4, 13.8, 0, 1050, '譜面作者', ?)
	`, chartUpdatedAt)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (song_id, level_star, attribute, notes, notes_designer, updated_at)
		VALUES
			(2, 3, '光', 999, 'WORLD''S END譜面作者', ?)
	`, worldsendChartUpdatedAt)
	require.NoError(t, err)

	repo := &songRepository{db: db}

	result, err := repo.FindLatestUpdatedAt(context.Background(), db)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, worldsendChartUpdatedAt.Equal(*result))
}

func TestFindLatestUpdatedAt_ReturnsNilWhenNoUpdatedAtExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS worldsend_charts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			song_id INTEGER NOT NULL,
			level_star INTEGER,
			attribute TEXT,
			notes INTEGER,
			notes_designer TEXT,
			updated_at TEXT,
			FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
		)
	`)
	require.NoError(t, err)

	repo := &songRepository{db: db}

	result, err := repo.FindLatestUpdatedAt(context.Background(), db)

	require.NoError(t, err)
	assert.Nil(t, result)
}
