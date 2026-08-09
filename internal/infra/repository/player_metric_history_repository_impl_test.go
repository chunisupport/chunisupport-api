package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerMetricHistoryRepository_FindTimeline(t *testing.T) {
	db := setupTestDB(t)
	defer func() { require.NoError(t, db.Close()) }()
	_, err := db.Exec(`
		CREATE TABLE players (
			id INTEGER PRIMARY KEY, official_player_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, data_collected_at TIMESTAMP NOT NULL
		);
		CREATE TABLE player_metric_histories (
			player_id INTEGER NOT NULL, official_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, data_collected_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, data_collected_at)
		);
	`)
	require.NoError(t, err)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	_, err = db.Exec(`INSERT INTO players VALUES (?, ?, ?, ?)`, 1, 17.25, 12345.67, now)
	require.NoError(t, err)
	repo := &playerMetricHistoryQueryService{db: db}
	require.NoError(t, insertPlayerMetricHistory(context.Background(), db, entity.PlayerMetricHistoryEntry{
		PlayerID: 1, OfficialRating: 17.24, OfficialOverpower: 12340.12, DataCollectedAt: now.Add(-time.Hour),
	}))

	entries, err := repo.FindTimeline(context.Background(), 1)

	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, 17.25, entries[0].OfficialRating)
	assert.Equal(t, 17.24, entries[1].OfficialRating)
	assert.Equal(t, now, entries[0].DataCollectedAt)
}

func TestPlayerMetricHistoryRepository_Insert_主キー重複を識別可能なエラーへ変換する(t *testing.T) {
	err := wrapPlayerMetricHistoryInsertError(&mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry"})

	assert.ErrorIs(t, err, domainrepo.ErrPlayerMetricHistoryTimestampConflict)
}

func TestPlayerMetricHistoryRepository_FindTimeline_現行プレイヤーがなければ履歴を返さない(t *testing.T) {
	db := setupTestDB(t)
	defer func() { require.NoError(t, db.Close()) }()
	_, err := db.Exec(`
		CREATE TABLE players (
			id INTEGER PRIMARY KEY, official_player_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, data_collected_at TIMESTAMP NULL
		);
		CREATE TABLE player_metric_histories (
			player_id INTEGER NOT NULL, official_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, data_collected_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, data_collected_at)
		);
		INSERT INTO player_metric_histories VALUES (1, 17.24, 12340.12, '2026-08-08 11:00:00');
	`)
	require.NoError(t, err)
	repo := &playerMetricHistoryQueryService{db: db}

	entries, err := repo.FindTimeline(context.Background(), 1)

	require.NoError(t, err)
	assert.Empty(t, entries)
}
