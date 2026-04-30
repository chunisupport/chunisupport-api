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

func setupAPITokenRepositorySQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec(`
		CREATE TABLE api_tokens (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			hashed_token TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)

	return db
}

func TestAPITokenRepository_FindByUserID_ReturnsToken(t *testing.T) {
	db := setupAPITokenRepositorySQLite(t)
	repo := &apiTokenRepository{}
	createdAt := time.Date(2026, 4, 16, 12, 34, 56, 0, time.UTC)

	_, err := db.Exec(
		`INSERT INTO api_tokens (id, user_id, name, hashed_token, created_at) VALUES (?, ?, ?, ?, ?)`,
		1,
		10,
		"テスト用",
		"hashed-token",
		createdAt,
	)
	require.NoError(t, err)

	tokens, err := repo.FindByUserID(context.Background(), db, 10)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	token := tokens[0]
	require.NotNil(t, token)
	assert.Equal(t, int64(1), token.ID)
	assert.Equal(t, 10, token.UserID)
	assert.Equal(t, "テスト用", token.Name)
	assert.Equal(t, "hashed-token", token.HashedToken)
	assert.True(t, token.CreatedAt.Equal(createdAt))
}
