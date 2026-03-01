package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorldsendChartRepository は WorldsendChartRepository のモックです。
type MockWorldsendChartRepository struct {
	mock.Mock
}

func (m *MockWorldsendChartRepository) FindAll(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*repository.WorldsendSongWithChart, error) {
	args := m.Called(ctx, exec, includeDeleted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.WorldsendSongWithChart), args.Error(1)
}

func (m *MockWorldsendChartRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*repository.WorldsendSongWithChart, error) {
	args := m.Called(ctx, exec, displayID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.WorldsendSongWithChart), args.Error(1)
}

func (m *MockWorldsendChartRepository) DeleteSong(ctx context.Context, exec repository.Executor, displayID string) error {
	args := m.Called(ctx, exec, displayID)
	return args.Error(0)
}

func (m *MockWorldsendChartRepository) RestoreSong(ctx context.Context, exec repository.Executor, displayID string) error {
	args := m.Called(ctx, exec, displayID)
	return args.Error(0)
}

func (m *MockWorldsendChartRepository) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song, charts []*entity.WorldsendChart) error {
	args := m.Called(ctx, exec, songs, charts)
	return args.Error(0)
}

// MockTransactionManager は TransactionManager のモックです。
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) Transactional(ctx context.Context, fn func(tx repository.Executor) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func TestGetAllWorldsendSongs_WithDeletedSongs_RequiresEditorPermission(t *testing.T) {
	tests := []struct {
		name                   string
		includeDeleted         bool
		requesterAccountTypeID *int
		expectedIncludeDeleted bool
		description            string
	}{
		{
			name:                   "EDITOR権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeEditor),
			expectedIncludeDeleted: true,
			description:            "EDITOR権限がある場合、includeDeleted=trueで削除済み楽曲を取得できる",
		},
		{
			name:                   "ADMIN権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeAdmin),
			expectedIncludeDeleted: true,
			description:            "ADMIN権限がある場合、includeDeleted=trueで削除済み楽曲を取得できる",
		},
		{
			name:                   "PLAYER権限のみ_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypePlayer),
			expectedIncludeDeleted: false,
			description:            "PLAYER権限の場合、includeDeleted=trueでも削除済み楽曲は除外される",
		},
		{
			name:                   "権限なし_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: nil,
			expectedIncludeDeleted: false,
			description:            "権限がない場合、includeDeleted=trueでも削除済み楽曲は除外される",
		},
		{
			name:                   "権限なし_includeDeleted=false_削除済みを除外",
			includeDeleted:         false,
			requesterAccountTypeID: nil,
			expectedIncludeDeleted: false,
			description:            "includeDeleted=falseの場合、削除済み楽曲は除外される",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockWorldsendChartRepository)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)

			usecase := NewWorldsendUsecase(mockRepo, mockTM, mockExec)

			ctx := context.Background()

			// 期待されるリポジトリの呼び出し
			expectedSongs := []*repository.WorldsendSongWithChart{
				{
					Song: &entity.Song{
						ID:          1,
						DisplayID:   "WE001",
						Title:       "Active Song",
						IsWorldsend: true,
						IsDeleted:   false,
					},
					Chart: &entity.WorldsendChart{
						ID:     1,
						SongID: 1,
					},
				},
			}

			// tt.expectedIncludeDeleted に基づいてリポジトリが呼び出されることを期待
			mockRepo.On("FindAll", ctx, mockExec, tt.expectedIncludeDeleted).Return(expectedSongs, nil)

			// Act
			result, err := usecase.GetAllWorldsendSongs(ctx, tt.includeDeleted, tt.requesterAccountTypeID)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, expectedSongs, result)
			mockRepo.AssertExpectations(t)
		})
	}
}

// intPtr はint値へのポインタを返すヘルパー関数です。
func intPtr(i int) *int {
	return &i
}
