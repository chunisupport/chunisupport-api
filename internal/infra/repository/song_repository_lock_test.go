package repository

import (
	"context"
	"strings"
	"testing"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sqliteLockExecutor は受け取った SQL を記録し、行ロック句だけを取り除いて SQLite に委譲します。
// SQLite は FOR UPDATE に対応しないため、句の有無は記録した原文で検証します。
type sqliteLockExecutor struct {
	domainrepo.Executor
	songQuery   string
	songArgs    []any
	chartsQuery string
	chartsArgs  []any
}

func (e *sqliteLockExecutor) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	e.songQuery, e.songArgs = query, args
	return e.Executor.GetContext(ctx, dest, strings.ReplaceAll(query, "FOR UPDATE", ""), args...)
}

func (e *sqliteLockExecutor) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	e.chartsQuery, e.chartsArgs = query, args
	return e.Executor.SelectContext(ctx, dest, strings.ReplaceAll(query, "FOR UPDATE", ""), args...)
}

var _ domainrepo.Executor = (*sqliteLockExecutor)(nil)

func TestSongRepositoryFindByOfficialIdxForChange_楽曲と譜面を排他ロックする(t *testing.T) {
	// Given: 取得対象とは別の楽曲と譜面を混ぜて用意する。
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
		VALUES
			(1, 'DISPLAY001', 'Song 1', 'Artist 1', 1, 180, NULL, 'IDX001', NULL, 0, 0, 0),
			(2, 'DISPLAY002', 'Song 2', 'Artist 2', 2, 200, NULL, 'IDX002', NULL, 0, 0, 0)
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer)
		VALUES
			(1, 3, 12.3, 0, 850, '譜面作者A'),
			(1, 4, 13.8, 0, 1050, '譜面作者B'),
			(2, 4, 14.3, 0, 1250, '混ざってはいけない譜面作者')
	`)
	require.NoError(t, err)
	repo := &songRepository{}
	exec := &sqliteLockExecutor{Executor: db}

	// When
	song, err := repo.FindByOfficialIdxForChange(context.Background(), exec, "IDX001")

	// Then: 楽曲の検索条件と行ロック句を検証する。
	require.NoError(t, err)
	require.NotNil(t, song)
	assert.Contains(t, exec.songQuery, "WHERE official_idx = ?")
	assert.Contains(t, exec.songQuery, "FOR UPDATE")
	assert.Equal(t, []any{"IDX001"}, exec.songArgs)

	// Then: 譜面は取得した楽曲の ID で絞り込み、行ロック句を検証する。
	assert.Contains(t, exec.chartsQuery, "WHERE song_id = ?")
	assert.Contains(t, exec.chartsQuery, "FOR UPDATE")
	assert.Equal(t, []any{song.ID}, exec.chartsArgs)

	// Then: 別の楽曲が混ざらず、楽曲と譜面の内容が返ることを検証する。
	assert.Equal(t, 1, song.ID)
	assert.Equal(t, "DISPLAY001", song.DisplayID)
	assert.Equal(t, "Song 1", song.Title)
	assert.Equal(t, "Artist 1", song.Artist)
	assert.Equal(t, "IDX001", song.OfficialIdx)
	require.Len(t, song.Charts, 2)
	for _, chart := range song.Charts {
		assert.Equal(t, song.ID, chart.SongID)
	}
	assert.Equal(t, 3, song.Charts[0].DifficultyID)
	assert.InDelta(t, 12.3, float64(song.Charts[0].Const), 0.001)
	require.NotNil(t, song.Charts[0].NotesDesigner)
	assert.Equal(t, "譜面作者A", *song.Charts[0].NotesDesigner)
	assert.Equal(t, 4, song.Charts[1].DifficultyID)
	assert.InDelta(t, 13.8, float64(song.Charts[1].Const), 0.001)
	require.NotNil(t, song.Charts[1].NotesDesigner)
	assert.Equal(t, "譜面作者B", *song.Charts[1].NotesDesigner)
}
