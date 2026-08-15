package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type favoriteSongReadRow struct {
	DisplayID   string  `db:"display_id"`
	Title       string  `db:"title"`
	Jacket      *string `db:"jacket"`
	FavoritedAt string  `db:"favorited_at"`
}

func TestPlayerFavoriteSongQueryServiceWrapsPersistenceErrors(t *testing.T) {
	tests := []struct {
		name string
		act  func(context.Context) error
	}{
		{
			name: "お気に入り一覧取得の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context) error {
				qs := &PlayerFavoriteSongQueryService{}
				_, err := qs.ListWithSongDetailsByPlayerID(ctx, closedSQLiteExecutor(t), 1)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.act(context.Background())
			require.Error(t, err)
			assert.ErrorIs(t, err, domainrepo.ErrRepositoryOperationFailed)
			assert.NotErrorIs(t, err, sql.ErrConnDone)
		})
	}
}

func setupFavoriteSongQueryTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	_, err = db.Exec(`
		CREATE TABLE player_favorite_songs (
			player_id INTEGER NOT NULL,
			song_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (player_id, song_id)
		);
		CREATE TABLE songs (
			id INTEGER PRIMARY KEY,
			display_id TEXT NOT NULL,
			title TEXT NOT NULL,
			artist TEXT NOT NULL DEFAULT '',
			jacket TEXT,
			is_worldsend INTEGER NOT NULL DEFAULT 0,
			is_deleted INTEGER NOT NULL DEFAULT 0
		);
		INSERT INTO songs (id, display_id, title, artist, jacket, is_worldsend, is_deleted) VALUES
			(10, '0000000000000001', '楽曲A', 'Artist1', 'jacket_a.jpg', 0, 0),
			(20, '0000000000000002', '楽曲B', 'Artist2', 'jacket_b.jpg', 0, 0),
			(30, '0000000000000003', '削除済み楽曲', 'Artist3', NULL, 0, 1),
			(40, '0000000000000004', 'WORLDS END', 'Artist4', NULL, 1, 0);
		INSERT INTO player_favorite_songs (player_id, song_id, created_at) VALUES
			(1, 10, '2026-07-05T12:00:00Z'),
			(1, 20, '2026-07-04T12:00:00Z'),
			(1, 30, '2026-07-03T12:00:00Z'),
			(1, 40, '2026-07-02T12:00:00Z'),
			(2, 10, '2026-07-01T12:00:00Z');
	`)
	require.NoError(t, err)
	return db
}

func TestFavoriteSongQuery_JoinsSongDetails(t *testing.T) {
	db := setupFavoriteSongQueryTestDB(t)
	ctx := context.Background()

	var rows []favoriteSongReadRow
	err := sqlx.SelectContext(ctx, db, &rows, `
		SELECT s.display_id, s.title, s.jacket, pfs.created_at AS favorited_at
		FROM player_favorite_songs pfs
		INNER JOIN songs s ON s.id = pfs.song_id
		WHERE pfs.player_id = ?
		  AND s.is_deleted = FALSE
		  AND s.is_worldsend = FALSE
		ORDER BY pfs.created_at DESC, s.display_id ASC
	`, 1)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "0000000000000001", rows[0].DisplayID)
	assert.Equal(t, "楽曲A", rows[0].Title)
	require.NotNil(t, rows[0].Jacket)
	assert.Equal(t, "jacket_a.jpg", *rows[0].Jacket)
	parsed0, err := time.Parse(time.RFC3339Nano, rows[0].FavoritedAt)
	require.NoError(t, err)
	assert.Equal(t, 2026, parsed0.Year())

	assert.Equal(t, "0000000000000002", rows[1].DisplayID)
	assert.Equal(t, "楽曲B", rows[1].Title)
	require.NotNil(t, rows[1].Jacket)
	assert.Equal(t, "jacket_b.jpg", *rows[1].Jacket)
}

func TestFavoriteSongQuery_EmptyForPlayerWithoutFavorites(t *testing.T) {
	db := setupFavoriteSongQueryTestDB(t)

	var rows []favoriteSongReadRow
	err := sqlx.SelectContext(context.Background(), db, &rows, `
		SELECT s.display_id, s.title, s.jacket, pfs.created_at AS favorited_at
		FROM player_favorite_songs pfs
		INNER JOIN songs s ON s.id = pfs.song_id
		WHERE pfs.player_id = ?
		  AND s.is_deleted = FALSE
		  AND s.is_worldsend = FALSE
		ORDER BY pfs.created_at DESC, s.display_id ASC
	`, 3)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestFavoriteSongQuery_ExcludesDeletedAndWorldsend(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	_, err = db.Exec(`
		CREATE TABLE player_favorite_songs (
			player_id INTEGER NOT NULL,
			song_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (player_id, song_id)
		);
		CREATE TABLE songs (
			id INTEGER PRIMARY KEY,
			display_id TEXT NOT NULL,
			title TEXT NOT NULL,
			artist TEXT NOT NULL DEFAULT '',
			jacket TEXT,
			is_worldsend INTEGER NOT NULL DEFAULT 0,
			is_deleted INTEGER NOT NULL DEFAULT 0
		);
		INSERT INTO songs (id, display_id, title, artist, jacket, is_worldsend, is_deleted) VALUES
			(10, 'D001', 'Deleted', '', NULL, 0, 1),
			(20, 'W001', 'Worldsend', '', NULL, 1, 0),
			(30, 'N001', 'Normal', '', NULL, 0, 0);
		INSERT INTO player_favorite_songs (player_id, song_id, created_at) VALUES
			(1, 10, '2026-07-05T12:00:00Z'),
			(1, 20, '2026-07-04T12:00:00Z'),
			(1, 30, '2026-07-03T12:00:00Z');
	`)
	require.NoError(t, err)

	// Only Normal song should be returned
	var rows []favoriteSongReadRow
	err = sqlx.SelectContext(context.Background(), db, &rows, `
		SELECT s.display_id, s.title, s.jacket, pfs.created_at AS favorited_at
		FROM player_favorite_songs pfs
		INNER JOIN songs s ON s.id = pfs.song_id
		WHERE pfs.player_id = ?
		  AND s.is_deleted = FALSE
		  AND s.is_worldsend = FALSE
		ORDER BY pfs.created_at DESC, s.display_id ASC
	`, 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "N001", rows[0].DisplayID)
}
