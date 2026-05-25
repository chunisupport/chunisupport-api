package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/go-sql-driver/mysql"
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

type honorExecResultExecutor struct {
	baseExecutor
	result sql.Result
	err    error
}

func (e *honorExecResultExecutor) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	if e.err != nil {
		return nil, e.err
	}
	return e.result, nil
}

func TestHonorSave_更新対象がない場合はErrHonorNotFoundを返す(t *testing.T) {
	// Given
	exec := &honorExecResultExecutor{result: rowsAffectedResult{rowsAffected: 0}}
	repo := &honorRepository{}

	// When
	err := repo.Save(context.Background(), exec, &entity.Honor{ID: 1, Name: "称号A", HonorTypeID: 1, ImageURL: ""})

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrHonorNotFound)
}

func TestHonorSave_一意制約違反はErrHonorConflictへ変換する(t *testing.T) {
	// Given
	exec := &honorExecResultExecutor{err: &mysql.MySQLError{
		Number:  mysqlDuplicateEntryErrorNumber,
		Message: "Duplicate entry '称号A-1-' for key 'unique_honor_name_type_image_url'",
	}}
	repo := &honorRepository{}

	// When
	err := repo.Save(context.Background(), exec, &entity.Honor{ID: 1, Name: "称号A", HonorTypeID: 1, ImageURL: ""})

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrHonorConflict)
}

func TestHonorDelete_更新対象がない場合はErrHonorNotFoundを返す(t *testing.T) {
	// Given
	exec := &honorExecResultExecutor{result: rowsAffectedResult{rowsAffected: 0}}
	repo := &honorRepository{}

	// When
	err := repo.Delete(context.Background(), exec, 1)

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrHonorNotFound)
}

func TestHonorDelete_参照制約違反はErrHonorConflictへ変換する(t *testing.T) {
	// Given
	exec := &honorExecResultExecutor{err: &mysql.MySQLError{
		Number:  mysqlCannotDeleteOrUpdateParentRowErrorNumber,
		Message: "Cannot delete or update a parent row: a foreign key constraint fails",
	}}
	repo := &honorRepository{}

	// When
	err := repo.Delete(context.Background(), exec, 1)

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrHonorConflict)
}

func TestWrapHonorDuplicateError_対象外のエラーは変換しない(t *testing.T) {
	// Given
	err := errors.New("other error")

	// When
	got := wrapHonorDuplicateError(err)

	// Then
	assert.ErrorIs(t, got, err)
	assert.NotErrorIs(t, got, domainrepo.ErrHonorConflict)
}
