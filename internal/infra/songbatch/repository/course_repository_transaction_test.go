package repository

import (
	"context"
	"errors"
	"testing"

	apirepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// TestTransactional_Rollback はエラー時にロールバックされることを確認
func TestTransactional_Rollback(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// テーブル作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	tm := transaction.NewTransactionManager(db)

	expectedErr := errors.New("intentional error")

	// トランザクション内で INSERT した後にエラーを返す
	err = tm.Transactional(ctx, func(tx apirepo.Executor) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO test_table (value) VALUES (?)", "test_value")
		if err != nil {
			return err
		}
		return expectedErr
	})

	// エラーが返されることを確認
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be '%v', got '%v'", expectedErr, err)
	}

	// 検証: データがロールバックされている
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM test_table")
	if err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 rows (rolled back), got %d", count)
	}
}

func TestTransactional_CourseRepositoryFailureRollsBackEarlierWrites(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE markers (value TEXT NOT NULL);
		CREATE TABLE course_classes (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE courses (
			display_id TEXT NOT NULL,
			official_idx TEXT NOT NULL,
			name TEXT NOT NULL,
			course_class_id INTEGER NOT NULL,
			is_deleted INTEGER NOT NULL
		)
	`); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	course, err := entity.NewCourse("course-1", "Course", "unknown")
	if err != nil {
		t.Fatalf("NewCourse: %v", err)
	}
	err = transaction.NewTransactionManager(db).Transactional(ctx, func(tx apirepo.Executor) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO markers (value) VALUES (?)`, "before-course-sync"); err != nil {
			return err
		}
		return NewCourseRepository(tx).SaveAll(ctx, []entity.Course{course})
	})
	if err == nil {
		t.Fatal("expected course repository failure")
	}

	var count int
	if err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM markers`); err != nil {
		t.Fatalf("failed to count markers: %v", err)
	}
	if count != 0 {
		t.Fatalf("earlier writes were not rolled back: count=%d", count)
	}
}
