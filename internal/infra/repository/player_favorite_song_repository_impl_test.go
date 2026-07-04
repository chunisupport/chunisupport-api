package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestPlayerFavoriteSongRepositoryWrapsPersistenceErrors(t *testing.T) {
	tests := []struct {
		name string
		act  func(context.Context) error
	}{
		{
			name: "お気に入り件数取得の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context) error {
				repo := &PlayerFavoriteSongRepository{}
				_, err := repo.CountByPlayerID(ctx, closedSQLiteExecutor(t), 1)
				return err
			},
		},
		{
			name: "お気に入り存在確認の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context) error {
				repo := &PlayerFavoriteSongRepository{}
				_, err := repo.Exists(ctx, closedSQLiteExecutor(t), 1, 1)
				return err
			},
		},
		{
			name: "お気に入り保存の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context) error {
				repo := &PlayerFavoriteSongRepository{}
				return repo.Save(ctx, closedSQLiteExecutor(t), &entity.PlayerFavoriteSong{PlayerID: 1, SongID: 1})
			},
		},
		{
			name: "お気に入り削除の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context) error {
				repo := &PlayerFavoriteSongRepository{}
				return repo.Delete(ctx, closedSQLiteExecutor(t), 1, 1)
			},
		},
		{
			name: "楽曲単位削除の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context) error {
				repo := &PlayerFavoriteSongRepository{}
				return repo.DeleteBySongID(ctx, closedSQLiteExecutor(t), 1)
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

type favoriteSongRowForTest struct {
	PlayerID  int    `db:"player_id"`
	SongID    int    `db:"song_id"`
	CreatedAt string `db:"created_at"`
}

func setupFavoriteSongTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	_, err = db.Exec(`
		CREATE TABLE player_favorite_songs (
			player_id INTEGER NOT NULL,
			song_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (player_id, song_id)
		)
	`)
	require.NoError(t, err)
	return db
}

func TestPlayerFavoriteSongRepository_SaveAndCount(t *testing.T) {
	db := setupFavoriteSongTestDB(t)
	ctx := context.Background()
	repo := &PlayerFavoriteSongRepository{}

	err := repo.Save(ctx, db, &entity.PlayerFavoriteSong{PlayerID: 1, SongID: 10, CreatedAt: time.Now()})
	require.NoError(t, err)

	count, err := repo.CountByPlayerID(ctx, db, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count2, err := repo.CountByPlayerID(ctx, db, 2)
	require.NoError(t, err)
	assert.Equal(t, 0, count2)
}

func TestPlayerFavoriteSongRepository_Exists(t *testing.T) {
	db := setupFavoriteSongTestDB(t)
	ctx := context.Background()
	repo := &PlayerFavoriteSongRepository{}

	require.NoError(t, repo.Save(ctx, db, &entity.PlayerFavoriteSong{PlayerID: 1, SongID: 10, CreatedAt: time.Now()}))

	exists, err := repo.Exists(ctx, db, 1, 10)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.Exists(ctx, db, 1, 20)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPlayerFavoriteSongRepository_DeleteIsIdempotent(t *testing.T) {
	db := setupFavoriteSongTestDB(t)
	ctx := context.Background()
	repo := &PlayerFavoriteSongRepository{}

	require.NoError(t, repo.Save(ctx, db, &entity.PlayerFavoriteSong{PlayerID: 1, SongID: 10, CreatedAt: time.Now()}))

	require.NoError(t, repo.Delete(ctx, db, 1, 10))
	require.NoError(t, repo.Delete(ctx, db, 1, 10))

	count, err := repo.CountByPlayerID(ctx, db, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPlayerFavoriteSongRepository_DeleteBySongID(t *testing.T) {
	db := setupFavoriteSongTestDB(t)
	ctx := context.Background()
	repo := &PlayerFavoriteSongRepository{}

	require.NoError(t, repo.Save(ctx, db, &entity.PlayerFavoriteSong{PlayerID: 1, SongID: 10, CreatedAt: time.Now()}))
	require.NoError(t, repo.Save(ctx, db, &entity.PlayerFavoriteSong{PlayerID: 2, SongID: 10, CreatedAt: time.Now()}))
	require.NoError(t, repo.Save(ctx, db, &entity.PlayerFavoriteSong{PlayerID: 1, SongID: 20, CreatedAt: time.Now()}))

	require.NoError(t, repo.DeleteBySongID(ctx, db, 10))

	count1, err := repo.CountByPlayerID(ctx, db, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, count1)

	count2, err := repo.CountByPlayerID(ctx, db, 2)
	require.NoError(t, err)
	assert.Equal(t, 0, count2)
}

func TestPlayerFavoriteSongRepository_Validate(t *testing.T) {
	db := setupFavoriteSongTestDB(t)
	ctx := context.Background()
	repo := &PlayerFavoriteSongRepository{}

	err := repo.Save(ctx, db, &entity.PlayerFavoriteSong{PlayerID: 0, SongID: 10})
	require.Error(t, err)
	assert.ErrorContains(t, err, "player_id")
}
