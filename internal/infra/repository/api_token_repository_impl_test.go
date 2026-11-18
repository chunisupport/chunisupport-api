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
			display_id TEXT NOT NULL UNIQUE,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			permission TEXT NOT NULL DEFAULT 'read_write',
			public_only BOOLEAN NOT NULL DEFAULT FALSE,
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
	token, err := entity.NewAPIToken(10, "0123456789abcdef", "CLI", strings.Repeat("a", 64), "abcde", "read_write", false)
	require.NoError(t, err)

	require.NoError(t, repo.Save(context.Background(), db, token))
	assert.NotZero(t, token.ID)

	tokens, err := repo.ListByUserID(context.Background(), db, 10)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, "CLI", tokens[0].Name.String())
	assert.Equal(t, "read_write", tokens[0].Permission.String())
	require.NotNil(t, tokens[0].TokenPrefix)
	assert.Equal(t, "abcde", *tokens[0].TokenPrefix)

	usedAt := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	require.NoError(t, tokens[0].Rename("Batch"))
	tokens[0].RecordUsage(usedAt)
	require.NoError(t, repo.Save(context.Background(), db, tokens[0]))

	updated, err := repo.FindByDisplayIDAndUserID(context.Background(), db, token.DisplayID, 10)
	require.NoError(t, err)
	assert.Equal(t, "Batch", updated.Name.String())
	require.NotNil(t, updated.LastUsedAt)
	assert.Equal(t, usedAt, *updated.LastUsedAt)
}

func TestAPITokenRepository_SaveAndLoad_ReadPermission(t *testing.T) {
	db := setupAPITokenRepositorySQLite(t)
	repo := &apiTokenRepository{}
	token, err := entity.NewAPIToken(10, "0123456789abcdef", "Read only", strings.Repeat("d", 64), "abcde", "read", false)
	require.NoError(t, err)

	require.NoError(t, repo.Save(context.Background(), db, token))
	loaded, err := repo.FindByDisplayIDAndUserID(context.Background(), db, token.DisplayID, 10)

	require.NoError(t, err)
	assert.Equal(t, "read", loaded.Permission.String())
}

func TestAPITokenRepository_SaveAndLoad_PublicOnly(t *testing.T) {
	// Given
	db := setupAPITokenRepositorySQLite(t)
	repo := &apiTokenRepository{}
	token, err := entity.NewAPIToken(10, "0123456789abcdef", "Public", strings.Repeat("e", 64), "abcde", "read", true)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), db, token))

	// When
	loaded, err := repo.FindByHashedToken(context.Background(), db, strings.Repeat("e", 64))

	// Then
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.True(t, loaded.PublicOnly)
}

func TestAPITokenRepository_FindByHashedToken_LegacyPrefixCanBeNull(t *testing.T) {
	db := setupAPITokenRepositorySQLite(t)
	repo := &apiTokenRepository{}
	createdAt := time.Date(2026, 4, 16, 12, 34, 56, 0, time.UTC)

	_, err := db.Exec(
		`INSERT INTO api_tokens (id, display_id, user_id, name, hashed_token, token_prefix, created_at) VALUES (?, ?, ?, ?, ?, NULL, ?)`,
		1,
		"fedcba9876543210",
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
	assert.Equal(t, "fedcba9876543210", token.DisplayID.String())
	assert.Nil(t, token.TokenPrefix)
	assert.True(t, token.CreatedAt.Equal(createdAt))
}

func TestAPITokenRepository_DeleteByDisplayIDAndUserID_IsScopedToOwner(t *testing.T) {
	db := setupAPITokenRepositorySQLite(t)
	repo := &apiTokenRepository{}
	token, err := entity.NewAPIToken(10, "0123456789abcdef", "CLI", strings.Repeat("c", 64), "abcde", "read_write", false)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), db, token))

	err = repo.DeleteByDisplayIDAndUserID(context.Background(), db, token.DisplayID, 11)
	assert.ErrorIs(t, err, domainrepo.ErrAPITokenNotFound)

	require.NoError(t, repo.DeleteByDisplayIDAndUserID(context.Background(), db, token.DisplayID, 10))
	_, err = repo.FindByDisplayIDAndUserID(context.Background(), db, token.DisplayID, 10)
	assert.ErrorIs(t, err, domainrepo.ErrAPITokenNotFound)
}
