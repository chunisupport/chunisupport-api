package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPlayerRecordRepositoryDB(t *testing.T, db *sqlx.DB) {
	t.Helper()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS player_records (
			player_id INTEGER NOT NULL,
			chart_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL,
			slot_id INTEGER NOT NULL,
			slot_order INTEGER,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (player_id, chart_id)
		);
		CREATE TABLE IF NOT EXISTS player_worldsend_records (
			player_id INTEGER NOT NULL,
			worldsend_chart_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (player_id, worldsend_chart_id)
		);
		CREATE TABLE IF NOT EXISTS player_course_records (
			player_id INTEGER NOT NULL,
			course_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (player_id, course_id)
		);
	`)
	require.NoError(t, err)
}

func TestGetLastScoreUpdate_通常譜面とWORLDSEND譜面の最新時刻を比較して返す(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupPlayerRecordRepositoryDB(t, db)

	recordUpdatedAt := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	worldsendUpdatedAt := recordUpdatedAt.Add(time.Hour)

	_, err := db.Exec(`
		INSERT INTO player_records (
			player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, slot_id, slot_order, updated_at
		) VALUES
			(1, 101, 1000000, 1, 1, 1, 1, NULL, ?),
			(1, 102, 1005000, 1, 1, 1, 1, NULL, ?),
			(2, 201, 990000, 1, 1, 1, 1, NULL, ?)
	`, recordUpdatedAt.Add(-time.Hour), recordUpdatedAt, worldsendUpdatedAt.Add(2*time.Hour))
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO player_worldsend_records (
			player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at
		) VALUES
			(1, 301, 1007500, 1, 1, 1, ?),
			(2, 302, 980000, 1, 1, 1, ?)
	`, worldsendUpdatedAt, recordUpdatedAt.Add(3*time.Hour))
	require.NoError(t, err)

	repo := &playerRecordRepository{db: db}

	result, err := repo.GetLastScoreUpdate(context.Background(), db, 1)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, worldsendUpdatedAt.Equal(*result))
}

func TestGetLastScoreUpdate_通常譜面だけ存在する場合はその最新時刻を返す(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupPlayerRecordRepositoryDB(t, db)

	recordUpdatedAt := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)

	_, err := db.Exec(`
		INSERT INTO player_records (
			player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, slot_id, slot_order, updated_at
		) VALUES
			(1, 101, 1000000, 1, 1, 1, 1, NULL, ?),
			(1, 102, 1005000, 1, 1, 1, 1, NULL, ?)
	`, recordUpdatedAt.Add(-time.Hour), recordUpdatedAt)
	require.NoError(t, err)

	repo := &playerRecordRepository{db: db}

	result, err := repo.GetLastScoreUpdate(context.Background(), db, 1)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, recordUpdatedAt.Equal(*result))
}

func TestGetLastScoreUpdate_レコードが存在しない場合はnilを返す(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupPlayerRecordRepositoryDB(t, db)

	repo := &playerRecordRepository{db: db}

	result, err := repo.GetLastScoreUpdate(context.Background(), db, 1)

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPlayerRecordRepository_FindByPlayerIDAndSongDisplayID_指定楽曲だけを返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupPlayerRecordRepositoryDB(t, db)
	_, err := db.Exec(`
		ALTER TABLE difficulties ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;
		CREATE TABLE clear_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE full_chain_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE slots (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		INSERT INTO clear_lamp_types VALUES (1, 'CLEAR');
		INSERT INTO combo_lamp_types VALUES (1, 'NONE');
		INSERT INTO full_chain_types VALUES (1, 'NONE');
		INSERT INTO slots VALUES (1, 'best');
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx) VALUES
			(1, 'SONG001', '曲1', 'artist', 1, 'IDX001'),
			(2, 'SONG002', '曲2', 'artist', 1, 'IDX002');
		INSERT INTO charts (id, song_id, difficulty_id, const) VALUES
			(101, 1, 4, 14.0),
			(102, 2, 4, 14.5);
		INSERT INTO player_records
			(player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, slot_id, updated_at)
		VALUES
			(10, 101, 1000000, 1, 1, 1, 1, CURRENT_TIMESTAMP),
			(10, 102, 1005000, 1, 1, 1, 1, CURRENT_TIMESTAMP);
	`)
	require.NoError(t, err)

	// When
	records, err := (&playerRecordRepository{db: db}).FindByPlayerIDAndSongDisplayID(context.Background(), db, 10, "SONG001")

	// Then
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "SONG001", records[0].Song.DisplayID)
}
