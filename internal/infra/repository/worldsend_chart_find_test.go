package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFindAll_正常系(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	songUpdatedAt := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	chartUpdatedAt := songUpdatedAt.Add(2 * time.Hour)

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted, updated_at)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0, ?)
	`, songUpdatedAt.Format(time.RFC3339Nano))
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, updated_at)
		VALUES (101, 1, 4, '速', 1200, ?)
	`, chartUpdatedAt.Format(time.RFC3339Nano))
	require.NoError(t, err)

	repo := &worldsendChartRepository{db: db}

	got, err := repo.FindAll(context.Background(), db, false)

	require.NoError(t, err)
	require.Len(t, got, 1)
}
