package repository

import (
	"context"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestUserUpdatedAtQueryService_FindByUsername(t *testing.T) {
	// Given
	db := setupUserUpdatedAtQueryServiceDB(t)
	service := NewUserUpdatedAtQueryService()

	// When
	result, err := service.FindByUsername(context.Background(), db, "tester")

	// Then
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.User)
	assert.Equal(t, "tester", result.User.Username.String())
	require.NotNil(t, result.PlayerUpdatedAt)
	assert.Equal(t, time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC), result.PlayerUpdatedAt.UTC())
	require.NotNil(t, result.RecordsUpdatedAt)
	assert.Equal(t, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), result.RecordsUpdatedAt.UTC())
}

func TestUserUpdatedAtQueryService_FindByUsername_プレイヤー未連携の場合は更新日時がnil(t *testing.T) {
	// Given
	db := setupUserUpdatedAtQueryServiceDB(t)
	_, err := db.Exec(`
		INSERT INTO users (
			id, username, firebase_uid, created_at, updated_at, player_id,
			account_type_id, is_suspicious, is_private
		) VALUES (
			2, 'unlinked', NULL, '2026-07-01 09:00:00', '2026-07-01 09:00:00',
			NULL, 1, 0, 0
		)
	`)
	require.NoError(t, err)
	service := NewUserUpdatedAtQueryService()

	// When
	result, err := service.FindByUsername(context.Background(), db, "unlinked")

	// Then
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Nil(t, result.PlayerUpdatedAt)
	assert.Nil(t, result.RecordsUpdatedAt)
}

func TestUserUpdatedAtQueryService_FindByUsername_存在しないユーザーはエラー(t *testing.T) {
	// Given
	db := setupUserUpdatedAtQueryServiceDB(t)
	service := NewUserUpdatedAtQueryService()

	// When
	result, err := service.FindByUsername(context.Background(), db, "missing")

	// Then
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domainrepo.ErrUserNotFound)
}

func TestUserUpdatedAtQueryService_FindByUsername_レコードがない場合はプレイヤー更新日時のみ返す(t *testing.T) {
	// Given
	db := setupUserUpdatedAtQueryServiceDB(t)
	_, err := db.Exec(`
		DELETE FROM player_records;
		DELETE FROM player_worldsend_records;
	`)
	require.NoError(t, err)
	service := NewUserUpdatedAtQueryService()

	// When
	result, err := service.FindByUsername(context.Background(), db, "tester")

	// Then
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.PlayerUpdatedAt)
	assert.Equal(t, time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC), result.PlayerUpdatedAt.UTC())
	assert.Nil(t, result.RecordsUpdatedAt)
}

func TestUserUpdatedAtQueryService_FindByUsername_WORLDSENDの更新日時が新しい場合はその日時を返す(t *testing.T) {
	// Given
	db := setupUserUpdatedAtQueryServiceDB(t)
	_, err := db.Exec(`
		INSERT INTO player_worldsend_records (player_id, updated_at)
		VALUES (10, '2026-07-01 13:00:00')
	`)
	require.NoError(t, err)
	service := NewUserUpdatedAtQueryService()

	// When
	result, err := service.FindByUsername(context.Background(), db, "tester")

	// Then
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.RecordsUpdatedAt)
	assert.Equal(t, time.Date(2026, 7, 1, 13, 0, 0, 0, time.UTC), result.RecordsUpdatedAt.UTC())
}

func setupUserUpdatedAtQueryServiceDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			firebase_uid TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			player_id INTEGER,
			account_type_id INTEGER NOT NULL,
			is_suspicious INTEGER NOT NULL,
			is_private INTEGER NOT NULL
		);
		CREATE TABLE players (
			id INTEGER PRIMARY KEY,
			updated_at DATETIME NOT NULL
		);
		CREATE TABLE player_records (
			player_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE TABLE player_worldsend_records (
			player_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE INDEX idx_player_records_player_updated_at
			ON player_records(player_id, updated_at DESC);
		CREATE INDEX idx_player_worldsend_records_player_updated_at
			ON player_worldsend_records(player_id, updated_at DESC);

		INSERT INTO users (
			id, username, firebase_uid, created_at, updated_at, player_id,
			account_type_id, is_suspicious, is_private
		) VALUES (
			1, 'tester', NULL, '2026-07-01 09:00:00', '2026-07-01 09:00:00',
			10, 1, 0, 0
		);
		INSERT INTO players (id, updated_at)
		VALUES (10, '2026-07-01 10:00:00');
		INSERT INTO player_records (player_id, updated_at)
		VALUES
			(10, '2026-07-01 11:00:00'),
			(10, '2026-07-01 12:00:00');
		INSERT INTO player_worldsend_records (player_id, updated_at)
		VALUES (10, '2026-07-01 11:30:00');
	`)
	require.NoError(t, err)

	return db
}
