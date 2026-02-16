package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSongRepository は SongRepository のモックです。
type MockSongRepository struct {
	mock.Mock
}

func (m *MockSongRepository) FindAllExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*entity.Song, error) {
	args := m.Called(ctx, exec, includeDeleted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Song), args.Error(1)
}

func (m *MockSongRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*entity.Song, error) {
	args := m.Called(ctx, exec, displayID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Song), args.Error(1)
}

func (m *MockSongRepository) FindByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) ([]*entity.Song, error) {
	args := m.Called(ctx, exec, displayIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Song), args.Error(1)
}

func (m *MockSongRepository) DeleteSong(ctx context.Context, exec repository.Executor, displayID string) error {
	args := m.Called(ctx, exec, displayID)
	return args.Error(0)
}

func (m *MockSongRepository) RestoreSong(ctx context.Context, exec repository.Executor, displayID string) error {
	args := m.Called(ctx, exec, displayID)
	return args.Error(0)
}

func (m *MockSongRepository) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song) error {
	args := m.Called(ctx, exec, songs)
	return args.Error(0)
}

// MockSongMasterProvider は SongMasterProvider のモックです。
type MockSongMasterProvider struct {
	mock.Mock
}

func (m *MockSongMasterProvider) SongMasters() *masterdata.SongMasters {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*masterdata.SongMasters)
}

func TestGetAllSongsExcludingWorldsend_WithDeletedSongs_RequiresEditorPermission(t *testing.T) {
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
			mockRepo := new(MockSongRepository)
			mockMasterCache := new(MockSongMasterProvider)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)

			usecase := NewSongService(mockRepo, mockMasterCache, mockTM, mockExec)

			ctx := context.Background()

			// 期待されるリポジトリの呼び出し
			expectedSongs := []*entity.Song{
				{
					ID:          1,
					DisplayID:   "S001",
					Title:       "Active Song",
					IsWorldsend: false,
					IsDeleted:   false,
				},
			}

			// tt.expectedIncludeDeleted に基づいてリポジトリが呼び出されることを期待
			mockRepo.On("FindAllExcludingWorldsend", ctx, mockExec, tt.expectedIncludeDeleted).Return(expectedSongs, nil)

			// Act
			result, err := usecase.GetAllSongsExcludingWorldsend(ctx, tt.includeDeleted, tt.requesterAccountTypeID)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, expectedSongs, result)
			mockRepo.AssertExpectations(t)
		})
	}
}
