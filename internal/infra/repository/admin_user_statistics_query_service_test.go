package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestAdminUserStatisticsQueryService_Get(t *testing.T) {
	// Given
	db := setupAdminUserStatisticsQueryServiceDB(t)
	queryService := NewAdminUserStatisticsQueryService(db)
	cutoff := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)

	// When
	result, err := queryService.Get(context.Background(), cutoff)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalUsers)
	assert.Equal(t, 4, result.UsersWithPlayerData)
	assert.Equal(t, 3, result.ActivePlayerDataLast30Days)
}

func TestAdminUserStatisticsQueryService_Get_データがない場合はすべて0(t *testing.T) {
	// Given
	db := setupAdminUserStatisticsQueryServiceDB(t)
	_, err := db.Exec("DELETE FROM players; DELETE FROM users;")
	require.NoError(t, err)
	queryService := NewAdminUserStatisticsQueryService(db)

	// When
	result, err := queryService.Get(context.Background(), time.Now())

	// Then
	require.NoError(t, err)
	assert.Zero(t, result.TotalUsers)
	assert.Zero(t, result.UsersWithPlayerData)
	assert.Zero(t, result.ActivePlayerDataLast30Days)
}

func setupAdminUserStatisticsQueryServiceDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			player_id INTEGER
		);
		CREATE TABLE players (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL UNIQUE,
			data_collected_at DATETIME
		);
		INSERT INTO users (id, player_id) VALUES
			(1, 10),
			(2, 20),
			(3, 30),
			(4, NULL),
			(5, 40);
	`)
	require.NoError(t, err)
	_, err = db.Exec(
		"INSERT INTO players (id, user_id, data_collected_at) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?)",
		10, 1, time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC),
		20, 2, time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		30, 3, time.Date(2026, 7, 2, 11, 59, 59, 0, time.UTC),
		40, 5, nil,
		// users.player_id が未設定でも、期間内に更新されたプレイヤーデータなら3つ目の集計対象に含めます。
		50, 4, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	return db
}
