package repository

import (
	"context"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCourseRepositoryDB(t *testing.T) *courseRepository {
	t.Helper()
	db := setupTestDB(t)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err := db.Exec(`
		CREATE TABLE course_classes (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL);
		CREATE TABLE courses (id INTEGER PRIMARY KEY, display_id TEXT NOT NULL UNIQUE, official_idx TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
			course_class_id INTEGER NOT NULL, is_deleted INTEGER NOT NULL, updated_at DATETIME NOT NULL);
		CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE player_course_records (player_id INTEGER NOT NULL, course_id INTEGER NOT NULL, score INTEGER NOT NULL,
			is_clear INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL, updated_at DATETIME NOT NULL, PRIMARY KEY(player_id, course_id));
		INSERT INTO course_classes VALUES (1, '1', 0), (7, 'extra', 6);
		INSERT INTO combo_lamp_types VALUES (1, 'NONE'), (3, 'ALL JUSTICE');
		INSERT INTO courses VALUES
			(10, '0000000000000010', '50020', '通常コース', 1, 0, '2026-07-01 00:00:00'),
			(11, '0000000000000011', '50029', '削除コース', 7, 1, '2026-07-01 00:00:00');
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
	assert.Equal(t, "0000000000000010", courses[0].DisplayID.String())
	assert.Equal(t, "1", courses[0].CourseClass.Name)
}

func TestCourseRepository_FindByDisplayID_表示用IDで取得する(t *testing.T) {
	repo := setupCourseRepositoryDB(t)

	course, err := repo.FindByDisplayID(context.Background(), repo.db, "0000000000000010", false)

	require.NoError(t, err)
	assert.Equal(t, "50020", course.OfficialIdx)
}

func TestCourseRepository_FindByDisplayID_削除済みの公開取得を拒否する(t *testing.T) {
	repo := setupCourseRepositoryDB(t)

	_, err := repo.FindByDisplayID(context.Background(), repo.db, "0000000000000011", false)

	assert.ErrorIs(t, err, domainrepo.ErrCourseNotFound)
}

func TestCourseRepository_FindByDisplayID_存在しないIDを拒否する(t *testing.T) {
	repo := setupCourseRepositoryDB(t)

	_, err := repo.FindByDisplayID(context.Background(), repo.db, "ffffffffffffffff", true)

	assert.ErrorIs(t, err, domainrepo.ErrCourseNotFound)
}

func TestCourseRepository_FindRecordsByPlayerID_未プレイを補完する(t *testing.T) {
	repo := setupCourseRepositoryDB(t)
	records, err := repo.FindRecordsByPlayerID(context.Background(), repo.db, 100, false, true)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, uint32(3023238), records[0].Score.Uint32())
	assert.True(t, records[0].IsClear)
}

func TestCourseRepository_FindLatestUpdatedAt_最大値を返す(t *testing.T) {
	// Given
	repo := setupCourseRepositoryDB(t)
	latest := time.Date(2026, 7, 14, 15, 0, 0, 0, time.UTC)
	_, err := repo.db.Exec(`UPDATE courses SET updated_at = ? WHERE id = 11`, latest)
	require.NoError(t, err)

	// When
	result, err := repo.FindLatestUpdatedAt(context.Background(), repo.db)

	// Then
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, latest.Equal(*result))
}

func TestCourseRepository_FindLatestUpdatedAt_コースが無い場合はnil(t *testing.T) {
	// Given
	db := setupTestDB(t)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err := db.Exec(`
		CREATE TABLE courses (
			id INTEGER PRIMARY KEY,
			display_id TEXT NOT NULL UNIQUE,
			official_idx TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			course_class_id INTEGER NOT NULL,
			is_deleted INTEGER NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)
	repo := &courseRepository{db: db}

	// When
	result, err := repo.FindLatestUpdatedAt(context.Background(), db)

	// Then
	require.NoError(t, err)
	assert.Nil(t, result)
}
