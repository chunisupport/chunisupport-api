package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupGoalRepositorySQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	schema := []string{
		`CREATE TABLE goals (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			achievement_type_id INTEGER NOT NULL,
			achievement_params TEXT NOT NULL,
			attributes TEXT NOT NULL,
			invert INTEGER NOT NULL,
			sort_order INTEGER NOT NULL,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE songs (
			id INTEGER PRIMARY KEY,
			genre_id INTEGER NULL,
			released_at TEXT NULL,
			is_deleted INTEGER NOT NULL
		)`,
		`CREATE TABLE charts (
			id INTEGER PRIMARY KEY,
			song_id INTEGER NOT NULL,
			difficulty_id INTEGER NOT NULL,
			const REAL NOT NULL
		)`,
		`INSERT INTO songs (id, genre_id, released_at, is_deleted) VALUES
			(1, 10, '2024-01-01', 0),
			(2, 10, '2024-01-01', 0),
			(3, 10, '2024-01-01', 1)`,
		`INSERT INTO charts (id, song_id, difficulty_id, const) VALUES
			(97, 1, 1, 3.0),
			(98, 1, 2, 6.0),
			(99, 1, 3, 10.0),
			(101, 1, 4, 14.0),
			(102, 1, 5, 15.0),
			(201, 2, 4, 14.5),
			(202, 2, 5, 14.5),
			(301, 3, 5, 16.0)`,
	}
	for _, stmt := range schema {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	return db
}

func TestGoalRepository_ListByUserIDReturnsSortOrder(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	_, err := db.Exec(`INSERT INTO goals
		(id, user_id, title, achievement_type_id, achievement_params, attributes, invert, sort_order, created_at)
		VALUES
		(10, 1, 'second', 1, '{}', '{}', 0, 2, '2026-01-01'),
		(20, 1, 'first', 1, '{}', '{}', 0, 1, '2026-01-02')`)
	require.NoError(t, err)
	repo := &goalRepository{db: db}

	// When
	goals, err := repo.ListByUserID(context.Background(), db, 1)

	// Then
	require.NoError(t, err)
	require.Len(t, goals, 2)
	assert.Equal(t, uint32(20), goals[0].ID)
	assert.Equal(t, uint16(1), goals[0].SortOrder)
	assert.Equal(t, uint32(10), goals[1].ID)
	assert.Equal(t, uint16(2), goals[1].SortOrder)
}

func TestGoalRepository_SaveGoalOrderAssignsDenseSortOrders(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	_, err := db.Exec(`INSERT INTO goals
		(id, user_id, title, achievement_type_id, achievement_params, attributes, invert, sort_order, created_at)
		VALUES
		(10, 1, 'first', 1, '{}', '{}', 0, 1, '2026-01-01'),
		(20, 1, 'second', 1, '{}', '{}', 0, 2, '2026-01-02'),
		(30, 1, 'third', 1, '{}', '{}', 0, 3, '2026-01-03'),
		(99, 2, 'other user', 1, '{}', '{}', 0, 1, '2026-01-04')`)
	require.NoError(t, err)
	repo := &goalRepository{db: db}

	// When
	order, err := entity.NewGoalOrder(1, []*entity.Goal{
		{ID: 30, UserID: 1, SortOrder: 1},
		{ID: 10, UserID: 1, SortOrder: 2},
		{ID: 20, UserID: 1, SortOrder: 3},
	})
	require.NoError(t, err)
	err = repo.SaveGoalOrder(context.Background(), db, order)

	// Then
	require.NoError(t, err)
	goals, err := repo.ListByUserID(context.Background(), db, 1)
	require.NoError(t, err)
	assert.Equal(t, []uint32{30, 10, 20}, []uint32{goals[0].ID, goals[1].ID, goals[2].ID})
	otherGoals, err := repo.ListByUserID(context.Background(), db, 2)
	require.NoError(t, err)
	require.Len(t, otherGoals, 1)
	assert.Equal(t, uint16(1), otherGoals[0].SortOrder)
}

func TestGoalRepository_SaveGoalOrderRejectsMismatchedGoalSet(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	_, err := db.Exec(`INSERT INTO goals
		(id, user_id, title, achievement_type_id, achievement_params, attributes, invert, sort_order, created_at)
		VALUES (10, 1, 'first', 1, '{}', '{}', 0, 1, '2026-01-01')`)
	require.NoError(t, err)
	repo := &goalRepository{db: db}
	order, err := entity.NewGoalOrder(1, []*entity.Goal{{ID: 99, UserID: 1}})
	require.NoError(t, err)

	// When
	err = repo.SaveGoalOrder(context.Background(), db, order)

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrGoalOrderInconsistent)
}

func TestGoalRepository_GetTargetStatsOPTargetOnly(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	repo := &goalRepository{db: db}

	// When
	stats, err := repo.GetTargetStats(context.Background(), db, domainrepo.GoalTargetFilter{
		OPTargetOnly: true,
	})

	// Then
	require.NoError(t, err)
	assert.Equal(t, 2, stats.ChartCount)
	assert.Equal(t, 1, stats.SongCount)
	assert.InDelta(t, 29.5, stats.TotalChartConst, 0.0001)
}

func TestGoalRepository_GetTargetStatsOPTargetOnlyWithConstFilter(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	repo := &goalRepository{db: db}
	maxConst := 14.9

	// When
	stats, err := repo.GetTargetStats(context.Background(), db, domainrepo.GoalTargetFilter{
		ConstMax:     &maxConst,
		OPTargetOnly: true,
	})

	// Then
	require.NoError(t, err)
	assert.Equal(t, 1, stats.ChartCount)
	assert.Equal(t, 0, stats.SongCount)
	assert.InDelta(t, 14.5, stats.TotalChartConst, 0.0001)
}
