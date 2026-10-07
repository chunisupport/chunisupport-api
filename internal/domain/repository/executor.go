package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Executor は *sqlx.DB と *sqlx.Tx に共通するSQL実行インターフェースです。
// Context対応メソッドを通じて、キャンセルやタイムアウトを伝播します。
// リポジトリの契約はdomain層が定義する責務を持つため、sqlxへの依存を例外的に許容してここに配置しています。
type Executor interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
	Rebind(query string) string
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
	QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row
}

var _ Executor = (*sqlx.DB)(nil)
var _ Executor = (*sqlx.Tx)(nil)
