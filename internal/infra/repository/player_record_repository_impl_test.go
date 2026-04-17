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
