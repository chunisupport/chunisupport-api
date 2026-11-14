package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	apirepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func transactionTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, value TEXT NOT NULL)")
	require.NoError(t, err)
	return db
}

func TestTransactional_Commit(t *testing.T) {
	db := transactionTestDB(t)
	err := NewTransactionManager(db).Transactional(context.Background(), func(tx apirepo.Executor) error {
		_, err := tx.ExecContext(context.Background(), "INSERT INTO test_table (value) VALUES (?), (?)", "first", "second")
		return err
	})
	require.NoError(t, err)
	var values []string
	require.NoError(t, db.Select(&values, "SELECT value FROM test_table ORDER BY id"))
	assert.Equal(t, []string{"first", "second"}, values)
}

func TestTransactional_NestedError(t *testing.T) {
	db := transactionTestDB(t)
	cause := errors.New("挿入後の失敗")
	err := NewTransactionManager(db).Transactional(context.Background(), func(tx apirepo.Executor) error {
		_, err := tx.ExecContext(context.Background(), "INSERT INTO test_table (value) VALUES (?), (?)", "first", "second")
		if err != nil {
			return err
		}
		return cause
	})
	assert.ErrorIs(t, err, cause)
	var count int
	require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM test_table"))
	assert.Zero(t, count)
}

func TestTransactional_PanicRecovery(t *testing.T) {
	db := transactionTestDB(t)
	marker := &struct{ message string }{"元のpanic"}
	reached := false
	require.PanicsWithValue(t, marker, func() {
		_ = NewTransactionManager(db).Transactional(context.Background(), func(tx apirepo.Executor) error {
			_, err := tx.ExecContext(context.Background(), "INSERT INTO test_table (value) VALUES (?)", "before panic")
			if err != nil {
				return err
			}
			reached = true
			panic(marker)
		})
	})
	assert.True(t, reached)
	var count int
	require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM test_table"))
	assert.Zero(t, count)
}

func TestTransactional_Begin失敗ならコールバックを実行しない(t *testing.T) {
	db := transactionTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	err := NewTransactionManager(db).Transactional(ctx, func(apirepo.Executor) error { called = true; return nil })
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, called)
}

func TestTransactional_Commit失敗を呼び出し元に返す(t *testing.T) {
	db := transactionTestDB(t)
	reached := false
	err := NewTransactionManager(db).Transactional(context.Background(), func(exec apirepo.Executor) error {
		// 実トランザクションを先に終了させ、外側のCommitを確実に失敗させる。
		tx, ok := exec.(*sqlx.Tx)
		require.True(t, ok)
		require.NoError(t, tx.Rollback())
		reached = true
		return nil
	})
	assert.True(t, reached)
	assert.ErrorIs(t, err, sql.ErrTxDone)
}

func TestTransactional_Rollback失敗でも元のエラーとpanicを保持する(t *testing.T) {
	for _, panics := range []bool{false, true} {
		name := "エラー"
		if panics {
			name = "panic"
		}
		t.Run(name, func(t *testing.T) {
			db := transactionTestDB(t)
			cause := errors.New("元の失敗")
			run := func() error {
				return NewTransactionManager(db).Transactional(context.Background(), func(exec apirepo.Executor) error {
					tx, ok := exec.(*sqlx.Tx)
					require.True(t, ok)
					require.NoError(t, tx.Rollback())
					if panics {
						panic(cause)
					}
					return cause
				})
			}
			if panics {
				require.PanicsWithValue(t, cause, func() { _ = run() })
			} else {
				assert.ErrorIs(t, run(), cause)
			}
		})
	}
}
