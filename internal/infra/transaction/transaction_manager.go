package transaction

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
)

// transactionManager はトランザクションを管理するインフラ層の実装です。
type transactionManager struct {
	db *sqlx.DB
}

// NewTransactionManager は新しいTransactionManagerを作成します。
func NewTransactionManager(db *sqlx.DB) *transactionManager {
	return &transactionManager{db: db}
}

// Transactional は、渡された関数をトランザクション内で実行します。
// 関数内でエラーが発生した場合はトランザクションをロールバックし、そうでなければコミットします。
func (tm *transactionManager) Transactional(ctx context.Context, f func(tx repository.Executor) error) (err error) {
	slog.Debug("Beginning transaction")
	tx, err := tm.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			slog.Debug("Rolling back transaction due to panic")
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("Failed to rollback transaction after panic", "error", rbErr)
			}
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			slog.Debug("Rolling back transaction due to error", "error", err)
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("Failed to rollback transaction", "error", rbErr)
			} // err is non-nil; don't change it
		} else {
			slog.Debug("Committing transaction")
			err = tx.Commit() // if Commit returns error, update err
			if err == nil {
				slog.Debug("Transaction committed successfully")
			}
		}
	}()

	err = f(tx)
	return err
}
