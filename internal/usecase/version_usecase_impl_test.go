package usecase

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
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
	// rangeFrom/rangeTo は削除時の利用有無判定に使った検索範囲の記録です。
	rangeFrom time.Time
	rangeTo   *time.Time
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

// ExistsSongInRange は削除可否判定の検索範囲を記録します。
// 下限へ対象版のReleasedAt、上限へnilを渡す契約を固定するため引数を捨てません。
func (s *versionRepositoryStub) ExistsSongInRange(_ context.Context, _ repository.Executor, from time.Time, to *time.Time) (bool, error) {
	s.rangeFrom = from
	s.rangeTo = to
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

	created, err := uc.Create(context.Background(), " CHUNITHM VERSE ", " VRS ", dateForVersionTest(2025, 1, 1))

	require.NoError(t, err)
	assert.Equal(t, 3, created.ID)
	assert.Equal(t, "CHUNITHM VERSE", created.Name)
	assert.Equal(t, "VRS", created.ShortName)
	assert.Equal(t, 1, reloader.calls)
}

func TestVersionUsecase_Create_同日を拒否する(t *testing.T) {
	repo := &versionRepositoryStub{versions: []*entity.Version{{ReleasedAt: dateForVersionTest(2025, 1, 1)}}}
	reloader := &versionCacheReloaderStub{}
	uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)

	_, err := uc.Create(context.Background(), "CHUNITHM VERSE", "VRS", dateForVersionTest(2025, 1, 1))

	assert.ErrorIs(t, err, ErrInvalidVersionInput)
	assert.Zero(t, reloader.calls)
}

func TestVersionUsecase_Create_同日でなければ受理する(t *testing.T) {
	tests := []struct {
		name string
		// Given: 既存バージョンのリリース日
		existing time.Time
		// When: 登録するリリース日
		releasedAt time.Time
	}{
		{
			name:       "年と月が同じで日だけ違う日付は受理する",
			existing:   dateForVersionTest(2025, 1, 1),
			releasedAt: dateForVersionTest(2025, 1, 15),
		},
		{
			name:       "年と日が同じで月だけ違う日付は受理する",
			existing:   dateForVersionTest(2025, 1, 15),
			releasedAt: dateForVersionTest(2025, 2, 15),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &versionRepositoryStub{versions: []*entity.Version{{ID: 1, ReleasedAt: tt.existing}}}
			reloader := &versionCacheReloaderStub{}
			uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)

			// When
			created, err := uc.Create(context.Background(), "CHUNITHM VERSE", "VRS", tt.releasedAt)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.releasedAt, created.ReleasedAt)
			assert.Equal(t, 1, reloader.calls)
		})
	}
}

func TestVersionUsecase_Create_不正な超ショート名を拒否する(t *testing.T) {
	// Given
	repo := &versionRepositoryStub{}
	uc := NewVersionUsecase(repo, &versionCacheReloaderStub{}, transactionManagerStub{}, nil)

	// When
	_, err := uc.Create(context.Background(), "CHUNITHM VERSE", "", dateForVersionTest(2025, 1, 1))

	// Then
	assert.ErrorIs(t, err, ErrInvalidVersionInput)
	assert.ErrorIs(t, err, entity.ErrInvalidVersion)
	assert.Nil(t, repo.created)
}

func TestVersionUsecase_Create_名前重複を返す(t *testing.T) {
	repo := &versionRepositoryStub{writeErr: repository.ErrVersionConflict}
	uc := NewVersionUsecase(repo, &versionCacheReloaderStub{}, transactionManagerStub{}, nil)

	_, err := uc.Create(context.Background(), "CHUNITHM VERSE", "VRS", dateForVersionTest(2025, 1, 1))

	assert.ErrorIs(t, err, repository.ErrVersionConflict)
}

func TestVersionUsecase_Rename_稼働日は変えない(t *testing.T) {
	releasedAt := dateForVersionTest(2025, 1, 1)
	repo := &versionRepositoryStub{versions: []*entity.Version{{ID: 1, Name: "CHUNITHM VERS", ReleasedAt: releasedAt}}}
	reloader := &versionCacheReloaderStub{}
	uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)

	updated, err := uc.Rename(context.Background(), 1, "CHUNITHM VERSE", "VRS")

	require.NoError(t, err)
	assert.Equal(t, releasedAt, updated.ReleasedAt)
	assert.Equal(t, "CHUNITHM VERSE", repo.saved.Name)
	assert.Equal(t, "VRS", repo.saved.ShortName)
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
				assert.Zero(t, repo.deleted)
				if tt.inUse {
					// 利用有無判定まで進んだ場合だけ検索範囲を固定する。
					assert.Equal(t, dateForVersionTest(2025, 1, 1), repo.rangeFrom)
					assert.Nil(t, repo.rangeTo)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantDelete, repo.deleted)
				assert.Equal(t, 1, reloader.calls)
				assert.Equal(t, dateForVersionTest(2025, 1, 1), repo.rangeFrom)
				assert.Nil(t, repo.rangeTo)
			}
		})
	}
}

func TestVersionUsecase_コミット後の再読込失敗を操作失敗として返さない(t *testing.T) {
	var logBuffer bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuffer, nil)))
	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	tests := []struct {
		name      string
		operation func(VersionUsecase) error
	}{
		{
			name: "作成",
			operation: func(uc VersionUsecase) error {
				_, err := uc.Create(context.Background(), "CHUNITHM VERSE", "VRS", dateForVersionTest(2026, 1, 1))
				return err
			},
		},
		{
			name: "名称変更",
			operation: func(uc VersionUsecase) error {
				_, err := uc.Rename(context.Background(), 2, "CHUNITHM VERSE PLUS", "VRS")
				return err
			},
		},
		{
			name: "削除",
			operation: func(uc VersionUsecase) error {
				return uc.Delete(context.Background(), 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &versionRepositoryStub{versions: []*entity.Version{
				{ID: 1, Name: "CHUNITHM LUMINOUS", ReleasedAt: dateForVersionTest(2024, 1, 1)},
				{ID: 2, Name: "CHUNITHM VERSE", ReleasedAt: dateForVersionTest(2025, 1, 1)},
			}}
			reloader := &versionCacheReloaderStub{err: errors.New("reload failed")}
			uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)

			// When
			err := tt.operation(uc)

			// Then
			assert.NoError(t, err)
			assert.Equal(t, 1, reloader.calls)
		})
	}
	assert.Equal(t, len(tests), strings.Count(logBuffer.String(), "バージョンキャッシュの再読込に失敗しました"))
}

func TestVersionUsecase_再読込は要求キャンセルから切り離す(t *testing.T) {
	repo := &versionRepositoryStub{versions: []*entity.Version{}}
	reloader := &versionCacheReloaderStub{}
	uc := NewVersionUsecase(repo, reloader, transactionManagerStub{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := uc.Create(ctx, "CHUNITHM VERSE", "VRS", dateForVersionTest(2025, 1, 1))

	require.NoError(t, err)
	assert.NoError(t, reloader.ctxErr)
}

func dateForVersionTest(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestVersionUsecase_不正な名前はドメインエラーを保持する(t *testing.T) {
	tests := []struct {
		name string
		// When: 不正な名前で実行する操作
		run func(uc VersionUsecase) error
	}{
		{
			name: "作成",
			run: func(uc VersionUsecase) error {
				_, err := uc.Create(context.Background(), "VERSE", "VRS", dateForVersionTest(2025, 1, 1))
				return err
			},
		},
		{
			name: "名称変更",
			run: func(uc VersionUsecase) error {
				_, err := uc.Rename(context.Background(), 1, "VERSE", "VRS")
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &versionRepositoryStub{versions: []*entity.Version{{ID: 1, Name: "CHUNITHM VERS", ReleasedAt: dateForVersionTest(2024, 1, 1)}}}
			uc := NewVersionUsecase(repo, &versionCacheReloaderStub{}, transactionManagerStub{}, nil)

			// When
			err := tt.run(uc)

			// Then
			assert.ErrorIs(t, err, ErrInvalidVersionInput)
			assert.ErrorIs(t, err, entity.ErrInvalidVersion)
		})
	}
}
