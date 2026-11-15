package repository

import (
	"context"
	"database/sql"
)

// Executor はリポジトリがSQLを実行するためのインターフェースです。
// 本アプリはRDBを前提とするため、標準ライブラリ database/sql の型は公開してよいものとし、
// 特定のDBライブラリ（sqlx など）の型は公開しません。
// Context対応メソッドを通じて、キャンセルやタイムアウトを伝播します。
type Executor interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
	Rebind(query string) string
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
