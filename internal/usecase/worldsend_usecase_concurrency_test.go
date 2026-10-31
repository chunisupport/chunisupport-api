package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// worldsendMutationTx は行ロックを取得したトランザクションを表します。
type worldsendMutationTx struct {
	repository.Executor
	locked bool
	song   entity.Song
}

// worldsendMutationRepository は楽曲行の排他ロックを模し、
// コミット済み状態の検索と、行ロック後の更新を本番と同じ順序で行います。
type worldsendMutationRepository struct {
	row       sync.Mutex
	committed sync.RWMutex
	first     sync.Once
	attempted chan struct{}
	acquired  chan struct{}
	release   chan struct{}
	song      entity.Song
}

func (r *worldsendMutationRepository) Transactional(_ context.Context, fn func(repository.Executor) error) error {
	tx := &worldsendMutationTx{}
	defer func() {
		if tx.locked {
			r.row.Unlock()
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if tx.locked {
		// 更新後・コミット前で止め、別トランザクションに未確定の値が見えないことを確認する。
		r.first.Do(func() {
			close(r.acquired)
			<-r.release
		})
		r.committed.Lock()
		r.song = tx.song
		r.committed.Unlock()
	}
	return nil
}

func (r *worldsendMutationRepository) lock(exec repository.Executor) error {
	tx, ok := exec.(*worldsendMutationTx)
	if !ok {
		return errors.New("transaction required")
	}
	r.attempted <- struct{}{}
	r.row.Lock()
	tx.locked = true
	r.committed.RLock()
	tx.song = r.song
	r.committed.RUnlock()
	return nil
}

func (r *worldsendMutationRepository) FindAll(context.Context, repository.Executor, bool) ([]*entity.WorldsendSongWithChart, error) {
	return nil, errors.New("not implemented")
}

func (r *worldsendMutationRepository) FindByDisplayID(context.Context, repository.Executor, string) (*entity.WorldsendSongWithChart, error) {
	return nil, errors.New("not implemented")
}

func (r *worldsendMutationRepository) FindByDisplayIDForChange(_ context.Context, exec repository.Executor, _ string) (*entity.WorldsendSongWithChart, error) {
	if err := r.lock(exec); err != nil {
		return nil, err
	}
	song := exec.(*worldsendMutationTx).song
	return &entity.WorldsendSongWithChart{
		Song:  &song,
		Chart: &entity.WorldsendChart{ID: 1, SongID: song.ID},
	}, nil
}

func (r *worldsendMutationRepository) SaveSong(_ context.Context, exec repository.Executor, song *entity.Song) error {
	tx, ok := exec.(*worldsendMutationTx)
	if !ok || !tx.locked {
		return errors.New("locked transaction required")
	}
	tx.song = *song
	return nil
}

func (r *worldsendMutationRepository) UpdateSongs(_ context.Context, exec repository.Executor, updates []*repository.WorldsendUpdate) error {
	// 本番と同じく、行ロックを待つ前にコミット済みの有効な楽曲を検索する。
	// 未コミットの復活は見えないため、削除済みなら UPDATE に進まない。
	r.committed.RLock()
	deleted := r.song.IsDeleted
	r.committed.RUnlock()
	if deleted {
		return repository.ErrSongNotFound
	}
	// 実更新は is_deleted を書かず、行ロックの解放を待ってから対象列だけを更新する。
	if err := r.lock(exec); err != nil {
		return err
	}
	if len(updates) != 1 || updates[0] == nil || updates[0].Song == nil {
		return errors.New("unexpected update")
	}
	tx := exec.(*worldsendMutationTx)
	tx.song.Title = updates[0].Song.Title
	// 譜面更新を省略した本番処理は、UPDATE 後に有効な対象の存在を再確認する。
	// 自トランザクションの更新行では、直前にコミットされた削除状態も見える。
	if updates[0].Chart == nil && tx.song.IsDeleted {
		return repository.ErrSongNotFound
	}
	return nil
}

func (r *worldsendMutationRepository) CreateSong(context.Context, repository.Executor, *entity.Song, *entity.WorldsendChart) (*entity.WorldsendSongWithChart, error) {
	return nil, errors.New("not implemented")
}

func TestWorldsendUsecase_ConcurrentMutationsPreserveIndependentFields(t *testing.T) {
	tests := []struct {
		name string
		// Given
		deleted          bool
		editFirst        bool
		initiallyDeleted bool
		editErr          error
	}{
		{name: "編集が先に行ロックを取得しても削除状態を保持する", deleted: true, editFirst: true},
		{name: "削除が先なら編集はロック待機後に未検出となり変更を戻す", deleted: true, editFirst: false, editErr: repository.ErrSongNotFound},
		{name: "有効な楽曲の編集が先に行ロックを取得しても復元状態を保持する", deleted: false, editFirst: true},
		{name: "未コミットの復元中は編集がロックを待たずに未検出となる", deleted: false, editFirst: false, initiallyDeleted: true, editErr: repository.ErrSongNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			const displayID = "0123456789abcdef"
			repo := &worldsendMutationRepository{
				attempted: make(chan struct{}, 2),
				acquired:  make(chan struct{}),
				release:   make(chan struct{}),
				song: entity.Song{
					ID:          1,
					DisplayID:   displayID,
					Title:       "変更前",
					Artist:      "artist",
					IsWorldsend: true,
					IsDeleted:   tt.initiallyDeleted,
				},
			}
			var release sync.Once
			unblock := func() { release.Do(func() { close(repo.release) }) }
			defer unblock()
			uc := NewWorldsendUsecase(repo, repo, nil)
			masters := &domainmasterdata.SongMasters{}
			edit := func() error {
				return uc.UpdateWorldsendSongs(context.Background(), []*UpdateWorldsendSongInput{{
					DisplayID: displayID,
					Title:     "変更後",
					Artist:    "artist",
				}}, masters)
			}
			state := func() error {
				if tt.deleted {
					return uc.DeleteWorldsendSong(context.Background(), displayID)
				}
				return uc.RestoreWorldsendSong(context.Background(), displayID)
			}
			editDone := make(chan error, 1)
			stateDone := make(chan error, 1)
			first := func() { editDone <- edit() }
			second := func() { stateDone <- state() }
			if !tt.editFirst {
				first, second = second, first
			}

			// When
			go first()
			waitWorldsendSignal(t, repo.acquired)
			waitWorldsendSignal(t, repo.attempted)
			go second()
			if tt.initiallyDeleted {
				// 復活側のロック解放前に編集が終了することも確認する。
				require.ErrorIs(t, waitWorldsendResult(t, editDone), tt.editErr)
			} else {
				waitWorldsendSignal(t, repo.attempted)
			}
			unblock()

			// Then
			require.NoError(t, waitWorldsendResult(t, stateDone))
			if !tt.initiallyDeleted {
				err := waitWorldsendResult(t, editDone)
				if tt.editErr != nil {
					require.ErrorIs(t, err, tt.editErr)
				} else {
					require.NoError(t, err)
				}
			}
			expectedTitle := "変更後"
			if tt.editErr != nil {
				expectedTitle = "変更前"
			}
			assert.Equal(t, expectedTitle, repo.song.Title)
			assert.Equal(t, tt.deleted, repo.song.IsDeleted)
		})
	}
}

func waitWorldsendSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "WORLD'S END楽曲ロックの同期が完了しません")
	}
}

func waitWorldsendResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		require.FailNow(t, "WORLD'S END楽曲の更新が完了しません")
		return nil
	}
}
