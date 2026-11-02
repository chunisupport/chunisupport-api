package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type honorEnsureExec struct {
	baseExecutor
	query           string
	args            []any
	execErr         error
	existingID      int
	afterMissID     int
	nameMatchID     int
	nameForUpdateID int
	getCalled       bool
	getCount        int
	getQueries      []string
	getArgs         [][]any
}

func (e *honorEnsureExec) GetContext(_ context.Context, dest any, query string, args ...any) error {
	e.getCalled = true
	e.getCount++
	e.getQueries = append(e.getQueries, query)
	e.getArgs = append(e.getArgs, args)
	existingID := e.existingID
	switch {
	case strings.Contains(query, "FOR UPDATE") && strings.Contains(query, "name = ?"):
		existingID = e.nameForUpdateID
	case strings.Contains(query, "FOR UPDATE"):
		existingID = e.afterMissID
	case strings.Contains(query, "image_url"):
		existingID = e.existingID
	case e.getCount > 1:
		existingID = e.nameMatchID
	}
	if existingID == 0 {
		return sql.ErrNoRows
	}
	*(dest.(*int)) = existingID
	return nil
}

func (e *honorEnsureExec) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	e.query = query
	e.args = args
	if e.execErr != nil {
		return nil, e.execErr
	}
	return rowsAffectedResult{lastInsertID: 10, rowsAffected: 1}, nil
}

func TestEnsureHonor_SP称号は画像URLの重複を事前確認して新規登録する(t *testing.T) {
	// Given
	imageURL := " https://example.com/honor.png "
	exec := &honorEnsureExec{}
	repo := &honorRepository{}

	// When
	result, err := repo.EnsureHonor(context.Background(), exec, " 称号A ", 2, &imageURL)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 10, result.ID)
	assert.True(t, result.ImageURLRegistered)
	assert.True(t, exec.getCalled)
	assert.Contains(t, exec.query, "INSERT INTO honors (name, honor_type_id, image_url)")
	assert.NotContains(t, exec.query, "ON DUPLICATE KEY UPDATE")
	assert.NotContains(t, exec.query, "image_url = VALUES(image_url)")
	assert.Equal(t, []any{"称号A", 2, "https://example.com/honor.png"}, exec.args)
}

func TestEnsureHonor_画像URLがnilの場合はNULLでUpsertする(t *testing.T) {
	// Given
	exec := &honorEnsureExec{}
	repo := &honorRepository{}

	// When
	result, err := repo.EnsureHonor(context.Background(), exec, "称号A", 2, nil)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 10, result.ID)
	assert.False(t, result.ImageURLRegistered)
	assert.Contains(t, exec.query, "ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)")
	assert.Equal(t, []any{"称号A", 2, nil}, exec.args)
}

func TestEnsureHonor_既存の通常称号はINSERTせず既存IDを返す(t *testing.T) {
	// Given: INSERT ... ON DUPLICATE KEY UPDATE は重複時もAUTO_INCREMENTを消費するため、既存称号では実行しない
	exec := &honorEnsureExec{existingID: 20}
	repo := &honorRepository{}

	// When
	result, err := repo.EnsureHonor(context.Background(), exec, " 称号A ", 2, nil)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 20, result.ID)
	assert.False(t, result.ImageURLRegistered)
	require.Len(t, exec.getQueries, 1)
	assert.Contains(t, exec.getQueries[0], "WHERE name = ? AND honor_type_id = ?")
	assert.Equal(t, []any{"称号A", 2}, exec.getArgs[0])
	assert.Empty(t, exec.query)
}

func TestEnsureHonor_画像URLが空文字の場合はNULLでUpsertする(t *testing.T) {
	// Given
	imageURL := "  "
	exec := &honorEnsureExec{}
	repo := &honorRepository{}

	// When
	result, err := repo.EnsureHonor(context.Background(), exec, "称号A", 2, &imageURL)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 10, result.ID)
	assert.Equal(t, []any{"称号A", 2, nil}, exec.args)
}

func TestEnsureHonor_同じ画像URLの称号がある場合は手動変更済みの既存IDを返す(t *testing.T) {
	// Given
	imageURL := "https://example.com/honor.png"
	exec := &honorEnsureExec{existingID: 20}
	repo := &honorRepository{}

	// When
	result, err := repo.EnsureHonor(context.Background(), exec, "honor.png", 2, &imageURL)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 20, result.ID)
	assert.False(t, result.ImageURLRegistered)
	assert.True(t, exec.getCalled)
	assert.Empty(t, exec.query)
	assert.Empty(t, exec.args)
}

func TestEnsureHonor_SP称号の名称だけが重複した場合はErrHonorConflictを返す(t *testing.T) {
	// Given
	imageURL := "https://example.com/v2/honor.png"
	exec := &honorEnsureExec{execErr: &mysql.MySQLError{
		Number:  mysqlDuplicateEntryErrorNumber,
		Message: "Duplicate entry 'honor.png-2' for key 'unique_honor_name_type'",
	}}
	repo := &honorRepository{}

	// When
	_, err := repo.EnsureHonor(context.Background(), exec, "honor.png", 2, &imageURL)

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrHonorConflict)
}

func TestEnsureHonor_SP称号の並行登録で画像URLが重複した場合は既存IDを返す(t *testing.T) {
	tests := []struct {
		name    string
		keyName string
	}{
		{name: "画像URL側の制約違反", keyName: "unique_honor_image_url"},
		{name: "称号名側の制約違反", keyName: "unique_honor_name_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			imageURL := "https://example.com/honor.png"
			exec := &honorEnsureExec{
				afterMissID: 30,
				execErr: &mysql.MySQLError{
					Number:  mysqlDuplicateEntryErrorNumber,
					Message: "Duplicate entry for key '" + tt.keyName + "'",
				},
			}
			repo := &honorRepository{}

			// When
			result, err := repo.EnsureHonor(context.Background(), exec, "honor.png", 2, &imageURL)

			// Then
			require.NoError(t, err)
			assert.Equal(t, 30, result.ID)
			assert.False(t, result.ImageURLRegistered)
			require.Len(t, exec.getQueries, 3)
			assert.Contains(t, exec.getQueries[1], "name = ? AND honor_type_id = ?")
			assert.NotContains(t, exec.getQueries[1], "FOR UPDATE")
			assert.Contains(t, exec.getQueries[2], "image_url = ? FOR UPDATE")
		})
	}
}

func TestEnsureHonor_同じファイル名でURL文字列だけ異なるSP称号は既存行を使う(t *testing.T) {
	tests := []struct {
		name     string
		imageURL string
	}{
		{
			name:     "クエリだけ異なる",
			imageURL: "https://new.chunithm-net.com/img/honor_xxx.png?ver=2",
		},
		{
			name:     "フラグメントだけ異なる",
			imageURL: "https://new.chunithm-net.com/img/honor_xxx.png#v2",
		},
		{
			name:     "ホストだけ異なる",
			imageURL: "https://old.chunithm-net.com/img/honor_xxx.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 画像URLの完全一致では見つからず、一意制約と同じ name・honor_type_id では見つかる
			exec := &honorEnsureExec{nameMatchID: 42}
			repo := &honorRepository{}

			// When
			result, err := repo.EnsureHonor(context.Background(), exec, "honor_xxx.png", 2, &tt.imageURL)

			// Then: 保存済みURLは書き換えず、既存IDを返す
			require.NoError(t, err)
			assert.Equal(t, 42, result.ID)
			assert.False(t, result.ImageURLRegistered)
			assert.Empty(t, exec.query)
			require.Len(t, exec.getQueries, 2)
			assert.Contains(t, exec.getQueries[0], "WHERE image_url = ?")
			assert.Equal(t, []any{tt.imageURL}, exec.getArgs[0])
			assert.Contains(t, exec.getQueries[1], "WHERE name = ? AND honor_type_id = ?")
			assert.NotContains(t, exec.getQueries[1], "FOR UPDATE")
			assert.Equal(t, []any{"honor_xxx.png", 2}, exec.getArgs[1])
		})
	}
}

func TestEnsureHonor_URL違いのSP称号が名称制約に衝突した場合は既存IDを返す(t *testing.T) {
	// Given: 事前検索では見えず、INSERT が unique_honor_name_type に衝突する。
	// image_url のカレントリードでは見つからず、name・honor_type_id のカレントリードで見つかる。
	imageURL := "https://other.example.com/img/honor_xxx.png?ver=2"
	exec := &honorEnsureExec{
		nameForUpdateID: 77,
		execErr: &mysql.MySQLError{
			Number:  mysqlDuplicateEntryErrorNumber,
			Message: "Duplicate entry 'honor_xxx.png-2' for key 'unique_honor_name_type'",
		},
	}
	repo := &honorRepository{}

	// When
	result, err := repo.EnsureHonor(context.Background(), exec, "honor_xxx.png", 2, &imageURL)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 77, result.ID)
	assert.False(t, result.ImageURLRegistered)
	require.Len(t, exec.getQueries, 4)
	assert.Contains(t, exec.getQueries[2], "image_url = ? FOR UPDATE")
	assert.Contains(t, exec.getQueries[3], "name = ? AND honor_type_id = ?")
	assert.Contains(t, exec.getQueries[3], "FOR UPDATE")
	assert.Equal(t, []any{"honor_xxx.png", 2}, exec.getArgs[3])
}

func TestDeletePlayerHonorsExceptSlots_指定スロットを削除対象から除外する(t *testing.T) {
	// Given
	exec := &honorEnsureExec{}
	repo := &honorRepository{}

	// When
	err := repo.DeletePlayerHonorsExceptSlots(context.Background(), exec, 100, []int{1, 3})

	// Then
	require.NoError(t, err)
	assert.Equal(t, "DELETE FROM player_honors WHERE player_id = ? AND slot NOT IN (?, ?)", exec.query)
	assert.Equal(t, []any{100, 1, 3}, exec.args)
}

type honorExecResultExecutor struct {
	baseExecutor
	result sql.Result
	err    error
	args   []any
}

func (e *honorExecResultExecutor) ExecContext(_ context.Context, _ string, args ...any) (sql.Result, error) {
	e.args = args
	if e.err != nil {
		return nil, e.err
	}
	return e.result, nil
}

type honorCreateExecutor struct {
	baseExecutor
	args []any
}

func (e *honorCreateExecutor) ExecContext(_ context.Context, _ string, args ...any) (sql.Result, error) {
	e.args = args
	return rowsAffectedResult{lastInsertID: 10, rowsAffected: 1}, nil
}

func (e *honorCreateExecutor) GetContext(_ context.Context, dest any, _ string, _ ...any) error {
	row := dest.(*honorRow)
	row.ID = 10
	row.Name = "称号A"
	row.HonorTypeID = 1
	return nil
}

func TestHonorCreate_空の画像URLはNULLで保存する(t *testing.T) {
	// Given
	exec := &honorCreateExecutor{}
	repo := &honorRepository{}

	// When
	created, err := repo.Create(context.Background(), exec, &entity.Honor{Name: "称号A", HonorTypeID: 1, ImageURL: "  "})

	// Then
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, []any{"称号A", 1, nil}, exec.args)
}

func TestHonorSave_空の画像URLはNULLで保存する(t *testing.T) {
	// Given
	exec := &honorExecResultExecutor{result: rowsAffectedResult{rowsAffected: 1}}
	repo := &honorRepository{}

	// When
	err := repo.Save(context.Background(), exec, &entity.Honor{ID: 1, Name: "称号A", HonorTypeID: 1, ImageURL: "  "})

	// Then
	require.NoError(t, err)
	assert.Equal(t, []any{"称号A", 1, nil, 1}, exec.args)
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

func TestHonorSave_現行の一意制約違反はErrHonorConflictへ変換する(t *testing.T) {
	tests := []struct {
		name    string
		keyName string
	}{
		{name: "称号名と種類の重複", keyName: "unique_honor_name_type"},
		{name: "画像URLの重複", keyName: "unique_honor_image_url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			exec := &honorExecResultExecutor{err: &mysql.MySQLError{
				Number:  mysqlDuplicateEntryErrorNumber,
				Message: "Duplicate entry for key '" + tt.keyName + "'",
			}}
			repo := &honorRepository{}

			// When
			err := repo.Save(context.Background(), exec, &entity.Honor{ID: 1, Name: "称号A", HonorTypeID: 1})

			// Then
			assert.ErrorIs(t, err, domainrepo.ErrHonorConflict)
		})
	}
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
