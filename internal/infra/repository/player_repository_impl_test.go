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

type noRowsWrappedExecutor struct{}

func (e *noRowsWrappedExecutor) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	return errors.Join(errors.New("wrapped no rows"), sql.ErrNoRows)
}

func (e *noRowsWrappedExecutor) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	return nil
}

func (e *noRowsWrappedExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (e *noRowsWrappedExecutor) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	return nil, nil
}

func (e *noRowsWrappedExecutor) Rebind(query string) string {
	return query
}

func (e *noRowsWrappedExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}

func (e *noRowsWrappedExecutor) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	return nil, nil
}

func (e *noRowsWrappedExecutor) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	return nil
}

var _ domainrepo.Executor = (*noRowsWrappedExecutor)(nil)

func TestFindByUserID_ReturnsNilWhenWrappedNoRows(t *testing.T) {
	repo := &playerRepository{}
	exec := &noRowsWrappedExecutor{}

	player, err := repo.FindByUserID(context.Background(), exec, 10)
	require.NoError(t, err)
	require.Nil(t, player)
}
