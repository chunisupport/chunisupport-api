package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestPlayerLockedSongRepositoryWrapsPersistenceErrors(t *testing.T) {
	tests := []struct {
		name string
		act  func(context.Context, *PlayerLockedSongRepository) error
	}{
		{
			name: "未解禁楽曲一覧取得の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context, repo *PlayerLockedSongRepository) error {
				_, err := repo.ListByPlayerID(ctx, closedSQLiteExecutor(t), 1)
				return err
			},
		},
		{
			name: "未解禁楽曲作成の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context, repo *PlayerLockedSongRepository) error {
				return repo.Create(ctx, closedSQLiteExecutor(t), &entity.PlayerLockedSong{
					PlayerID: 1,
					SongID:   1,
					IsUltima: true,
				})
			},
		},
		{
			name: "未解禁楽曲削除の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context, repo *PlayerLockedSongRepository) error {
				return repo.Delete(ctx, closedSQLiteExecutor(t), 1, 1, true)
			},
		},
		{
			name: "未解禁楽曲表示用一覧取得の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context, repo *PlayerLockedSongRepository) error {
				_, err := repo.ListWithSongDisplayIDAndTitleByPlayerID(ctx, closedSQLiteExecutor(t), 1)
				return err
			},
		},
		{
			name: "楽曲ID解決の永続化エラーはドメイン定義エラーになる",
			act: func(ctx context.Context, repo *PlayerLockedSongRepository) error {
				_, err := repo.ResolveSongIDByDisplayID(ctx, closedSQLiteExecutor(t), "0000000000000001")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &PlayerLockedSongRepository{}

			// When
			err := tt.act(context.Background(), repo)

			// Then
			require.Error(t, err)
			assert.ErrorIs(t, err, domainrepo.ErrRepositoryOperationFailed)
			assert.NotErrorIs(t, err, sql.ErrConnDone)
		})
	}
}

func TestResolveSongIDByDisplayIDReturnsNilWhenNoRows(t *testing.T) {
	// Given
	repo := &PlayerLockedSongRepository{}
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	_, err = db.Exec(`CREATE TABLE songs (id INTEGER PRIMARY KEY, display_id TEXT NOT NULL)`)
	require.NoError(t, err)

	// When
	id, err := repo.ResolveSongIDByDisplayID(context.Background(), db, "0000000000000001")

	// Then
	require.NoError(t, err)
	assert.Nil(t, id)
}

func closedSQLiteExecutor(t *testing.T) domainrepo.Executor {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Close())
	return db
}
