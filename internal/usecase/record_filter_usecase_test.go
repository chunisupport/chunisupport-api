package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRecordFilterRepository struct {
	filters map[string]*entity.RecordFilter
}

func newStubRecordFilterRepository() *stubRecordFilterRepository {
	return &stubRecordFilterRepository{filters: map[string]*entity.RecordFilter{}}
}

func (s *stubRecordFilterRepository) ListByUserID(ctx context.Context, userID int) ([]*entity.RecordFilter, error) {
	filters := make([]*entity.RecordFilter, 0, len(s.filters))
	for _, filter := range s.filters {
		if filter.UserID() == userID {
			filters = append(filters, filter)
		}
	}
	return filters, nil
}

func (s *stubRecordFilterRepository) FindByIDAndUserID(ctx context.Context, id []byte, userID int) (*entity.RecordFilter, error) {
	filter, ok := s.filters[string(id)]
	if !ok || filter.UserID() != userID {
		return nil, repository.ErrRecordFilterNotFound
	}
	return filter, nil
}

func (s *stubRecordFilterRepository) Save(ctx context.Context, filter *entity.RecordFilter) error {
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

func (s *stubRecordFilterRepository) DeleteByIDAndUserID(ctx context.Context, id []byte, userID int) error {
	filter, ok := s.filters[string(id)]
	if !ok || filter.UserID() != userID {
		return repository.ErrRecordFilterNotFound
	}
	delete(s.filters, string(id))
	return nil
}

func (s *stubRecordFilterRepository) CountByUserID(ctx context.Context, userID int) (int, error) {
	count := 0
	for _, filter := range s.filters {
		if filter.UserID() == userID {
			count++
		}
	}
	return count, nil
}

func TestRecordFilterUsecase_CreateListUpdateDelete(t *testing.T) {
	ctx := context.Background()
	repo := newStubRecordFilterRepository()
	uc := NewRecordFilterUsecase(repo)

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
	assert.Equal(t, "2026-06-15T12:00:00Z", created.CreatedAt)
	assert.Equal(t, "2026-06-15T12:00:00Z", created.UpdatedAt)
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
	uc := NewRecordFilterUsecase(newStubRecordFilterRepository())
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
		id := uuid.New()
		idBytes, err := id.MarshalBinary()
		require.NoError(t, err)
		payload, err := gzipBytes([]byte(`{"schema_version":3,"filter":{"title":""}}`))
		require.NoError(t, err)
		filter, err := entity.RestoreRecordFilter(idBytes, 10, "条件", payload, false, time.Now(), time.Now())
		require.NoError(t, err)
		repo.filters[string(idBytes)] = filter
	}

	uc := NewRecordFilterUsecase(repo)
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
	uc := NewRecordFilterUsecase(newStubRecordFilterRepository())

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
	uc := NewRecordFilterUsecase(newStubRecordFilterRepository())

	_, err := uc.Update(context.Background(), 10, "not-uuid", &RecordFilterInput{
		Name:          "条件",
		FilterType:    RecordFilterTypeStandard,
		SchemaVersion: 3,
		Filter:        []byte(`{"title":""}`),
	})

	assert.True(t, errors.Is(err, ErrInvalidRecordFilterID))
}
