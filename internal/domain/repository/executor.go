package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Executor は *sqlx.DB と *sqlx.Tx の両方を扱うためのインターフェースです。
// これにより、リポジトリのメソッドはトランザクションの内外で再利用可能になります。
// Clean Architectureの原則に従い、このインターフェースはdomain層に配置され、
// infra層の実装詳細（sqlx）に依存していますが、domain層がリポジトリの契約を定義する責務を持つため、
// ここに配置されています。
//
// 全てのメソッドはContext対応版を使用し、リクエストのキャンセルやタイムアウトを適切に伝播します。
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

// *sqlx.DB と *sqlx.Tx が Executor インターフェースを実装していることを確認します。
var _ Executor = (*sqlx.DB)(nil)
var _ Executor = (*sqlx.Tx)(nil)
