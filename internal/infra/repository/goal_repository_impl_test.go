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
		`CREATE TABLE goal_groups (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			sort_order INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (user_id, name)
		)`,
		`CREATE TABLE goals (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			group_id INTEGER NULL,
			title TEXT NOT NULL,
			achievement_type_id INTEGER NOT NULL,
			achievement_params TEXT NOT NULL,
			attributes TEXT NOT NULL,
			invert_value INTEGER NOT NULL,
			invert_percentage INTEGER NOT NULL,
			sort_order INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
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
			const REAL NOT NULL,
			is_const_unknown INTEGER NOT NULL DEFAULT 0
		)`,
		`INSERT INTO songs (id, genre_id, released_at, is_deleted) VALUES
			(1, 10, '2024-01-01', 0),
			(2, 10, '2024-01-01', 0),
			(3, 10, '2024-01-01', 1)`,
		`INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown) VALUES
			(97, 1, 1, 3.0, 0),
			(98, 1, 2, 6.0, 0),
			(99, 1, 3, 10.0, 0),
			(101, 1, 4, 14.0, 0),
			(102, 1, 5, 15.0, 0),
			(201, 2, 4, 14.5, 0),
			(202, 2, 5, 14.5, 0),
			(301, 3, 5, 16.0, 0)`,
	}
	for _, stmt := range schema {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	return db
}

func TestGoalRepository_ListByUserIDOrdersByGroupAndPutsUnclassifiedLast(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	_, err := db.Exec(`INSERT INTO goal_groups (id, user_id, name, sort_order, created_at) VALUES
		(10, 1, 'second group', 2, '2026-01-01'),
		(20, 1, 'first group', 1, '2026-01-01')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO goals
		(id, user_id, group_id, title, achievement_type_id, achievement_params, attributes, invert_value, invert_percentage, sort_order, created_at)
		VALUES
		(1, 1, NULL, 'unclassified', 1, '{}', '{}', 0, 0, 1, '2026-01-01'),
		(2, 1, 10, 'group second', 1, '{}', '{}', 0, 0, 1, '2026-01-01'),
		(3, 1, 20, 'within second', 1, '{}', '{}', 0, 0, 2, '2026-01-01'),
		(4, 1, 20, 'within first', 1, '{}', '{}', 0, 0, 1, '2026-01-01')`)
	require.NoError(t, err)
	repo := &goalRepository{db: db}

	// When
	goals, err := repo.ListByUserID(context.Background(), db, 1)

	// Then
	require.NoError(t, err)
	assert.Equal(t, []uint32{4, 3, 2, 1}, []uint32{goals[0].ID, goals[1].ID, goals[2].ID, goals[3].ID})
}

func TestGoalRepository_DeleteByUserIDDeletesOnlyOwnedGoals(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	_, err := db.Exec(`INSERT INTO goals
		(id, user_id, title, achievement_type_id, achievement_params, attributes, invert_value, invert_percentage, sort_order)
		VALUES
		(1, 10, '削除対象', 1, '{}', '{}', 0, 0, 1),
		(2, 20, '別ユーザー', 1, '{}', '{}', 0, 0, 1)`)
	require.NoError(t, err)
	repo := &goalRepository{db: db}

	// When
	err = repo.DeleteByUserID(context.Background(), db, 10)

	// Then
	require.NoError(t, err)
	var userIDs []int
	require.NoError(t, db.Select(&userIDs, `SELECT user_id FROM goals ORDER BY id`))
	assert.Equal(t, []int{20}, userIDs)
}

func TestGoalRepository_ListByUserIDReturnsSortOrder(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	_, err := db.Exec(`INSERT INTO goals
		(id, user_id, title, achievement_type_id, achievement_params, attributes, invert_value, invert_percentage, sort_order, created_at)
		VALUES
		(10, 1, 'second', 1, '{}', '{}', 0, 0, 2, '2026-01-01'),
		(20, 1, 'first', 1, '{}', '{}', 0, 0, 1, '2026-01-02')`)
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

func TestGoalRepository_PersistsInvertFlagsIndependently(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	repo := &goalRepository{db: db}
	goal := &entity.Goal{
		UserID:            1,
		Title:             "反転テスト",
		AchievementTypeID: 1,
		AchievementParams: []byte(`{}`),
		Attributes:        []byte(`{}`),
		InvertValue:       true,
		InvertPercentage:  false,
		SortOrder:         1,
	}

	// When
	err := repo.Create(context.Background(), db, goal)

	// Then
	require.NoError(t, err)
	created, err := repo.FindByIDAndUserID(context.Background(), db, goal.ID, goal.UserID)
	require.NoError(t, err)
	assert.True(t, created.InvertValue)
	assert.False(t, created.InvertPercentage)

	// When
	created.InvertValue = false
	created.InvertPercentage = true
	err = repo.Save(context.Background(), db, created)

	// Then
	require.NoError(t, err)
	updated, err := repo.FindByIDAndUserID(context.Background(), db, goal.ID, goal.UserID)
	require.NoError(t, err)
	assert.False(t, updated.InvertValue)
	assert.True(t, updated.InvertPercentage)
}

func TestGoalRepository_SaveGoalOrderAssignsDenseSortOrders(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	_, err := db.Exec(`INSERT INTO goals
		(id, user_id, title, achievement_type_id, achievement_params, attributes, invert_value, invert_percentage, sort_order, created_at)
		VALUES
		(10, 1, 'first', 1, '{}', '{}', 0, 0, 1, '2026-01-01'),
		(20, 1, 'second', 1, '{}', '{}', 0, 0, 2, '2026-01-02'),
		(30, 1, 'third', 1, '{}', '{}', 0, 0, 3, '2026-01-03'),
		(99, 2, 'other user', 1, '{}', '{}', 0, 0, 1, '2026-01-04')`)
	require.NoError(t, err)
	repo := &goalRepository{db: db}

	// When
	arrangement, err := entity.NewGoalArrangement(1, []*entity.Goal{
		{ID: 30, UserID: 1, SortOrder: 1},
		{ID: 10, UserID: 1, SortOrder: 2},
		{ID: 20, UserID: 1, SortOrder: 3},
	})
	require.NoError(t, err)
	err = repo.SaveGoalArrangement(context.Background(), db, arrangement)

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
		(id, user_id, title, achievement_type_id, achievement_params, attributes, invert_value, invert_percentage, sort_order, created_at)
		VALUES (10, 1, 'first', 1, '{}', '{}', 0, 0, 1, '2026-01-01')`)
	require.NoError(t, err)
	repo := &goalRepository{db: db}
	arrangement, err := entity.NewGoalArrangement(1, []*entity.Goal{{ID: 99, UserID: 1}})
	require.NoError(t, err)

	// When
	err = repo.SaveGoalArrangement(context.Background(), db, arrangement)

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

func TestGoalRepository_GetTargetStatsFiltersChartsByTheoreticalSingleRating(t *testing.T) {
	tests := []struct {
		name                    string
		minRatingHundredths     int64
		expectedChartCount      int
		expectedTotalChartConst float64
	}{
		{name: "18.00では15.8を除外して15.9を含む", minRatingHundredths: 1800, expectedChartCount: 1, expectedTotalChartConst: 15.9},
		{name: "17.45では15.3を含む", minRatingHundredths: 1745, expectedChartCount: 3, expectedTotalChartConst: 47.0},
		{name: "17.46では15.3を除外する", minRatingHundredths: 1746, expectedChartCount: 2, expectedTotalChartConst: 31.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupGoalRepositorySQLite(t)
			repo := &goalRepository{db: db}
			_, err := db.Exec(`INSERT INTO songs (id, genre_id, released_at, is_deleted) VALUES
				(10, 99, '2024-01-01', 0),
				(11, 99, '2024-01-01', 0),
				(12, 99, '2024-01-01', 0),
				(13, 99, '2024-01-01', 0)`)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown) VALUES
				(1001, 10, 4, 15.8, 0),
				(1002, 11, 4, 15.9, 0),
				(1003, 12, 4, 15.3, 0),
				(1004, 13, 4, 16.0, 1)`)
			require.NoError(t, err)

			// When
			stats, err := repo.GetTargetStats(context.Background(), db, domainrepo.GoalTargetFilter{
				GenreIDs:                       []int{99},
				MinTheoreticalRatingHundredths: &tt.minRatingHundredths,
			})

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedChartCount, stats.ChartCount)
			assert.InDelta(t, tt.expectedTotalChartConst, stats.TotalChartConst, 0.0001)
		})
	}
}
