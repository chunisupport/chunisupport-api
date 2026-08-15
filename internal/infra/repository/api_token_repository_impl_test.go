package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
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
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			hashed_token TEXT NOT NULL UNIQUE,
			token_prefix TEXT NULL,
			last_used_at DATETIME NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (user_id, name)
		)
	`)
	require.NoError(t, err)

	return db
}

func TestAPITokenRepository_SaveListAndUpdate(t *testing.T) {
	db := setupAPITokenRepositorySQLite(t)
	repo := &apiTokenRepository{}
	token, err := entity.NewAPIToken(10, "CLI", strings.Repeat("a", 64), "abcde")
	require.NoError(t, err)

	require.NoError(t, repo.Save(context.Background(), db, token))
	assert.NotZero(t, token.ID)

	tokens, err := repo.ListByUserID(context.Background(), db, 10)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, "CLI", tokens[0].Name.String())
	require.NotNil(t, tokens[0].TokenPrefix)
	assert.Equal(t, "abcde", *tokens[0].TokenPrefix)

	usedAt := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	require.NoError(t, tokens[0].Rename("Batch"))
	tokens[0].RecordUsage(usedAt)
	require.NoError(t, repo.Save(context.Background(), db, tokens[0]))

	updated, err := repo.FindByIDAndUserID(context.Background(), db, token.ID, 10)
	require.NoError(t, err)
	assert.Equal(t, "Batch", updated.Name.String())
	require.NotNil(t, updated.LastUsedAt)
	assert.Equal(t, usedAt, *updated.LastUsedAt)
}

func TestAPITokenRepository_FindByHashedToken_LegacyPrefixCanBeNull(t *testing.T) {
	db := setupAPITokenRepositorySQLite(t)
	repo := &apiTokenRepository{}
	createdAt := time.Date(2026, 4, 16, 12, 34, 56, 0, time.UTC)

	_, err := db.Exec(
		`INSERT INTO api_tokens (id, user_id, name, hashed_token, token_prefix, created_at) VALUES (?, ?, ?, ?, NULL, ?)`,
		1,
		10,
		"既存のトークン",
		strings.Repeat("b", 64),
		createdAt,
	)
	require.NoError(t, err)

	token, err := repo.FindByHashedToken(context.Background(), db, strings.Repeat("b", 64))
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, uint64(1), token.ID)
	assert.Nil(t, token.TokenPrefix)
	assert.True(t, token.CreatedAt.Equal(createdAt))
}

func TestAPITokenRepository_DeleteByIDAndUserID_IsScopedToOwner(t *testing.T) {
	db := setupAPITokenRepositorySQLite(t)
	repo := &apiTokenRepository{}
	token, err := entity.NewAPIToken(10, "CLI", strings.Repeat("c", 64), "abcde")
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), db, token))

	err = repo.DeleteByIDAndUserID(context.Background(), db, token.ID, 11)
	assert.ErrorIs(t, err, domainrepo.ErrAPITokenNotFound)

	require.NoError(t, repo.DeleteByIDAndUserID(context.Background(), db, token.ID, 10))
	_, err = repo.FindByIDAndUserID(context.Background(), db, token.ID, 10)
	assert.ErrorIs(t, err, domainrepo.ErrAPITokenNotFound)
}
