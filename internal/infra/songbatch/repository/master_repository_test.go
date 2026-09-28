package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type courseRow struct {
	ID            int    `db:"id"`
	DisplayID     string `db:"display_id"`
	OfficialIdx   string `db:"official_idx"`
	Name          string `db:"name"`
	CourseClassID int    `db:"course_class_id"`
	IsDeleted     bool   `db:"is_deleted"`
}

func setupCourseSQLite(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	_, err = db.Exec(`
		CREATE TABLE course_classes (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE courses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			display_id TEXT NOT NULL UNIQUE,
			official_idx TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			course_class_id INTEGER NOT NULL,
			is_deleted INTEGER NOT NULL DEFAULT 0
		);
		INSERT INTO course_classes (id, name) VALUES (1, 'CLASS I'), (2, 'CLASS II');
		INSERT INTO courses (id, display_id, official_idx, name, course_class_id, is_deleted) VALUES
			(1, 'display-000000001', 'course-1', 'Course 1', 1, 0),
			(2, 'display-000000002', 'course-2', 'Course 2', 1, 0),
			(3, 'display-000000003', 'course-3', 'Course 3', 1, 1);
	`)
	require.NoError(t, err)
	return db
}

func findCourseRows(t *testing.T, db *sqlx.DB) []courseRow {
	t.Helper()
	rows := []courseRow{}
	require.NoError(t, db.Select(&rows, `SELECT id, display_id, official_idx, name, course_class_id, is_deleted FROM courses ORDER BY id`))
	return rows
}

func newTestCourse(t *testing.T, officialIdx, name, className string) entity.Course {
	t.Helper()
	course, err := entity.NewCourse(officialIdx, name, className)
	require.NoError(t, err)
	return course
}

func TestCourseRepositorySaveAll_既存コースはINSERTせず変更のあるコースだけ更新する(t *testing.T) {
	// Given: INSERT ... ON DUPLICATE KEY UPDATE は既存行でもAUTO_INCREMENTを消費するため、既存コースはUPDATEで反映する
	db := setupCourseSQLite(t)
	courses := []entity.Course{
		newTestCourse(t, "course-1", "Course 1", "CLASS I"),
		newTestCourse(t, "course-2", "Course 2 改", "CLASS II"),
		newTestCourse(t, "course-3", "Course 3", "CLASS I"),
		newTestCourse(t, "course-4", "Course 4", "CLASS II"),
	}

	// When
	err := NewCourseRepository(db).SaveAll(context.Background(), courses)

	// Then
	require.NoError(t, err)
	assert.Equal(t, []courseRow{
		{ID: 1, DisplayID: "display-000000001", OfficialIdx: "course-1", Name: "Course 1", CourseClassID: 1, IsDeleted: false},
		{ID: 2, DisplayID: "display-000000002", OfficialIdx: "course-2", Name: "Course 2 改", CourseClassID: 2, IsDeleted: false},
		{ID: 3, DisplayID: "display-000000003", OfficialIdx: "course-3", Name: "Course 3", CourseClassID: 1, IsDeleted: false},
		{ID: 4, DisplayID: courses[3].DisplayID.String(), OfficialIdx: "course-4", Name: "Course 4", CourseClassID: 2, IsDeleted: false},
	}, findCourseRows(t, db))
}
