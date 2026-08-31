package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
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
			official_overpower REAL NOT NULL, official_overpower_percent REAL NULL,
			data_collected_at TIMESTAMP NOT NULL
		);
		CREATE TABLE player_metric_histories (
			player_id INTEGER NOT NULL, official_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, official_overpower_percent REAL NULL,
			data_collected_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, data_collected_at)
		);
	`)
	require.NoError(t, err)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	_, err = db.Exec(`INSERT INTO players VALUES (?, ?, ?, ?, ?)`, 1, 17.25, 12345.67, 98.76, now)
	require.NoError(t, err)
	repo := &playerMetricHistoryQueryService{db: db}
	require.NoError(t, insertPlayerMetricHistory(context.Background(), db, entity.PlayerMetricHistoryEntry{
		PlayerID: 1, OfficialRating: 17.24, OfficialOverpower: 12340.12, OfficialOverpowerPercent: nil, DataCollectedAt: now.Add(-time.Hour),
	}))

	entries, err := repo.FindTimeline(context.Background(), 1)

	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, 17.25, entries[0].OfficialRating)
	assert.Equal(t, 17.24, entries[1].OfficialRating)
	require.NotNil(t, entries[0].OfficialOverpowerPercent)
	assert.Equal(t, 98.76, *entries[0].OfficialOverpowerPercent)
	assert.Nil(t, entries[1].OfficialOverpowerPercent)
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
			official_overpower REAL NOT NULL, official_overpower_percent REAL NULL,
			data_collected_at TIMESTAMP NULL
		);
		CREATE TABLE player_metric_histories (
			player_id INTEGER NOT NULL, official_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, official_overpower_percent REAL NULL,
			data_collected_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, data_collected_at)
		);
		INSERT INTO player_metric_histories VALUES (1, 17.24, 12340.12, NULL, '2026-08-08 11:00:00');
	`)
	require.NoError(t, err)
	repo := &playerMetricHistoryQueryService{db: db}

	entries, err := repo.FindTimeline(context.Background(), 1)

	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestPrunePlayerMetricHistories_上限超過時は古い履歴から削除する(t *testing.T) {
	db := setupTestDB(t)
	defer func() { require.NoError(t, db.Close()) }()
	_, err := db.Exec(`
		CREATE TABLE player_metric_histories (
			player_id INTEGER NOT NULL, official_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, official_overpower_percent REAL NULL,
			data_collected_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, data_collected_at)
		);
	`)
	require.NoError(t, err)
	oldest := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < info.MaxMetricHistoryEntriesPerPlayer+2; i++ {
		_, err := db.Exec(`INSERT INTO player_metric_histories VALUES (?, ?, ?, ?, ?)`,
			1, float64(i), float64(i), nil, oldest.Add(time.Duration(i)*time.Second))
		require.NoError(t, err, fmt.Sprintf("履歴%d件目", i+1))
	}

	err = prunePlayerMetricHistories(context.Background(), db, "sqlite", 1)

	require.NoError(t, err)
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM player_metric_histories WHERE player_id = 1`))
	assert.Equal(t, info.MaxMetricHistoryEntriesPerPlayer, count)
	var retainedOldest time.Time
	require.NoError(t, db.Get(&retainedOldest, `SELECT data_collected_at FROM player_metric_histories WHERE player_id = 1 ORDER BY data_collected_at ASC LIMIT 1`))
	assert.Equal(t, oldest.Add(2*time.Second), retainedOldest)
}

func TestPrunePlayerMetricHistories_上限以下では削除しない(t *testing.T) {
	db := setupTestDB(t)
	defer func() { require.NoError(t, db.Close()) }()
	_, err := db.Exec(`
		CREATE TABLE player_metric_histories (
			player_id INTEGER NOT NULL, official_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, official_overpower_percent REAL NULL,
			data_collected_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, data_collected_at)
		);
		INSERT INTO player_metric_histories VALUES (1, 17.24, 12340.12, NULL, '2026-08-08 11:00:00');
	`)
	require.NoError(t, err)

	err = prunePlayerMetricHistories(context.Background(), db, "sqlite", 1)

	require.NoError(t, err)
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM player_metric_histories WHERE player_id = 1`))
	assert.Equal(t, 1, count)
}

func TestPlayerMetricHistoryRepository_FindTimeline_現行値込みの上限まで返す(t *testing.T) {
	db := setupTestDB(t)
	defer func() { require.NoError(t, db.Close()) }()
	_, err := db.Exec(`
		CREATE TABLE players (
			id INTEGER PRIMARY KEY, official_player_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, official_overpower_percent REAL NULL,
			data_collected_at TIMESTAMP NOT NULL
		);
		CREATE TABLE player_metric_histories (
			player_id INTEGER NOT NULL, official_rating REAL NOT NULL,
			official_overpower REAL NOT NULL, official_overpower_percent REAL NULL,
			data_collected_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, data_collected_at)
		);
	`)
	require.NoError(t, err)
	current := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	_, err = db.Exec(`INSERT INTO players VALUES (?, ?, ?, ?, ?)`, 1, 17.25, 12345.67, 98.76, current)
	require.NoError(t, err)
	for i := 0; i < info.MaxMetricHistoryEntriesPerPlayer+2; i++ {
		_, err := db.Exec(`INSERT INTO player_metric_histories VALUES (?, ?, ?, ?, ?)`,
			1, float64(i), float64(i), nil, current.Add(-time.Duration(i+1)*time.Second))
		require.NoError(t, err, fmt.Sprintf("履歴%d件目", i+1))
	}
	repo := &playerMetricHistoryQueryService{db: db}

	entries, err := repo.FindTimeline(context.Background(), 1)

	require.NoError(t, err)
	require.Len(t, entries, info.MaxMetricHistoryEntriesPerPlayer+1)
	assert.Equal(t, current, entries[0].DataCollectedAt)
	assert.Equal(t, current.Add(-time.Duration(info.MaxMetricHistoryEntriesPerPlayer)*time.Second), entries[len(entries)-1].DataCollectedAt)
}
