package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindByDisplayID_ScansLevelStarValueObject(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, 4, '狂', 1200)
	`)
	require.NoError(t, err)

	repo := &worldsendChartRepository{db: db}

	got, err := repo.FindByDisplayID(context.Background(), db, "WE001")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Chart)
	require.NotNil(t, got.Chart.LevelStar)
	assert.Equal(t, 4, got.Chart.LevelStar.Int())
}

func TestFindByDisplayID_ScansNilLevelStarAsNil(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, NULL, '狂', 1200)
	`)
	require.NoError(t, err)

	repo := &worldsendChartRepository{db: db}

	got, err := repo.FindByDisplayID(context.Background(), db, "WE001")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Chart)
	assert.Nil(t, got.Chart.LevelStar)
}

func TestGetLatestUpdatedAt(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()
	activeSongUpdatedAt := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	activeChartUpdatedAt := activeSongUpdatedAt.Add(2 * time.Hour)
	deletedSongUpdatedAt := activeChartUpdatedAt.Add(2 * time.Hour)

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted, updated_at)
		VALUES
			(1, 'WE001', 'title1', 'artist1', 1, 180, NULL, 'WEIDX001', 'we1.png', 1, 0, ?),
			(2, 'WE002', 'title2', 'artist2', 1, 180, NULL, 'WEIDX002', 'we2.png', 1, 1, ?)
	`, activeSongUpdatedAt.Format(time.RFC3339Nano), deletedSongUpdatedAt.Format(time.RFC3339Nano))
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, updated_at)
		VALUES
			(101, 1, 4, '狂', 1200, ?),
			(102, 2, 5, '狂', 1500, ?)
	`, activeChartUpdatedAt.Format(time.RFC3339Nano), deletedSongUpdatedAt.Add(time.Hour).Format(time.RFC3339Nano))
	require.NoError(t, err)

	repo := &worldsendChartRepository{db: db}

	t.Run("削除済み除外時でも削除済み楽曲自体の updated_at を含む最大時刻を返す", func(t *testing.T) {
		updatedAt, err := repo.GetLatestUpdatedAt(ctx, db, false)
		require.NoError(t, err)
		require.NotNil(t, updatedAt)
		assert.True(t, deletedSongUpdatedAt.Equal(updatedAt.UTC()))
	})

	t.Run("削除済み含む時は削除済みも含めた最大時刻を返す", func(t *testing.T) {
		updatedAt, err := repo.GetLatestUpdatedAt(ctx, db, true)
		require.NoError(t, err)
		require.NotNil(t, updatedAt)
		expected := deletedSongUpdatedAt.Add(time.Hour)
		assert.True(t, expected.Equal(updatedAt.UTC()))
	})

	t.Run("対象データがない場合はnilを返す", func(t *testing.T) {
		emptyDB := setupWorldsendUpdateDB(t)
		defer emptyDB.Close()

		repo := &worldsendChartRepository{db: emptyDB}
		updatedAt, err := repo.GetLatestUpdatedAt(ctx, emptyDB, false)
		require.NoError(t, err)
		assert.Nil(t, updatedAt)
	})
}
