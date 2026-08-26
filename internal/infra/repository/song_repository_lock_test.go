package repository

import (
	"context"
	"database/sql"
	"testing"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSongRepositoryFindByOfficialIdxForChange_楽曲と譜面を排他ロックする(t *testing.T) {
	// Given
	exec := &songLockExecutor{}
	repo := &songRepository{}

	// When
	song, err := repo.FindByOfficialIdxForChange(context.Background(), exec, "IDX001")

	// Then
	require.NoError(t, err)
	require.NotNil(t, song)
	assert.Equal(t, "IDX001", song.OfficialIdx)
	assert.Contains(t, exec.songQuery, "WHERE official_idx = ?")
	assert.Contains(t, exec.songQuery, "FOR UPDATE")
	assert.Contains(t, exec.chartsQuery, "FOR UPDATE")
}

type songLockExecutor struct {
	songQuery   string
	chartsQuery string
}

func (e *songLockExecutor) GetContext(_ context.Context, dest any, query string, _ ...any) error {
	e.songQuery = query
	row := dest.(*songRow)
	*row = songRow{ID: 1, DisplayID: "DISPLAY001", Title: "Title", Artist: "Artist", OfficialIdx: "IDX001"}
	return nil
}

func (e *songLockExecutor) SelectContext(_ context.Context, dest any, query string, _ ...any) error {
	e.chartsQuery = query
	rows := dest.(*[]chartRow)
	*rows = []chartRow{}
	return nil
}

func (e *songLockExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	panic("unexpected call")
}

func (e *songLockExecutor) NamedExecContext(context.Context, string, any) (sql.Result, error) {
	panic("unexpected call")
}

func (e *songLockExecutor) Rebind(query string) string {
	return query
}

func (e *songLockExecutor) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	panic("unexpected call")
}

func (e *songLockExecutor) QueryxContext(context.Context, string, ...any) (*sqlx.Rows, error) {
	panic("unexpected call")
}

func (e *songLockExecutor) QueryRowxContext(context.Context, string, ...any) *sqlx.Row {
	panic("unexpected call")
}

var _ domainrepo.Executor = (*songLockExecutor)(nil)
