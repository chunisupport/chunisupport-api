package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindByDisplayIDs_LoadsChartsForEachSong(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES
			(1, 'DISPLAY001', 'Song 1', 'Artist 1', 1, 180, NULL, 'IDX001', NULL, 0, 0),
			(2, 'DISPLAY002', 'Song 2', 'Artist 2', 2, 200, NULL, 'IDX002', NULL, 0, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES
			(1, 3, 12.3, 0, 850),
			(1, 4, 13.8, 0, 1050),
			(2, 4, 14.3, 0, 1250)
	`)
	require.NoError(t, err)

	repo := &songRepository{db: db}
	songs, err := repo.FindByDisplayIDs(ctx, db, []string{"DISPLAY001", "DISPLAY002"})
	require.NoError(t, err)
	require.Len(t, songs, 2)

	songsByDisplayID := make(map[string]*entity.Song, len(songs))
	for _, song := range songs {
		require.NotNil(t, song.Charts)
		songsByDisplayID[song.DisplayID] = song
	}

	song1, ok := songsByDisplayID["DISPLAY001"]
	require.True(t, ok)
	require.Len(t, song1.Charts, 2)

	song1ChartsByDifficulty := make(map[int]*entity.Chart, len(song1.Charts))
	for _, chart := range song1.Charts {
		song1ChartsByDifficulty[chart.DifficultyID] = chart
	}

	chart3, ok := song1ChartsByDifficulty[3]
	require.True(t, ok)
	assert.Equal(t, 1, chart3.SongID)
	assert.InDelta(t, 12.3, float64(chart3.Const), 0.001)

	chart4, ok := song1ChartsByDifficulty[4]
	require.True(t, ok)
	assert.Equal(t, 1, chart4.SongID)
	assert.InDelta(t, 13.8, float64(chart4.Const), 0.001)

	// ドメインサービスによる譜面集約が適用されていることを検証
	assert.InDelta(t, 13.8, song1.MaxChartConst, 0.001)
	assert.False(t, song1.IsMaxOPUnknown)

	song2, ok := songsByDisplayID["DISPLAY002"]
	require.True(t, ok)
	require.Len(t, song2.Charts, 1)
	assert.Equal(t, 4, song2.Charts[0].DifficultyID)
	assert.Equal(t, 2, song2.Charts[0].SongID)
	assert.InDelta(t, 14.3, float64(song2.Charts[0].Const), 0.001)

	// Song2の集約結果も検証
	assert.InDelta(t, 14.3, song2.MaxChartConst, 0.001)
	assert.False(t, song2.IsMaxOPUnknown)
}

func TestFindByDisplayIDs_ReturnsEmptyChartsWhenSongHasNoCharts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES
			(1, 'DISPLAY001', 'Song 1', 'Artist 1', 1, 180, NULL, 'IDX001', NULL, 0, 0)
	`)
	require.NoError(t, err)

	repo := &songRepository{db: db}
	songs, err := repo.FindByDisplayIDs(ctx, db, []string{"DISPLAY001"})
	require.NoError(t, err)
	require.Len(t, songs, 1)

	assert.NotNil(t, songs[0].Charts)
	assert.Len(t, songs[0].Charts, 0)

	// 譜面なし楽曲の集約結果を検証（ゼロ値）
	assert.Equal(t, float64(0), songs[0].MaxChartConst)
	assert.False(t, songs[0].IsMaxOPUnknown)
}

func TestFindByDisplayIDs_SetsIsMaxOPUnknownWhenMasterOrUltimaIsConstUnknown(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES
			(1, 'DISPLAY001', 'Song 1', 'Artist 1', 1, 180, NULL, 'IDX001', NULL, 0, 0)
	`)
	require.NoError(t, err)

	// MASTER(4)がknown、ULTIMA(5)がunknown
	_, err = db.Exec(`
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES
			(1, 4, 14.6, 0, 1050),
			(1, 5, 14.5, 1, 1100)
	`)
	require.NoError(t, err)

	repo := &songRepository{db: db}
	songs, err := repo.FindByDisplayIDs(ctx, db, []string{"DISPLAY001"})
	require.NoError(t, err)
	require.Len(t, songs, 1)

	// ULTIMAがunknownなのでIsMaxOPUnknown=trueであること
	assert.InDelta(t, 14.6, songs[0].MaxChartConst, 0.001)
	assert.True(t, songs[0].IsMaxOPUnknown)
}
