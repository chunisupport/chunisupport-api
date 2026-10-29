package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// worldsendLockQueryExecutor は発行SQLを記録し、SQLiteが解釈できない行ロック句だけを除いて実行します。
type worldsendLockQueryExecutor struct {
	repository.Executor
	query string
	args  []any
}

func (e *worldsendLockQueryExecutor) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	e.query = query
	e.args = args
	return e.Executor.QueryRowxContext(ctx, strings.TrimSuffix(query, " FOR UPDATE OF s"), args...)
}

func TestWorldsendChartRepository_FindByDisplayIDForChange_楽曲を排他ロックする(t *testing.T) {
	tests := []struct {
		name string
		// Given
		displayID string
		canceled  bool
		// Then
		wantErr error
	}{
		{name: "削除済み楽曲も行ロック付きで取得する", displayID: "WE001"},
		{name: "存在しない楽曲は未検出エラーを返す", displayID: "WE999", wantErr: repository.ErrSongNotFound},
		{name: "キャンセル済みのコンテキストではキャンセルエラーを返す", displayID: "WE001", canceled: true, wantErr: context.Canceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupWorldsendUpdateDB(t)
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			_, err := db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
				VALUES (1, 'WE001', '削除曲', 'artist', 1, 'WEIDX001', 1, 1)
			`)
			require.NoError(t, err)
			_, err = db.Exec(`
				INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
				VALUES (101, 1, 4, '狂', 1200)
			`)
			require.NoError(t, err)

			tx, err := db.Beginx()
			require.NoError(t, err)
			t.Cleanup(func() { _ = tx.Rollback() })
			exec := &worldsendLockQueryExecutor{Executor: tx}
			repo := &worldsendChartRepository{db: db}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.canceled {
				cancel()
			}

			// When
			got, err := repo.FindByDisplayIDForChange(ctx, exec, tt.displayID)

			// Then
			assert.True(t, strings.HasSuffix(exec.query, " FOR UPDATE OF s"))
			assert.Contains(t, exec.query, "s.display_id = ?")
			assert.Contains(t, exec.query, "s.is_worldsend = 1")
			assert.NotContains(t, exec.query, "is_deleted =")
			assert.Equal(t, []any{tt.displayID}, exec.args)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotNil(t, got.Song)
			assert.Equal(t, "削除曲", got.Song.Title)
			assert.True(t, got.Song.IsDeleted)
			require.NotNil(t, got.Chart)
			require.NotNil(t, got.Chart.LevelStar)
			assert.Equal(t, 4, got.Chart.LevelStar.Int())
		})
	}
}

func TestWorldsendChartRepository_FindByDisplayID_参照取得は行ロックしない(t *testing.T) {
	// Given
	db := setupWorldsendUpdateDB(t)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 'WEIDX001', 1, 0)
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, 4, '狂', 1200)
	`)
	require.NoError(t, err)
	exec := &worldsendLockQueryExecutor{Executor: db}
	repo := &worldsendChartRepository{db: db}

	// When
	got, err := repo.FindByDisplayID(context.Background(), exec, "WE001")

	// Then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.NotContains(t, exec.query, "FOR UPDATE")
	assert.Equal(t, "title", got.Song.Title)
}
