package repository

import (
	"context"
	"strings"
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
	_, err := repo.db.Exec(`INSERT INTO courses VALUES (12, '0000000000000012', '50030', '未プレイコース', 7, 0, '2026-07-01 00:00:00')`)
	require.NoError(t, err)

	records, err := repo.FindRecordsByPlayerID(context.Background(), repo.db, 100, false)

	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, uint32(3023238), records[0].Score.Uint32())
	assert.True(t, records[0].IsClear)
	assert.Equal(t, "未プレイコース", records[1].Course.Name)
	assert.Equal(t, uint32(0), records[1].Score.Uint32())
	assert.False(t, records[1].IsClear)
	assert.True(t, records[1].UpdatedAt.IsZero())
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

type courseLockQueryExecutor struct {
	domainrepo.Executor
	query string
	args  []any
}

func (e *courseLockQueryExecutor) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	e.query, e.args = query, args
	// SQLiteは行ロック構文に対応しないため、検証対象の句を記録してから除去する。
	return e.Executor.GetContext(ctx, dest, strings.TrimSuffix(query, " FOR UPDATE"), args...)
}

func TestCourseRepository_FindByDisplayIDForUpdate(t *testing.T) {
	tests := []struct {
		name string
		// Given
		id       string
		canceled bool
		// Then
		wantErr error
	}{
		{name: "削除済みコースも行ロック付きで取得する", id: "0000000000000011"},
		{name: "存在しないコースは未検出エラーを返す", id: "ffffffffffffffff", wantErr: domainrepo.ErrCourseNotFound},
		{name: "キャンセル済みのコンテキストではキャンセルエラーを返す", id: "0000000000000011", canceled: true, wantErr: context.Canceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := setupCourseRepositoryDB(t)
			tx, err := repo.db.Beginx()
			require.NoError(t, err)
			defer tx.Rollback()
			exec := &courseLockQueryExecutor{Executor: tx}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.canceled {
				cancel()
			}

			// When
			course, err := repo.FindByDisplayIDForUpdate(ctx, exec, tt.id)

			// Then
			assert.True(t, strings.HasSuffix(exec.query, " FOR UPDATE"))
			assert.NotContains(t, exec.query, "is_deleted = FALSE")
			assert.Equal(t, []any{tt.id}, exec.args)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, course)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, course)
			assert.True(t, course.IsDeleted)
			assert.Equal(t, "削除コース", course.Name)
			assert.Equal(t, "extra", course.CourseClass.Name)
		})
	}
}
