package songchart

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nameFolderLockExecutor struct {
	*sqlx.DB
	query string
}

func (e *nameFolderLockExecutor) DriverName() string { return "mysql" }

func (e *nameFolderLockExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	e.query = query
	return e.DB.QueryContext(ctx, strings.TrimSuffix(query, " FOR UPDATE"), args...)
}

func TestSyncNameFolders_同時更新を上書きしないよう最新の楽曲行をロックする(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`CREATE TABLE songs (id INTEGER PRIMARY KEY, title TEXT, reading TEXT, name_folder_id INTEGER NOT NULL DEFAULT 17); INSERT INTO songs (id, title, reading) VALUES (1, 'Title', 'ABC')`)
	require.NoError(t, err)
	executor := &nameFolderLockExecutor{DB: db}
	err = syncMySQLSongNameFolderIDs(context.Background(), executor, map[string]int{"ABCD": 42}, 100)
	require.NoError(t, err)
	assert.Contains(t, executor.query, "ORDER BY id FOR UPDATE")
	var folderID int
	require.NoError(t, db.Get(&folderID, "SELECT name_folder_id FROM songs WHERE id = 1"))
	assert.Equal(t, 42, folderID)
}
