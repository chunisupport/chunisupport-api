package repository_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	infrarepo "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCourseConcurrencyMySQL(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("COURSE_CONCURRENCY_MYSQL_DSN")
	if dsn == "" {
		t.Skip("COURSE_CONCURRENCY_MYSQL_DSN が未設定です")
	}
	cfg, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	cfg.DBName = ""
	cfg.ParseTime = true
	cfg.Timeout = 5 * time.Second
	admin, err := sqlx.Connect("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	name := fmt.Sprintf("course_concurrency_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4")
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE " + name); require.NoError(t, err) })
	cfg.DBName = name
	db, err := sqlx.Connect("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	db.SetMaxOpenConns(5)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	for _, file := range []string{"000030_create_courses.up.sql", "000031_remove_created_at_from_courses.up.sql", "000033_add_display_id_to_courses.up.sql"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "migration", "mysql", file))
		require.NoError(t, err)
		// プレイヤー記録は検証対象外のため、別集約への外部キーを持つテーブルは作成しない。
		schema := strings.Split(string(data), "CREATE TABLE player_course_records")[0]
		for _, statement := range strings.Split(schema, ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			_, err = db.Exec(statement)
			require.NoError(t, err)
		}
	}
	var version, engine string
	require.NoError(t, db.Get(&version, "SELECT VERSION()"))
	require.NoError(t, db.Get(&engine, "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='courses'"))
	require.Equal(t, "InnoDB", engine)
	t.Logf("MySQL %s / %s", version, engine)
	return db
}

type pausedCourseMySQLRepository struct {
	domainrepo.CourseRepository
	first   sync.Once
	locked  chan struct{}
	release chan struct{}
}

func (r *pausedCourseMySQLRepository) FindByDisplayIDForUpdate(ctx context.Context, exec domainrepo.Executor, id string) (*entity.Course, error) {
	course, err := r.CourseRepository.FindByDisplayIDForUpdate(ctx, exec, id)
	if err != nil {
		return nil, err
	}
	r.first.Do(func() {
		close(r.locked)
		select {
		case <-r.release:
		case <-ctx.Done():
		}
	})
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return course, nil
}

func TestCourseConcurrencyMySQL_MutationsPreserveIndependentFields(t *testing.T) {
	tests := []struct {
		name string
		// Given
		deleted   bool
		editFirst bool
	}{
		{name: "編集が先に行ロックを取得しても削除状態を保持する", deleted: true, editFirst: true},
		{name: "削除が先に行ロックを取得しても編集内容を保持する", deleted: true, editFirst: false},
		{name: "編集が先に行ロックを取得しても復元状態を保持する", deleted: false, editFirst: true},
		{name: "復元が先に行ロックを取得しても編集内容を保持する", deleted: false, editFirst: false},
	}

	db := setupCourseConcurrencyMySQL(t)
	extraClass, err := infrarepo.NewCourseRepository(db).FindClassByName(context.Background(), db, "extra")
	require.NoError(t, err)

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			index := i + 1
			id := fmt.Sprintf("%016x", index)
			_, err := db.Exec("INSERT INTO courses (display_id, official_idx, name, course_class_id, is_deleted) VALUES (?, ?, ?, 1, ?)", id, fmt.Sprint(index), "変更前", !tt.deleted)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			repo := &pausedCourseMySQLRepository{CourseRepository: infrarepo.NewCourseRepository(db), locked: make(chan struct{}), release: make(chan struct{})}
			var release sync.Once
			unblock := func() { release.Do(func() { close(repo.release) }) }
			defer unblock()
			uc := usecase.NewCourseUsecase(db, transaction.NewTransactionManager(db), repo, nil, nil)
			edit := func() error {
				output, err := uc.Update(ctx, id, usecase.UpdateCourseInput{Name: "変更後", Class: "extra"})
				if err == nil && (output.Name != "変更後" || output.Class != "extra") {
					return fmt.Errorf("編集応答が保存内容と一致しません")
				}
				return err
			}
			state := func() error {
				if tt.deleted {
					return uc.Delete(ctx, id)
				}
				return uc.Restore(ctx, id)
			}
			first, second := edit, state
			if !tt.editFirst {
				first, second = state, edit
			}

			// When
			firstDone, secondDone := make(chan error, 1), make(chan error, 1)
			go func() { firstDone <- first() }()
			select {
			case <-repo.locked:
			case err := <-firstDone:
				require.NoError(t, err)
				require.FailNow(t, "先行処理がロックを保持していません")
			case <-ctx.Done():
				require.NoError(t, ctx.Err())
			}
			go func() { secondDone <- second() }()
			require.Eventually(t, func() bool {
				var count int
				err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM performance_schema.data_lock_waits w
                        INNER JOIN performance_schema.data_locks l ON l.ENGINE_LOCK_ID=w.REQUESTING_ENGINE_LOCK_ID AND l.ENGINE=w.ENGINE
                        WHERE l.OBJECT_SCHEMA=DATABASE() AND l.OBJECT_NAME='courses'`)
				return err == nil && count > 0
			}, 5*time.Second, 10*time.Millisecond, "後続処理のコース行ロック待ちを確認できません")
			select {
			case err := <-secondDone:
				require.NoError(t, err)
				require.FailNow(t, "先行処理のコミット前に後続処理が完了しました")
			default:
			}
			unblock()
			for _, done := range []<-chan error{firstDone, secondDone} {
				select {
				case err := <-done:
					require.NoError(t, err)
				case <-ctx.Done():
					require.NoError(t, ctx.Err())
				}
			}

			// Then
			saved, err := repo.CourseRepository.FindByDisplayID(ctx, db, id, true)
			require.NoError(t, err)
			require.NotNil(t, saved)
			assert.Equal(t, "変更後", saved.Name)
			assert.Equal(t, "extra", saved.CourseClass.Name)
			assert.Equal(t, extraClass.ID, saved.CourseClassID)
			assert.Equal(t, tt.deleted, saved.IsDeleted)
		})
	}
}
