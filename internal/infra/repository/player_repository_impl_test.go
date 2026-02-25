package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

type baseExecutor struct{}

func (e *baseExecutor) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	panic("unexpected call to GetContext")
}

func (e *baseExecutor) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	panic("unexpected call to SelectContext")
}

func (e *baseExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	panic("unexpected call to ExecContext")
}

func (e *baseExecutor) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	panic("unexpected call to NamedExecContext")
}

func (e *baseExecutor) Rebind(query string) string {
	panic("unexpected call to Rebind")
}

func (e *baseExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	panic("unexpected call to QueryContext")
}

func (e *baseExecutor) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	panic("unexpected call to QueryxContext")
}

func (e *baseExecutor) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	panic("unexpected call to QueryRowxContext")
}

var _ domainrepo.Executor = (*baseExecutor)(nil)

type noRowsWrappedExecutor struct {
	baseExecutor
}

func (e *noRowsWrappedExecutor) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	return errors.Join(errors.New("wrapped no rows"), sql.ErrNoRows)
}

var _ domainrepo.Executor = (*noRowsWrappedExecutor)(nil)

func TestFindByUserID_ReturnsNilWhenWrappedNoRows(t *testing.T) {
	repo := &playerRepository{}
	exec := &noRowsWrappedExecutor{}

	player, err := repo.FindByUserID(context.Background(), exec, 10)
	require.NoError(t, err)
	require.Nil(t, player)
}
