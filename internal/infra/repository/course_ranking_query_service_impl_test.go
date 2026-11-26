package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCourseRankingQueryService_相互承認とコース別に自己ベストを絞り込む(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupFriendChartRankingDB(t, db)
	insertFriendChartRankingFixtures(t, db)
	_, err := db.Exec(`
		CREATE TABLE player_course_records (
			player_id INTEGER NOT NULL,
			course_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			is_clear BOOLEAN NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (player_id, course_id)
		);
		INSERT INTO player_course_records VALUES
			(101, 10, 2800000, 0, 1, '2026-07-09 12:00:00'),
			(102, 10, 3020000, 1, 3, '2026-07-09 12:01:00'),
			(103, 10, 3020000, 1, 2, '2026-07-09 12:00:00'),
			(104, 10, 3030000, 1, 3, '2026-07-09 12:00:00'),
			(105, 20, 3030000, 1, 3, '2026-07-09 12:00:00');
		INSERT INTO friendships VALUES (1, 5, 2), (5, 1, 2);
	`)
	require.NoError(t, err)
	query := CourseRankingQueryService{}

	// When
	friends, err := query.ListFriends(t.Context(), db, 10, 1)
	require.NoError(t, err)
	top, total, err := query.ListTop(t.Context(), db, 10, 2)
	require.NoError(t, err)

	// Then
	require.Len(t, friends, 3)
	assert.Equal(t, []int{2, 3, 1}, []int{friends[0].UserID, friends[1].UserID, friends[2].UserID})
	assert.Equal(t, "ALL JUSTICE", friends[0].ComboLamp)
	assert.True(t, friends[0].IsClear)
	assert.False(t, friends[2].IsClear)
	require.Len(t, top, 2)
	assert.Equal(t, 4, total)
	assert.Equal(t, []int{4, 2}, []int{top[0].UserID, top[1].UserID})

	empty, emptyTotal, err := query.ListTop(t.Context(), db, 30, 2)
	require.NoError(t, err)
	assert.Empty(t, empty)
	assert.Zero(t, emptyTotal)
}
