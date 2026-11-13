package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/info"
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
		CREATE TABLE player_worldsend_records (
			player_id INTEGER NOT NULL, worldsend_chart_id INTEGER NOT NULL, score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL, updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, worldsend_chart_id)
		);
		CREATE TABLE player_worldsend_record_histories (
			player_id INTEGER NOT NULL, worldsend_chart_id INTEGER NOT NULL, score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL, updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, worldsend_chart_id, updated_at)
		);
	`)
	require.NoError(t, err)
	return &scoreHistoryRepository{db: db}, func() { require.NoError(t, db.Close()) }
}

func TestScoreHistoryRepository_FindStandardTimeline(t *testing.T) {
	// Given: 現行値は履歴のどれよりも古い日時を持つ。
	repo, cleanup := setupScoreHistoryTestDB(t)
	defer cleanup()
	base := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	_, err := repo.db.Exec(`INSERT INTO player_records VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, 10, 990000, 5, 3, 2, base)
	require.NoError(t, err)
	_, err = repo.db.Exec(`INSERT INTO player_record_histories VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, 10, 980000, 4, 2, 1, base.Add(time.Hour))
	require.NoError(t, err)
	_, err = repo.db.Exec(`INSERT INTO player_record_histories VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, 10, 970000, 3, 1, 1, base.Add(2*time.Hour))
	require.NoError(t, err)

	// When
	entries, err := repo.FindStandardTimeline(context.Background(), 1, 10)

	// Then: 日時が最も古くても現行値が先頭に来る。
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, 990000, entries[0].Score)
	assert.Equal(t, 5, entries[0].ClearLampID)
	assert.Equal(t, 3, entries[0].ComboLampID)
	assert.Equal(t, 2, entries[0].FullChainID)
	assert.True(t, entries[0].UpdatedAt.Equal(base))
	assert.Equal(t, 970000, entries[1].Score)
	assert.Equal(t, 3, entries[1].ClearLampID)
	assert.Equal(t, 1, entries[1].ComboLampID)
	assert.Equal(t, 1, entries[1].FullChainID)
	assert.True(t, entries[1].UpdatedAt.Equal(base.Add(2*time.Hour)))
	assert.Equal(t, 980000, entries[2].Score)
	assert.Equal(t, 4, entries[2].ClearLampID)
	assert.Equal(t, 2, entries[2].ComboLampID)
	assert.Equal(t, 1, entries[2].FullChainID)
	assert.True(t, entries[2].UpdatedAt.Equal(base.Add(time.Hour)))
}

func TestScoreHistoryRepository_FindStandardTimeline_上限を超えたら最も古い履歴を切り捨てる(t *testing.T) {
	// Given: 上限51件より1件多い52件（現行値1件と履歴51件）を用意する。現行値が最も古い。
	repo, cleanup := setupScoreHistoryTestDB(t)
	defer cleanup()
	base := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	_, err := repo.db.Exec(`INSERT INTO player_records VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, 10, 990000, 5, 3, 2, base)
	require.NoError(t, err)
	for i := range 51 {
		_, err := repo.db.Exec(`INSERT INTO player_record_histories VALUES (?, ?, ?, ?, ?, ?, ?)`,
			1, 10, 900000+i, 4, 2, 1, base.Add(time.Duration(i+1)*time.Second))
		require.NoError(t, err, fmt.Sprintf("履歴%d件目", i+1))
	}

	// When
	entries, err := repo.FindStandardTimeline(context.Background(), 1, 10)

	// Then: 上限51件だけが返り、最も古い履歴が切り捨てられる。
	require.NoError(t, err)
	require.Len(t, entries, 51)
	assert.Equal(t, 990000, entries[0].Score)
	assert.Equal(t, 5, entries[0].ClearLampID)
	assert.Equal(t, 3, entries[0].ComboLampID)
	assert.Equal(t, 2, entries[0].FullChainID)
	assert.True(t, entries[0].UpdatedAt.Equal(base))
	for i := range 50 {
		entry := entries[i+1]
		assert.Equal(t, 900050-i, entry.Score, fmt.Sprintf("%d件目", i+1))
		assert.Equal(t, 4, entry.ClearLampID, fmt.Sprintf("%d件目", i+1))
		assert.Equal(t, 2, entry.ComboLampID, fmt.Sprintf("%d件目", i+1))
		assert.Equal(t, 1, entry.FullChainID, fmt.Sprintf("%d件目", i+1))
		assert.True(t, entry.UpdatedAt.Equal(base.Add(time.Duration(51-i)*time.Second)), fmt.Sprintf("%d件目", i+1))
	}
	for _, entry := range entries {
		assert.NotEqual(t, 900000, entry.Score)
	}
}

func TestScoreHistoryRepository_FindWorldsendTimeline_現行値を先頭に返す(t *testing.T) {
	// Given: 上限51件より1件多い52件（現行値1件と履歴51件）を用意する。現行値が最も古い。
	repo, cleanup := setupScoreHistoryTestDB(t)
	defer cleanup()
	base := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	_, err := repo.db.Exec(`INSERT INTO player_worldsend_records VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, 20, 880000, 5, 3, 2, base)
	require.NoError(t, err)
	for i := range 51 {
		_, err := repo.db.Exec(`INSERT INTO player_worldsend_record_histories VALUES (?, ?, ?, ?, ?, ?, ?)`,
			1, 20, 800000+i, 4, 2, 1, base.Add(time.Duration(i+1)*time.Second))
		require.NoError(t, err, fmt.Sprintf("履歴%d件目", i+1))
	}

	// When
	entries, err := repo.FindWorldsendTimeline(context.Background(), 1, 20)

	// Then: 日時が最も古くても現行値が先頭に来て、上限51件だけが返る。
	require.NoError(t, err)
	require.Len(t, entries, 51)
	assert.Equal(t, 880000, entries[0].Score)
	assert.Equal(t, 5, entries[0].ClearLampID)
	assert.Equal(t, 3, entries[0].ComboLampID)
	assert.Equal(t, 2, entries[0].FullChainID)
	assert.True(t, entries[0].UpdatedAt.Equal(base))
	for i := range 50 {
		entry := entries[i+1]
		assert.Equal(t, 800050-i, entry.Score, fmt.Sprintf("%d件目", i+1))
		assert.Equal(t, 4, entry.ClearLampID, fmt.Sprintf("%d件目", i+1))
		assert.Equal(t, 2, entry.ComboLampID, fmt.Sprintf("%d件目", i+1))
		assert.Equal(t, 1, entry.FullChainID, fmt.Sprintf("%d件目", i+1))
		assert.True(t, entry.UpdatedAt.Equal(base.Add(time.Duration(51-i)*time.Second)), fmt.Sprintf("%d件目", i+1))
	}
	for _, entry := range entries {
		assert.NotEqual(t, 800000, entry.Score)
	}
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
	for i := range info.MaxScoreHistoryEntriesPerChart + 2 {
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
