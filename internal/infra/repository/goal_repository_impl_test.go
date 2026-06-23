package repository

import (
	"context"
	"testing"

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
	assert.InDelta(t, 14.5, stats.TotalChartConst, 0.0001)
}
