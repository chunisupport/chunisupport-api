package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// Save の ON DUPLICATE KEY UPDATE は MySQL 固有の構文で SQLite では実行できないため、ここでは読み取りだけを検証します。
func setupXAccountRepositorySQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec(`
		CREATE TABLE user_x_accounts (
			user_id INTEGER NOT NULL PRIMARY KEY,
			x_user_id TEXT NOT NULL,
			x_username TEXT NOT NULL,
			is_public BOOLEAN NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	return db
}

func TestXAccountRepository_FindByUserID_未保存ならnilを返す(t *testing.T) {
	// Given
	ctx := context.Background()
	db := setupXAccountRepositorySQLite(t)
	repo := NewXAccountRepository()

	// When
	account, err := repo.FindByUserID(ctx, db, 10)

	// Then
	require.NoError(t, err)
	assert.Nil(t, account)
}

func TestXAccountRepository_FindByUserID_指定ユーザーの行を返す(t *testing.T) {
	// Given
	ctx := context.Background()
	db := setupXAccountRepositorySQLite(t)
	_, err := db.Exec(`INSERT INTO user_x_accounts (user_id, x_user_id, x_username) VALUES (10, '111', 'chuni'), (20, '222', 'other_user')`)
	require.NoError(t, err)
	repo := NewXAccountRepository()

	// When
	found, err := repo.FindByUserID(ctx, db, 10)

	// Then
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, 10, found.UserID())
	assert.Equal(t, "111", found.XUserID())
	assert.Equal(t, "chuni", found.Username())
}
