package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRecordFilterRepository struct {
	filters map[string]*entity.RecordFilter
	// calls は件数確認と保存が同じトランザクションで行われたことを検証するため、呼び出し順と実行先を記録します。
	calls []recordFilterRepositoryCall
	// afterFind は取得直後に別リクエストの変更が割り込む状況を再現します。
	afterFind func()
}

type recordFilterRepositoryCall struct {
	name string
	exec repository.Executor
}

func newStubRecordFilterRepository() *stubRecordFilterRepository {
	return &stubRecordFilterRepository{filters: map[string]*entity.RecordFilter{}}
}

func (s *stubRecordFilterRepository) record(name string, exec repository.Executor) {
	s.calls = append(s.calls, recordFilterRepositoryCall{name: name, exec: exec})
}

func (s *stubRecordFilterRepository) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.RecordFilter, error) {
	filters := make([]*entity.RecordFilter, 0, len(s.filters))
	for _, filter := range s.filters {
		if filter.UserID() == userID {
			filters = append(filters, filter)
		}
	}
	return filters, nil
}

func (s *stubRecordFilterRepository) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id []byte, userID int) (*entity.RecordFilter, error) {
	filter, ok := s.filters[string(id)]
	if !ok || filter.UserID() != userID {
		return nil, repository.ErrRecordFilterNotFound
	}
	if s.afterFind != nil {
		s.afterFind()
	}
	return filter, nil
}

func (s *stubRecordFilterRepository) Create(ctx context.Context, exec repository.Executor, filter *entity.RecordFilter) error {
	s.record("Create", exec)
	return s.store(filter)
}

func (s *stubRecordFilterRepository) Update(ctx context.Context, exec repository.Executor, filter *entity.RecordFilter) error {
	existing, ok := s.filters[string(filter.ID())]
	if !ok || existing.UserID() != filter.UserID() {
		return repository.ErrRecordFilterNotFound
	}
	return s.store(filter)
}

func (s *stubRecordFilterRepository) store(filter *entity.RecordFilter) error {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	createdAt := filter.CreatedAt()
	if createdAt.IsZero() {
		createdAt = now
	}
	restored, err := entity.RestoreRecordFilter(filter.ID(), filter.UserID(), filter.Name(), filter.FilterValueGzip(), filter.IsWorldsend(), createdAt, now)
	if err != nil {
		return err
	}
	s.filters[string(filter.ID())] = restored
	return nil
}

func (s *stubRecordFilterRepository) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id []byte, userID int) error {
	filter, ok := s.filters[string(id)]
	if !ok || filter.UserID() != userID {
		return repository.ErrRecordFilterNotFound
	}
	delete(s.filters, string(id))
	return nil
}

func (s *stubRecordFilterRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	s.record("CountByUserID", exec)
	count := 0
	for _, filter := range s.filters {
		if filter.UserID() == userID {
			count++
		}
	}
	return count, nil
}

// recordFilterTx はトランザクション内で渡される実行先を識別するための値です。
type recordFilterTx struct {
	repository.Executor
}

type recordFilterTransactionManager struct {
	tx *recordFilterTx
}

func (m *recordFilterTransactionManager) Transactional(ctx context.Context, f func(repository.Executor) error) error {
	return f(m.tx)
}

type recordFilterUserRepository struct {
	repository.UserRepository
	repo    *stubRecordFilterRepository
	lockErr error
}

func (r *recordFilterUserRepository) FindByIDForUpdate(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	r.repo.record("LockUser", exec)
	if r.lockErr != nil {
		return nil, r.lockErr
	}
	return &entity.User{ID: id}, nil
}

func newRecordFilterUsecaseForTest(repo *stubRecordFilterRepository) RecordFilterUsecase {
	return NewRecordFilterUsecase(nil, &recordFilterTransactionManager{tx: &recordFilterTx{}}, repo, &recordFilterUserRepository{repo: repo})
}

func TestRecordFilterUsecase_CreateListUpdateDelete(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	repo := newStubRecordFilterRepository()
	uc := newRecordFilterUsecaseForTest(repo)

	created, err := uc.Create(ctx, 10, &RecordFilterInput{
		Name:          " 高難度 ",
		FilterType:    RecordFilterTypeStandard,
		SchemaVersion: 3,
		Filter:        []byte(`{"title":"","difficulties":["MASTER"]}`),
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "高難度", created.Name)
	assert.Equal(t, RecordFilterTypeStandard, created.FilterType)
	assert.Equal(t, 3, created.SchemaVersion)
	assert.JSONEq(t, `{"title":"","difficulties":["MASTER"]}`, string(created.Filter))
	assert.Equal(t, now, created.CreatedAt)
	assert.Equal(t, now, created.UpdatedAt)
	_, err = uuid.Parse(created.ID)
	require.NoError(t, err)

	standardFilters, err := uc.List(ctx, 10, RecordFilterTypeStandard)
	require.NoError(t, err)
	require.Len(t, standardFilters, 1)
	assert.Equal(t, created.ID, standardFilters[0].ID)

	worldsendFilters, err := uc.List(ctx, 10, RecordFilterTypeWorldsend)
	require.NoError(t, err)
	assert.Empty(t, worldsendFilters)

	updated, err := uc.Update(ctx, 10, created.ID, &RecordFilterInput{
		Name:          "WE用",
		FilterType:    RecordFilterTypeWorldsend,
		SchemaVersion: 2,
		Filter:        []byte(`{"attributes":["！"],"levelStarRange":{"min":4,"max":5}}`),
	})
	require.NoError(t, err)
	assert.Equal(t, "WE用", updated.Name)
	assert.Equal(t, RecordFilterTypeWorldsend, updated.FilterType)
	assert.Equal(t, 2, updated.SchemaVersion)
	assert.JSONEq(t, `{"attributes":["！"],"levelStarRange":{"min":4,"max":5}}`, string(updated.Filter))

	require.NoError(t, uc.Delete(ctx, 10, created.ID))
	err = uc.Delete(ctx, 10, created.ID)
	assert.ErrorIs(t, err, ErrRecordFilterNotFound)
}

func TestRecordFilterUsecase_CreateRejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	uc := newRecordFilterUsecaseForTest(newStubRecordFilterRepository())
	largeValue := make([]byte, info.RecordFilterMaxPayloadBytes+1)
	for i := range largeValue {
		largeValue[i] = 'a'
	}

	tests := []struct {
		name  string
		input *RecordFilterInput
	}{
		{
			name: "名前が空",
			input: &RecordFilterInput{
				Name:          " ",
				FilterType:    RecordFilterTypeStandard,
				SchemaVersion: 3,
				Filter:        []byte(`{"title":""}`),
			},
		},
		{
			name: "種別が不正",
			input: &RecordFilterInput{
				Name:          "条件",
				FilterType:    "other",
				SchemaVersion: 3,
				Filter:        []byte(`{"title":""}`),
			},
		},
		{
			name: "スキーマバージョンが不正",
			input: &RecordFilterInput{
				Name:          "条件",
				FilterType:    RecordFilterTypeStandard,
				SchemaVersion: 0,
				Filter:        []byte(`{"title":""}`),
			},
		},
		{
			name: "フィルタがオブジェクトではない",
			input: &RecordFilterInput{
				Name:          "条件",
				FilterType:    RecordFilterTypeStandard,
				SchemaVersion: 3,
				Filter:        []byte(`[]`),
			},
		},
		{
			name: "フィルタが8KBを超える",
			input: &RecordFilterInput{
				Name:          "条件",
				FilterType:    RecordFilterTypeStandard,
				SchemaVersion: 3,
				Filter:        append(append([]byte(`{"memo":"`), largeValue...), []byte(`"}`)...),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Create(ctx, 10, tt.input)
			assert.ErrorIs(t, err, ErrInvalidRecordFilterInput)
		})
	}
}

func TestRecordFilterUsecase_CreateRejectsWhenLimitExceeded(t *testing.T) {
	ctx := context.Background()
	repo := newStubRecordFilterRepository()
	for i := 0; i < info.RecordFilterMaxPerUser; i++ {
		id := uuid.NewV4()
		payload, err := gzipBytes([]byte(`{"schema_version":3,"filter":{"title":""}}`))
		require.NoError(t, err)
		filter, err := entity.RestoreRecordFilter(id[:], 10, "条件", payload, false, time.Now(), time.Now())
		require.NoError(t, err)
		repo.filters[string(id[:])] = filter
	}

	uc := newRecordFilterUsecaseForTest(repo)
	_, err := uc.Create(ctx, 10, &RecordFilterInput{
		Name:          "追加条件",
		FilterType:    RecordFilterTypeStandard,
		SchemaVersion: 3,
		Filter:        []byte(`{"title":""}`),
	})
	assert.ErrorIs(t, err, ErrRecordFilterLimitExceeded)
}

func TestRecordFilterUsecase_CreatePreservesFilterNumberExpression(t *testing.T) {
	ctx := context.Background()
	uc := newRecordFilterUsecaseForTest(newStubRecordFilterRepository())

	created, err := uc.Create(ctx, 10, &RecordFilterInput{
		Name:          "数値条件",
		FilterType:    RecordFilterTypeStandard,
		SchemaVersion: 3,
		Filter:        []byte(`{"large":9007199254740993,"decimal":1.2300}`),
	})

	require.NoError(t, err)
	assert.Equal(t, json.RawMessage(`{"large":9007199254740993,"decimal":1.2300}`), created.Filter)
}

func TestRecordFilterUsecase_UpdateRejectsInvalidID(t *testing.T) {
	uc := newRecordFilterUsecaseForTest(newStubRecordFilterRepository())

	_, err := uc.Update(context.Background(), 10, "not-uuid", &RecordFilterInput{
		Name:          "条件",
		FilterType:    RecordFilterTypeStandard,
		SchemaVersion: 3,
		Filter:        []byte(`{"title":""}`),
	})

	assert.True(t, errors.Is(err, ErrInvalidRecordFilterID))
}

func TestRecordFilterUsecase_CreateCountsAndSavesWhileUserIsLocked(t *testing.T) {
	// Given
	repo := newStubRecordFilterRepository()
	tx := &recordFilterTx{}
	uc := NewRecordFilterUsecase(nil, &recordFilterTransactionManager{tx: tx}, repo, &recordFilterUserRepository{repo: repo})

	// When
	_, err := uc.Create(context.Background(), 10, &RecordFilterInput{
		Name:          "条件",
		FilterType:    RecordFilterTypeStandard,
		SchemaVersion: 3,
		Filter:        []byte(`{"title":""}`),
	})

	// Then: 同時作成を直列化するため、ユーザー行のロック後に同じトランザクションで件数確認と保存を行う
	require.NoError(t, err)
	assert.Equal(t, []recordFilterRepositoryCall{
		{name: "LockUser", exec: tx},
		{name: "CountByUserID", exec: tx},
		{name: "Create", exec: tx},
	}, repo.calls)
}

func TestRecordFilterUsecase_UpdateDoesNotRecreateConcurrentlyDeletedFilter(t *testing.T) {
	// Given: 更新対象の取得直後に、別リクエストが同じフィルタを削除する
	ctx := context.Background()
	repo := newStubRecordFilterRepository()
	uc := newRecordFilterUsecaseForTest(repo)
	created, err := uc.Create(ctx, 10, &RecordFilterInput{
		Name:          "条件",
		FilterType:    RecordFilterTypeStandard,
		SchemaVersion: 3,
		Filter:        []byte(`{"title":""}`),
	})
	require.NoError(t, err)
	repo.afterFind = func() {
		repo.afterFind = nil
		require.NoError(t, uc.Delete(ctx, 10, created.ID))
	}

	// When
	_, err = uc.Update(ctx, 10, created.ID, &RecordFilterInput{
		Name:          "更新後",
		FilterType:    RecordFilterTypeStandard,
		SchemaVersion: 3,
		Filter:        []byte(`{"title":""}`),
	})

	// Then: 削除を取り消さず、未検出として扱う
	assert.ErrorIs(t, err, ErrRecordFilterNotFound)
	assert.Empty(t, repo.filters)
}

func TestRecordFilterUsecase_CreateReturnsUserNotFoundWhenUserIsDeleted(t *testing.T) {
	// Given: 認証後、作成処理がユーザー行をロックする前に退会している
	repo := newStubRecordFilterRepository()
	userRepo := &recordFilterUserRepository{repo: repo, lockErr: repository.ErrUserNotFound}
	uc := NewRecordFilterUsecase(nil, &recordFilterTransactionManager{tx: &recordFilterTx{}}, repo, userRepo)

	// When
	_, err := uc.Create(context.Background(), 10, &RecordFilterInput{
		Name:          "条件",
		FilterType:    RecordFilterTypeStandard,
		SchemaVersion: 3,
		Filter:        []byte(`{"title":""}`),
	})

	// Then: 内部エラーではなくユーザー未検出として扱い、保存しない
	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Empty(t, repo.filters)
}
