package repository

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/songcatalog"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingSongRepository は全件取得の回数を数え、再構築の回数を観測します。
// failNext が true の間は、DB障害を再現するため全件取得を失敗させます。
type countingSongRepository struct {
	domainrepo.SongRepository
	findAllCount atomic.Int32
	failNext     atomic.Bool
}

func (r *countingSongRepository) FindAllExcludingWorldsend(ctx context.Context, exec domainrepo.Executor, includeDeleted bool) ([]*entity.Song, error) {
	r.findAllCount.Add(1)
	if r.failNext.Load() {
		return nil, errors.New("db unavailable")
	}
	return r.SongRepository.FindAllExcludingWorldsend(ctx, exec, includeDeleted)
}

func setupSongCatalogMySQL(t *testing.T) (*sqlx.DB, *SongCatalogProvider, *countingSongRepository) {
	t.Helper()
	db := mysqltest.NewDatabase(t, "")
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted, updated_at)
		VALUES
			(1, 'song000000000001', 'Song 1', 'Artist', 1, 'IDX001', 0, 0, '2026-11-01 00:00:00'),
			(2, 'song000000000002', 'Song 2', 'Artist', 1, 'IDX002', 0, 1, '2026-11-02 00:00:00'),
			(3, 'we00000000000003', 'WE 3', 'Artist', 1, 'IDX003', 1, 0, '2026-11-03 00:00:00')
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, updated_at)
		VALUES
			(1, 3, 12.0, 0, 800, '2026-11-01 00:00:00'),
			(1, 4, 13.5, 0, 1000, '2026-11-04 00:00:00'),
			(2, 4, 14.0, 0, 1200, '2026-11-01 00:00:00')
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO worldsend_charts (song_id, level_star, attribute, notes, updated_at)
		VALUES (3, 3, '狂', 900, '2026-11-01 00:00:00')
	`)
	require.NoError(t, err)

	songRepo := &countingSongRepository{SongRepository: NewSongRepository(db)}
	provider := newSongCatalogProvider(db, songRepo, NewWorldsendChartRepository(db))
	return db, provider, songRepo
}

func songDisplayIDs(songs []*entity.Song) []string {
	ids := make([]string, 0, len(songs))
	for _, song := range songs {
		ids = append(ids, song.DisplayID)
	}
	return ids
}

func TestSongCatalogProviderMySQL_初回参照で通常楽曲とWORLDSENDを同じ世代として構築する(t *testing.T) {
	// Given
	ctx := context.Background()
	_, provider, _ := setupSongCatalogMySQL(t)

	// When
	catalog, err := provider.Snapshot(ctx)

	// Then
	require.NoError(t, err)
	assert.Equal(t, []string{"song000000000001", "song000000000002"}, songDisplayIDs(catalog.Songs(true)))
	require.Equal(t, []string{"song000000000001"}, songDisplayIDs(catalog.Songs(false)))
	activeSongs := catalog.Songs(false)
	require.Len(t, activeSongs[0].Charts, 2)
	assert.InDelta(t, 13.5, activeSongs[0].MaxChartConst, 0.001)
	worldsendSongs := catalog.WorldsendSongs(false)
	require.Len(t, worldsendSongs, 1)
	assert.Equal(t, "we00000000000003", worldsendSongs[0].Song.DisplayID)
	require.NotNil(t, catalog.UpdatedAt())
	assert.Equal(t, time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC), catalog.UpdatedAt().UTC())
}

func TestSongCatalogProviderMySQL_APIの保存後に再構築すると追加_編集_削除_復活が反映される(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, ctx context.Context, tx domainrepo.Executor)
		assert func(t *testing.T, catalog *songcatalog.Catalog)
	}{
		{
			name: "追加",
			mutate: func(t *testing.T, ctx context.Context, tx domainrepo.Executor) {
				song := entity.NewSong()
				song.DisplayID = "song000000000009"
				song.OfficialIdx = "IDX009"
				song.Title = "New"
				song.Artist = "Artist"
				song.GenreID = new(1)
				_, err := NewSongRepository(nil).Create(ctx, tx, song)
				require.NoError(t, err)
			},
			assert: func(t *testing.T, catalog *songcatalog.Catalog) {
				assert.Contains(t, songDisplayIDs(catalog.Songs(false)), "song000000000009")
			},
		},
		{
			name: "編集",
			mutate: func(t *testing.T, ctx context.Context, tx domainrepo.Executor) {
				repo := NewSongRepository(nil)
				song, err := repo.FindByDisplayIDForChange(ctx, tx, "song000000000001")
				require.NoError(t, err)
				song.Title = "Edited"
				require.NoError(t, repo.Save(ctx, tx, song))
			},
			assert: func(t *testing.T, catalog *songcatalog.Catalog) {
				assert.Equal(t, "Edited", catalog.Songs(false)[0].Title)
			},
		},
		{
			name: "削除",
			mutate: func(t *testing.T, ctx context.Context, tx domainrepo.Executor) {
				repo := NewWorldsendChartRepository(nil)
				song, err := repo.FindByDisplayIDForChange(ctx, tx, "we00000000000003")
				require.NoError(t, err)
				song.Song.Delete()
				require.NoError(t, repo.SaveSong(ctx, tx, song.Song))
			},
			assert: func(t *testing.T, catalog *songcatalog.Catalog) {
				assert.Empty(t, catalog.WorldsendSongs(false))
				assert.Len(t, catalog.WorldsendSongs(true), 1)
			},
		},
		{
			name: "復活",
			mutate: func(t *testing.T, ctx context.Context, tx domainrepo.Executor) {
				repo := NewSongRepository(nil)
				song, err := repo.FindByDisplayIDForChange(ctx, tx, "song000000000002")
				require.NoError(t, err)
				song.Restore()
				require.NoError(t, repo.Save(ctx, tx, song))
			},
			assert: func(t *testing.T, catalog *songcatalog.Catalog) {
				assert.Equal(t, []string{"song000000000001", "song000000000002"}, songDisplayIDs(catalog.Songs(false)))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			ctx := context.Background()
			db, provider, _ := setupSongCatalogMySQL(t)
			before, err := provider.Snapshot(ctx)
			require.NoError(t, err)
			require.NoError(t, transaction.NewTransactionManager(db).Transactional(ctx, func(tx domainrepo.Executor) error {
				tt.mutate(t, ctx, tx)
				return nil
			}))

			// When
			require.NoError(t, provider.Refresh(ctx))

			// Then
			after, err := provider.Snapshot(ctx)
			require.NoError(t, err)
			assert.Greater(t, after.Revision(), before.Revision())
			tt.assert(t, after)
		})
	}
}

func TestSongCatalogProviderMySQL_世代が変わらなければ再構築しない(t *testing.T) {
	// Given
	ctx := context.Background()
	_, provider, songRepo := setupSongCatalogMySQL(t)
	before, err := provider.Snapshot(ctx)
	require.NoError(t, err)

	// When
	require.NoError(t, provider.Refresh(ctx))

	// Then
	after, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	assert.Same(t, before, after)
	assert.Equal(t, int32(1), songRepo.findAllCount.Load())
}

func TestSongCatalogProviderMySQL_同時に再構築を要求しても1回だけ構築する(t *testing.T) {
	// Given
	ctx := context.Background()
	db, provider, songRepo := setupSongCatalogMySQL(t)
	_, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	// 別プロセスのsong-batchが更新した状況を、SQLの直接更新と世代の更新で再現します。
	_, err = db.Exec(`UPDATE songs SET title = 'Batch' WHERE id = 1`)
	require.NoError(t, err)
	require.NoError(t, TouchSongCatalogRevision(ctx, db))

	// When
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			assert.NoError(t, provider.Refresh(ctx))
			catalog, err := provider.Snapshot(ctx)
			if assert.NoError(t, err) {
				assert.NotEmpty(t, catalog.Songs(true))
			}
		})
	}
	wg.Wait()

	// Then
	assert.Equal(t, int32(2), songRepo.findAllCount.Load(), "初回構築と変更後の1回だけ")
	catalog, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Batch", catalog.Songs(false)[0].Title)
}

func TestSongCatalogProviderMySQL_世代を進めていない変更は反映せずコミット前の変更も公開しない(t *testing.T) {
	// Given
	ctx := context.Background()
	db, provider, _ := setupSongCatalogMySQL(t)
	_, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	_, err = tx.Exec(`UPDATE songs SET title = 'Uncommitted' WHERE id = 1`)
	require.NoError(t, err)

	// When
	require.NoError(t, provider.Refresh(ctx))

	// Then
	catalog, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Song 1", catalog.Songs(false)[0].Title)
}

func TestSongCatalogProviderMySQL_更新時刻は同じ世代の楽曲データと同時に差し替わる(t *testing.T) {
	// Given
	ctx := context.Background()
	db, provider, _ := setupSongCatalogMySQL(t)
	before, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE charts SET const = 14.5, updated_at = '2026-11-10 00:00:00' WHERE song_id = 1 AND difficulty_id = 4`)
	require.NoError(t, err)
	require.NoError(t, TouchSongCatalogRevision(ctx, db))

	// When
	require.NoError(t, provider.Refresh(ctx))

	// Then
	after, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	// 旧世代は旧データと旧時刻の組を保ち、新世代は新データと新時刻の組で公開されます。
	assert.InDelta(t, 13.5, before.Songs(false)[0].MaxChartConst, 0.001)
	assert.Equal(t, time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC), before.UpdatedAt().UTC())
	assert.InDelta(t, 14.5, after.Songs(false)[0].MaxChartConst, 0.001)
	assert.Equal(t, time.Date(2026, time.November, 10, 0, 0, 0, 0, time.UTC), after.UpdatedAt().UTC())
}

func TestSongCatalogProviderMySQL_再構築に失敗しても公開中の世代を維持する(t *testing.T) {
	// Given
	ctx := context.Background()
	db, provider, _ := setupSongCatalogMySQL(t)
	before, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	require.NoError(t, TouchSongCatalogRevision(ctx, db))
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	// When
	err = provider.Refresh(canceledCtx)

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	after, err := provider.Snapshot(ctx)
	require.NoError(t, err)
	assert.Same(t, before, after)
}

func TestSongCatalogProviderMySQL_初回構築に失敗した場合はエラーを返し次の参照で再試行する(t *testing.T) {
	// Given
	ctx := context.Background()
	_, provider, songRepo := setupSongCatalogMySQL(t)
	songRepo.failNext.Store(true)

	// When
	_, firstErr := provider.Snapshot(ctx)
	songRepo.failNext.Store(false)
	catalog, secondErr := provider.Snapshot(ctx)

	// Then
	require.Error(t, firstErr)
	assert.ErrorIs(t, firstErr, domainrepo.ErrRepositoryOperationFailed)
	require.NoError(t, secondErr)
	assert.NotEmpty(t, catalog.Songs(true))
	assert.Equal(t, int32(2), songRepo.findAllCount.Load())
}

func TestSongCatalogProviderMySQL_未構築時の同時参照は1回の構築を共有する(t *testing.T) {
	// Given
	ctx := context.Background()
	_, provider, songRepo := setupSongCatalogMySQL(t)

	// When
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			catalog, err := provider.Snapshot(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, catalog)
		})
	}
	wg.Wait()

	// Then
	assert.Equal(t, int32(1), songRepo.findAllCount.Load())
}

func TestSongCatalogProviderMySQL_未構築時の待機は呼び出し元の切断で戻る(t *testing.T) {
	// Given
	_, provider, _ := setupSongCatalogMySQL(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// When
	catalog, err := provider.Snapshot(ctx)

	// Then
	// 共有の構築とどちらが先に終わるかは実行順に依存するため、エラーの場合だけ原因を確認します。
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
		return
	}
	assert.NotNil(t, catalog)
}
