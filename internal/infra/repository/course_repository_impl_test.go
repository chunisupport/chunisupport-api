package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCourseRepositoryDB(t *testing.T) *courseRepository {
	t.Helper()
	db := setupTestDB(t)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err := db.Exec(`
		CREATE TABLE course_classes (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL);
		CREATE TABLE courses (id INTEGER PRIMARY KEY, official_idx TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
			course_class_id INTEGER NOT NULL, is_deleted INTEGER NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL);
		CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE player_course_records (player_id INTEGER NOT NULL, course_id INTEGER NOT NULL, score INTEGER NOT NULL,
			is_clear INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL, updated_at DATETIME NOT NULL, PRIMARY KEY(player_id, course_id));
		INSERT INTO course_classes VALUES (1, '1', 0), (7, 'extra', 6);
		INSERT INTO combo_lamp_types VALUES (1, 'NONE'), (3, 'ALL JUSTICE');
		INSERT INTO courses VALUES
			(10, '50020', '通常コース', 1, 0, '2026-07-01 00:00:00', '2026-07-01 00:00:00'),
			(11, '50029', '削除コース', 7, 1, '2026-07-01 00:00:00', '2026-07-01 00:00:00');
		INSERT INTO player_course_records VALUES (100, 10, 3023238, 1, 1, '2026-07-12 10:00:00');
	`)
	require.NoError(t, err)
	return &courseRepository{db: db}
}

func TestCourseRepository_FindAll_削除済みを除外する(t *testing.T) {
	repo := setupCourseRepositoryDB(t)
	courses, err := repo.FindAll(context.Background(), repo.db, false)
	require.NoError(t, err)
	require.Len(t, courses, 1)
	assert.Equal(t, "通常コース", courses[0].Name)
	assert.Equal(t, "1", courses[0].CourseClass.Name)
}

func TestCourseRepository_FindRecordsByPlayerID_未プレイを補完する(t *testing.T) {
	repo := setupCourseRepositoryDB(t)
	records, err := repo.FindRecordsByPlayerID(context.Background(), repo.db, 100, false, true)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, uint32(3023238), records[0].Score.Uint32())
	assert.True(t, records[0].IsClear)
}
