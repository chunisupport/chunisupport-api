package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type honorEnsureExec struct {
	baseExecutor
	query string
	args  []any
}

func (e *honorEnsureExec) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	e.query = query
	e.args = args
	return rowsAffectedResult{lastInsertID: 10, rowsAffected: 1}, nil
}

func TestEnsureHonor_称号の一意キー単位でUpsertする(t *testing.T) {
	// Given
	imageURL := " https://example.com/honor.png "
	exec := &honorEnsureExec{}
	repo := &honorRepository{}

	// When
	id, err := repo.EnsureHonor(context.Background(), exec, " 称号A ", 2, &imageURL)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 10, id)
	assert.Contains(t, exec.query, "INSERT INTO honors (name, honor_type_id, image_url)")
	assert.Contains(t, exec.query, "ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)")
	assert.NotContains(t, exec.query, "image_url = VALUES(image_url)")
	assert.Equal(t, []any{"称号A", 2, "https://example.com/honor.png"}, exec.args)
}

func TestEnsureHonor_画像URLがnilの場合は空文字でUpsertする(t *testing.T) {
	// Given
	exec := &honorEnsureExec{}
	repo := &honorRepository{}

	// When
	id, err := repo.EnsureHonor(context.Background(), exec, "称号A", 2, nil)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 10, id)
	assert.Equal(t, []any{"称号A", 2, ""}, exec.args)
}
