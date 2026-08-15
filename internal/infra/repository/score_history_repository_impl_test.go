package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScoreHistoryTestDB(t *testing.T) (*scoreHistoryRepository, func()) {
	db := setupTestDB(t)
	_, err := db.Exec(`
		CREATE TABLE player_records (
			player_id INTEGER NOT NULL, chart_id INTEGER NOT NULL, score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL, updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, chart_id)
		);
		CREATE TABLE player_record_histories (
			player_id INTEGER NOT NULL, chart_id INTEGER NOT NULL, score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL, updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, chart_id, updated_at)
		);
	`)
	require.NoError(t, err)
	return &scoreHistoryRepository{db: db}, func() { require.NoError(t, db.Close()) }
}

func TestScoreHistoryRepository_FindStandardTimeline(t *testing.T) {
	repo, cleanup := setupScoreHistoryTestDB(t)
	defer cleanup()
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	_, err := repo.db.Exec(`INSERT INTO player_records VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, 10, 1000000, 5, 3, 2, now)
	require.NoError(t, err)
	_, err = repo.db.Exec(`INSERT INTO player_record_histories VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, 10, 950000, 4, 2, 1, now.Add(-time.Hour))
	require.NoError(t, err)
	_, err = repo.db.Exec(`INSERT INTO player_record_histories VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, 10, 900000, 3, 1, 1, now.Add(-2*time.Hour))
	require.NoError(t, err)

	entries, err := repo.FindStandardTimeline(context.Background(), 1, 10)

	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, []int{1000000, 950000, 900000}, []int{entries[0].Score, entries[1].Score, entries[2].Score})
}

func TestScoreHistoryRepository_FindStandardTimeline_現行値がなければ履歴を返さない(t *testing.T) {
	repo, cleanup := setupScoreHistoryTestDB(t)
	defer cleanup()
	_, err := repo.db.Exec(`INSERT INTO player_record_histories VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, 10, 900000, 3, 1, 1, time.Now().UTC())
	require.NoError(t, err)

	entries, err := repo.FindStandardTimeline(context.Background(), 1, 10)

	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestScoreHistoryRepository_PruneStandardOverLimit(t *testing.T) {
	repo, cleanup := setupScoreHistoryTestDB(t)
	defer cleanup()
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	for i := 0; i < info.MaxScoreHistoryEntriesPerChart+2; i++ {
		_, err := repo.db.Exec(`INSERT INTO player_record_histories VALUES (?, ?, ?, ?, ?, ?, ?)`,
			1, 10, 900000+i, 1, 1, 1, now.Add(time.Duration(i)*time.Second))
		require.NoError(t, err, fmt.Sprintf("履歴%d件目", i+1))
	}

	err := repo.PruneStandardOverLimit(context.Background(), repo.db, 1, []int{10})

	require.NoError(t, err)
	var count int
	require.NoError(t, repo.db.Get(&count, `SELECT COUNT(*) FROM player_record_histories WHERE player_id = 1 AND chart_id = 10`))
	assert.Equal(t, info.MaxScoreHistoryEntriesPerChart, count)
	var oldestScore int
	require.NoError(t, repo.db.Get(&oldestScore, `SELECT score FROM player_record_histories ORDER BY updated_at ASC LIMIT 1`))
	assert.Equal(t, 900002, oldestScore)
}

func TestWrapScoreHistoryInsertError_主キー重複を識別可能なエラーへ変換する(t *testing.T) {
	err := wrapScoreHistoryInsertError(&mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry"})

	require.Error(t, err)
	assert.True(t, errors.Is(err, domainrepo.ErrScoreHistoryTimestampConflict))
}
