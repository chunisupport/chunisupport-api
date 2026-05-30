package usecase

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"
)

// MockExecutor は repository.Executor のテスト用モックです。
type MockExecutor struct {
	mock.Mock
}

func (m *MockExecutor) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	arg := m.Called(ctx, dest, query, args)
	return arg.Error(0)
}

func (m *MockExecutor) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	arg := m.Called(ctx, dest, query, args)
	return arg.Error(0)
}

func (m *MockExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	arg := m.Called(ctx, query, args)
	if arg.Get(0) == nil {
		return nil, arg.Error(1)
	}
	return arg.Get(0).(sql.Result), arg.Error(1)
}

func (m *MockExecutor) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	args := m.Called(ctx, query, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockExecutor) Rebind(query string) string {
	args := m.Called(query)
	return args.String(0)
}

func (m *MockExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	arg := m.Called(ctx, query, args)
	if arg.Get(0) == nil {
		return nil, arg.Error(1)
	}
	return arg.Get(0).(*sql.Rows), arg.Error(1)
}

func (m *MockExecutor) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	arg := m.Called(ctx, query, args)
	if arg.Get(0) == nil {
		return nil, arg.Error(1)
	}
	return arg.Get(0).(*sqlx.Rows), arg.Error(1)
}

func (m *MockExecutor) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	arg := m.Called(ctx, query, args)
	if arg.Get(0) == nil {
		return nil
	}
	return arg.Get(0).(*sqlx.Row)
}
