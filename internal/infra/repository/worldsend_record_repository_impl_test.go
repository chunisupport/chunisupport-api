package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWorldsendRecordDB(t *testing.T, db *sqlx.DB) {
	t.Helper()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS clear_lamp_types (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS combo_lamp_types (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS full_chain_types (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS worldsend_charts (
			id INTEGER PRIMARY KEY,
			song_id INTEGER NOT NULL,
			level_star INTEGER,
			attribute TEXT,
			notes INTEGER,
			FOREIGN KEY (song_id) REFERENCES songs(id)
		);
		CREATE TABLE IF NOT EXISTS player_worldsend_records (
			player_id INTEGER NOT NULL,
			worldsend_chart_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL
		);
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO clear_lamp_types (id, name) VALUES (1, 'CLEAR');
		INSERT INTO combo_lamp_types (id, name) VALUES (1, 'NONE');
		INSERT INTO full_chain_types (id, name) VALUES (1, 'NONE');
	`)
	require.NoError(t, err)
}

func TestFindByPlayerID_ScansLevelStarValueObject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupWorldsendRecordDB(t, db)

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0, 0);
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, 4, '狂', 1200);
		INSERT INTO player_worldsend_records (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
		VALUES (10, 101, 900000, 1, 1, 1, ?);
	`, time.Now().UTC())
	require.NoError(t, err)

	repo := &worldsendRecordRepository{db: db}
	records, err := repo.FindByPlayerID(context.Background(), db, 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.NotNil(t, records[0].WorldsendChart)
	require.NotNil(t, records[0].WorldsendChart.LevelStar)
	assert.Equal(t, 4, records[0].WorldsendChart.LevelStar.Int())
}

func TestFindByPlayerID_ScansNilLevelStarAsNil(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupWorldsendRecordDB(t, db)

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0, 0);
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, NULL, '狂', 1200);
		INSERT INTO player_worldsend_records (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
		VALUES (10, 101, 900000, 1, 1, 1, ?);
	`, time.Now().UTC())
	require.NoError(t, err)

	repo := &worldsendRecordRepository{db: db}
	records, err := repo.FindByPlayerID(context.Background(), db, 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.NotNil(t, records[0].WorldsendChart)
	assert.Nil(t, records[0].WorldsendChart.LevelStar)
}
