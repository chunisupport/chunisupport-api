package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type baseExecutor struct{}

func (e *baseExecutor) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	panic("unexpected call to GetContext")
}

func (e *baseExecutor) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	panic("unexpected call to SelectContext")
}

func (e *baseExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	panic("unexpected call to ExecContext")
}

func (e *baseExecutor) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	panic("unexpected call to NamedExecContext")
}

func (e *baseExecutor) Rebind(query string) string {
	panic("unexpected call to Rebind")
}

func (e *baseExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	panic("unexpected call to QueryContext")
}

func (e *baseExecutor) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	panic("unexpected call to QueryxContext")
}

func (e *baseExecutor) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	panic("unexpected call to QueryRowxContext")
}

var _ domainrepo.Executor = (*baseExecutor)(nil)

type noRowsWrappedExecutor struct {
	baseExecutor
}

func (e *noRowsWrappedExecutor) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	return errors.Join(errors.New("wrapped no rows"), sql.ErrNoRows)
}

var _ domainrepo.Executor = (*noRowsWrappedExecutor)(nil)

type rowsAffectedResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r rowsAffectedResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

func (r rowsAffectedResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

type execResultExecutor struct {
	baseExecutor
	result sql.Result
	err    error
}

func (e *execResultExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if e.err != nil {
		return nil, e.err
	}
	return e.result, nil
}

var _ domainrepo.Executor = (*execResultExecutor)(nil)

func TestFindByUserID_ReturnsNilWhenWrappedNoRows(t *testing.T) {
	repo := &playerRepository{}
	exec := &noRowsWrappedExecutor{}

	player, err := repo.FindByUserID(context.Background(), exec, 10)
	require.NoError(t, err)
	require.Nil(t, player)
}

func TestFindByID_ReturnsErrPlayerNotFoundWhenWrappedNoRows(t *testing.T) {
	repo := &playerRepository{}
	exec := &noRowsWrappedExecutor{}

	player, err := repo.FindByID(context.Background(), exec, 10)
	require.ErrorIs(t, err, domainrepo.ErrPlayerNotFound)
	require.Nil(t, player)
}

func TestAPITokenFindByHashedToken_ReturnsErrAPITokenNotFoundWhenWrappedNoRows(t *testing.T) {
	repo := &apiTokenRepository{}
	exec := &noRowsWrappedExecutor{}

	token, err := repo.FindByHashedToken(context.Background(), exec, "hashed-token")
	require.ErrorIs(t, err, domainrepo.ErrAPITokenNotFound)
	require.Nil(t, token)
}

func TestAPITokenFindByUserID_ReturnsErrAPITokenNotFoundWhenWrappedNoRows(t *testing.T) {
	repo := &apiTokenRepository{}
	exec := &noRowsWrappedExecutor{}

	token, err := repo.FindByUserID(context.Background(), exec, 10)
	require.ErrorIs(t, err, domainrepo.ErrAPITokenNotFound)
	require.Nil(t, token)
}

func TestGoalFindByIDAndUserID_ReturnsErrGoalNotFoundWhenWrappedNoRows(t *testing.T) {
	repo := &goalRepository{}
	exec := &noRowsWrappedExecutor{}

	goal, err := repo.FindByIDAndUserID(context.Background(), exec, 1, 1)
	require.ErrorIs(t, err, domainrepo.ErrGoalNotFound)
	require.Nil(t, goal)
}

func TestGoalUpdate_ReturnsErrGoalNotFoundWhenNoRowsAffected(t *testing.T) {
	repo := &goalRepository{}
	exec := &execResultExecutor{result: rowsAffectedResult{rowsAffected: 0}}

	err := repo.Update(context.Background(), exec, &entity.Goal{ID: 1, UserID: 1})
	require.ErrorIs(t, err, domainrepo.ErrGoalNotFound)
}

func TestGoalDelete_ReturnsErrGoalNotFoundWhenNoRowsAffected(t *testing.T) {
	repo := &goalRepository{}
	exec := &execResultExecutor{result: rowsAffectedResult{rowsAffected: 0}}

	err := repo.DeleteByIDAndUserID(context.Background(), exec, 1, 1)
	require.ErrorIs(t, err, domainrepo.ErrGoalNotFound)
}

func setupPlayerRepositorySQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	schema := []string{
		`CREATE TABLE players (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			player_name TEXT NOT NULL,
			player_level INTEGER NOT NULL,
			official_player_rating REAL NULL,
			calculated_player_rating REAL NULL,
			new_average_rating REAL NULL,
			best_average_rating REAL NULL,
			class_emblem_id INTEGER NULL,
			class_emblem_base_id INTEGER NULL,
			last_played_at DATETIME NULL,
			overpower_value REAL NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE honor_types (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE honors (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			honor_type_id INTEGER NOT NULL,
			image_url TEXT NULL
		)`,
		`CREATE TABLE player_honors (
			player_id INTEGER NOT NULL,
			honor_id INTEGER NOT NULL,
			slot INTEGER NOT NULL
		)`,
	}

	for _, stmt := range schema {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	return db
}

func seedPlayerWithHonors(t *testing.T, db *sqlx.DB, playerID int, withHonors bool) time.Time {
	t.Helper()

	now := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO players (
			id, user_id, player_name, player_level,
			official_player_rating, calculated_player_rating, new_average_rating, best_average_rating,
			class_emblem_id, class_emblem_base_id, last_played_at,
			overpower_value, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, playerID, 20, "テストプレイヤー", 30, 16.25, nil, nil, nil, nil, nil, nil, nil, now, now)
	require.NoError(t, err)

	if !withHonors {
		return now
	}

	_, err = db.Exec(`INSERT INTO honor_types (id, name) VALUES (1, 'normal'), (2, 'gold')`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO honors (id, name, honor_type_id, image_url) VALUES
		(10, '称号B', 2, 'https://example.com/b.png'),
		(11, '称号A', 1, 'https://example.com/a.png'),
		(12, '称号C', 2, NULL)
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO player_honors (player_id, honor_id, slot) VALUES
		(?, 10, 2),
		(?, 11, 1),
		(?, 12, 3)
	`, playerID, playerID, playerID)
	require.NoError(t, err)

	return now
}

func TestFindByIDWithHonors_ReturnsPlayerWithSortedHonors(t *testing.T) {
	db := setupPlayerRepositorySQLite(t)
	repo := &playerRepository{}
	now := seedPlayerWithHonors(t, db, 1, true)

	result, err := repo.FindByIDWithHonors(context.Background(), db, 1)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Player)
	require.Equal(t, 1, result.Player.ID)
	require.Equal(t, "テストプレイヤー", result.Player.Name.String())
	require.True(t, result.Player.UpdatedAt.Equal(now))
	require.Len(t, result.Honors, 3)
	require.Equal(t, 1, result.Honors[0].Slot)
	require.Equal(t, "称号A", result.Honors[0].Name)
	require.Equal(t, 2, result.Honors[1].Slot)
	require.Equal(t, "称号B", result.Honors[1].Name)
	require.Equal(t, 3, result.Honors[2].Slot)
	require.Equal(t, "称号C", result.Honors[2].Name)
}

func TestFindByIDWithHonors_ReturnsEmptyHonorsWhenNotAssigned(t *testing.T) {
	db := setupPlayerRepositorySQLite(t)
	repo := &playerRepository{}
	seedPlayerWithHonors(t, db, 2, false)

	result, err := repo.FindByIDWithHonors(context.Background(), db, 2)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Player)
	require.Empty(t, result.Honors)
}

func TestFindByIDWithHonors_ReturnsNoRowsWhenPlayerMissing(t *testing.T) {
	db := setupPlayerRepositorySQLite(t)
	repo := &playerRepository{}

	result, err := repo.FindByIDWithHonors(context.Background(), db, 999)
	require.ErrorIs(t, err, domainrepo.ErrPlayerNotFound)
	require.Nil(t, result)
}
