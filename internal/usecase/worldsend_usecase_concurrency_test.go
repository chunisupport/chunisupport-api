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
}

// worldsendMutationRepository は楽曲行の排他ロックを模し、
// ロック中に読み取った全列の書き戻しと、編集による列更新が競合しても両方残ることを検証します。
type worldsendMutationRepository struct {
	row       sync.Mutex
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
	return fn(tx)
}

func (r *worldsendMutationRepository) lock(exec repository.Executor) error {
	tx, ok := exec.(*worldsendMutationTx)
	if !ok {
		return errors.New("transaction required")
	}
	r.attempted <- struct{}{}
	r.row.Lock()
	tx.locked = true
	r.first.Do(func() {
		close(r.acquired)
		<-r.release
	})
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
	song := r.song
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
	r.song = *song
	return nil
}

func (r *worldsendMutationRepository) UpdateSongs(_ context.Context, exec repository.Executor, updates []*repository.WorldsendUpdate) error {
	// 実更新は is_deleted を書かず、行ロックの解放を待ってから対象列だけを更新する。
	if err := r.lock(exec); err != nil {
		return err
	}
	if len(updates) != 1 || updates[0] == nil || updates[0].Song == nil {
		return errors.New("unexpected update")
	}
	r.song.Title = updates[0].Song.Title
	return nil
}

func (r *worldsendMutationRepository) CreateSong(context.Context, repository.Executor, *entity.Song, *entity.WorldsendChart) (*entity.WorldsendSongWithChart, error) {
	return nil, errors.New("not implemented")
}

func TestWorldsendUsecase_ConcurrentMutationsPreserveIndependentFields(t *testing.T) {
	tests := []struct {
		name string
		// Given
		deleted   bool
		editFirst bool
	}{
		{name: "編集が先に行ロックを取得しても削除状態を保持する", deleted: true, editFirst: true},
		{name: "削除が先に行ロックを取得しても編集内容を保持する", deleted: true, editFirst: false},
		{name: "編集が先に行ロックを取得しても復元状態を保持する", deleted: false, editFirst: true},
		{name: "復元が先に行ロックを取得しても編集内容を保持する", deleted: false, editFirst: false},
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
					IsDeleted:   !tt.deleted,
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
			first, second := edit, state
			if !tt.editFirst {
				first, second = state, edit
			}

			// When
			done := make(chan error, 2)
			go func() { done <- first() }()
			waitWorldsendSignal(t, repo.acquired)
			waitWorldsendSignal(t, repo.attempted)
			go func() { done <- second() }()
			waitWorldsendSignal(t, repo.attempted)
			unblock()

			// Then
			for range 2 {
				select {
				case err := <-done:
					require.NoError(t, err)
				case <-time.After(5 * time.Second):
					require.FailNow(t, "WORLD'S END楽曲の更新が完了しません")
				}
			}
			assert.Equal(t, "変更後", repo.song.Title)
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
