package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type versionRepositoryStub struct {
	versions []*entity.Version
	created  *entity.Version
	saved    *entity.Version
	deleted  int
	inUse    bool
	findErr  error
	writeErr error
}

func (s *versionRepositoryStub) FindAll(context.Context, repository.Executor) ([]*entity.Version, error) {
	return s.versions, s.findErr
}
func (s *versionRepositoryStub) FindByID(context.Context, repository.Executor, int) (*entity.Version, error) {
	return nil, repository.ErrVersionNotFound
}
func (s *versionRepositoryStub) FindByIDForUpdate(_ context.Context, _ repository.Executor, id int) (*entity.Version, error) {
	for _, version := range s.versions {
		if version.ID == id {
			copy := *version
			return &copy, nil
		}
	}
	return nil, repository.ErrVersionNotFound
}
func (s *versionRepositoryStub) FindByName(context.Context, repository.Executor, string) (*entity.Version, error) {
	return nil, repository.ErrVersionNotFound
}
func (s *versionRepositoryStub) FindLatest(context.Context, repository.Executor) (*entity.Version, error) {
	return s.versions[len(s.versions)-1], nil
}
func (s *versionRepositoryStub) ExistsSongInRange(context.Context, repository.Executor, time.Time, *time.Time) (bool, error) {
	return s.inUse, nil
}
func (s *versionRepositoryStub) Create(_ context.Context, _ repository.Executor, version *entity.Version) (*entity.Version, error) {
	if s.writeErr != nil {
		return nil, s.writeErr
	}
	copy := *version
	copy.ID = 3
	s.created = &copy
	return &copy, nil
}
func (s *versionRepositoryStub) Save(_ context.Context, _ repository.Executor, version *entity.Version) error {
	copy := *version
	s.saved = &copy
	return s.writeErr
}
func (s *versionRepositoryStub) Delete(_ context.Context, _ repository.Executor, id int) error {
	s.deleted = id
	return s.writeErr
}

type transactionManagerStub struct{}

func (transactionManagerStub) Transactional(ctx context.Context, fn func(repository.Executor) error) error {
	return fn(nil)
}

type versionCacheReloaderStub struct {
	calls  int
	err    error
	ctxErr error
}

func (s *versionCacheReloaderStub) ReloadVersions(ctx context.Context) error {
	s.calls++
	s.ctxErr = ctx.Err()
	return s.err
}

func TestVersionUsecase_Create(t *testing.T) {
	repo := &versionRepositoryStub{versions: []*entity.Version{{ID: 1, ReleasedAt: dateForVersionTest(2024, 1, 1)}}}
	reloader := &versionCacheReloaderStub{}
	uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)

	created, err := uc.Create(context.Background(), " CHUNITHM VERSE ", dateForVersionTest(2025, 1, 1))

	require.NoError(t, err)
	assert.Equal(t, 3, created.ID)
	assert.Equal(t, "CHUNITHM VERSE", created.Name)
	assert.Equal(t, 1, reloader.calls)
}

func TestVersionUsecase_Create_同日を拒否する(t *testing.T) {
	repo := &versionRepositoryStub{versions: []*entity.Version{{ReleasedAt: dateForVersionTest(2025, 1, 1)}}}
	reloader := &versionCacheReloaderStub{}
	uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)

	_, err := uc.Create(context.Background(), "CHUNITHM VERSE", dateForVersionTest(2025, 1, 1))

	assert.ErrorIs(t, err, ErrInvalidVersionInput)
	assert.Zero(t, reloader.calls)
}

func TestVersionUsecase_Create_不正な名前を拒否する(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "空", input: ""},
		{name: "接頭辞なし", input: "VERSE"},
		{name: "長さ超過", input: "CHUNITHM 123456789012345678901234567890123456789012"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewVersionUsecase(&versionRepositoryStub{}, &versionCacheReloaderStub{}, transactionManagerStub{}, nil)

			_, err := uc.Create(context.Background(), tt.input, dateForVersionTest(2025, 1, 1))

			assert.ErrorIs(t, err, ErrInvalidVersionInput)
		})
	}
}

func TestVersionUsecase_Create_名前重複を返す(t *testing.T) {
	repo := &versionRepositoryStub{writeErr: repository.ErrVersionConflict}
	uc := NewVersionUsecase(repo, &versionCacheReloaderStub{}, transactionManagerStub{}, nil)

	_, err := uc.Create(context.Background(), "CHUNITHM VERSE", dateForVersionTest(2025, 1, 1))

	assert.ErrorIs(t, err, repository.ErrVersionConflict)
}

func TestVersionUsecase_Rename_稼働日は変えない(t *testing.T) {
	releasedAt := dateForVersionTest(2025, 1, 1)
	repo := &versionRepositoryStub{versions: []*entity.Version{{ID: 1, Name: "CHUNITHM VERS", ReleasedAt: releasedAt}}}
	reloader := &versionCacheReloaderStub{}
	uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)

	updated, err := uc.Rename(context.Background(), 1, "CHUNITHM VERSE")

	require.NoError(t, err)
	assert.Equal(t, releasedAt, updated.ReleasedAt)
	assert.Equal(t, "CHUNITHM VERSE", repo.saved.Name)
	assert.Equal(t, 1, reloader.calls)
}

func TestVersionUsecase_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         int
		inUse      bool
		wantErr    error
		wantDelete int
	}{
		{name: "最新かつ曲なしは削除できる", id: 2, wantDelete: 2},
		{name: "中間版は削除できない", id: 1, wantErr: ErrVersionNotLatest},
		{name: "曲あり最新版は削除できない", id: 2, inUse: true, wantErr: ErrVersionInUse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &versionRepositoryStub{versions: []*entity.Version{
				{ID: 1, ReleasedAt: dateForVersionTest(2024, 1, 1)},
				{ID: 2, ReleasedAt: dateForVersionTest(2025, 1, 1)},
			}, inUse: tt.inUse}
			reloader := &versionCacheReloaderStub{}
			uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)

			err := uc.Delete(context.Background(), tt.id)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Zero(t, reloader.calls)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantDelete, repo.deleted)
				assert.Equal(t, 1, reloader.calls)
			}
		})
	}
}

func TestVersionUsecase_コミット後の再読込失敗を返す(t *testing.T) {
	reloadErr := errors.New("reload failed")
	repo := &versionRepositoryStub{versions: []*entity.Version{}}
	uc := NewVersionUsecase(repo, &versionCacheReloaderStub{err: reloadErr}, transactionManagerStub{}, nil)

	_, err := uc.Create(context.Background(), "CHUNITHM VERSE", dateForVersionTest(2025, 1, 1))

	assert.ErrorIs(t, err, reloadErr)
}

func TestVersionUsecase_再読込は要求キャンセルから切り離す(t *testing.T) {
	repo := &versionRepositoryStub{versions: []*entity.Version{}}
	reloader := &versionCacheReloaderStub{}
	uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := uc.Create(ctx, "CHUNITHM VERSE", dateForVersionTest(2025, 1, 1))

	require.NoError(t, err)
	assert.NoError(t, reloader.ctxErr)
}

func dateForVersionTest(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
