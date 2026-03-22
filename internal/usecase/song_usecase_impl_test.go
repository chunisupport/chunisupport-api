package usecase

import (
	"context"
	"testing"
	"time"

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

func (m *MockSongRepository) GetLatestUpdatedAtExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) (*time.Time, error) {
	args := m.Called(ctx, exec, includeDeleted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	timeVal := args.Get(0).(time.Time)
	return &timeVal, args.Error(1)
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

func (m *MockSongRepository) Save(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	args := m.Called(ctx, exec, song)
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

type passthroughTransactionManager struct {
	tx repository.Executor
}

func (m *passthroughTransactionManager) Transactional(_ context.Context, fn func(tx repository.Executor) error) error {
	return fn(m.tx)
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
	}{
		{
			name:                   "editor権限なら削除済みを取得できる",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeEditor),
			expectedIncludeDeleted: true,
		},
		{
			name:                   "admin権限なら削除済みを取得できる",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeAdmin),
			expectedIncludeDeleted: true,
		},
		{
			name:                   "player権限では削除済み取得を無効化する",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypePlayer),
			expectedIncludeDeleted: false,
		},
		{
			name:                   "未知の権限では削除済み取得を無効化する",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(4),
			expectedIncludeDeleted: false,
		},
		{
			name:                   "権限なしでは削除済み取得を無効化する",
			includeDeleted:         true,
			requesterAccountTypeID: nil,
			expectedIncludeDeleted: false,
		},
		{
			name:                   "includeDeletedがfalseならそのままfalseになる",
			includeDeleted:         false,
			requesterAccountTypeID: nil,
			expectedIncludeDeleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSongRepository)
			mockMasterCache := new(MockSongMasterProvider)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)

			uc := NewSongService(mockRepo, mockMasterCache, mockTM, mockExec)
			ctx := context.Background()
			expectedSongs := []*entity.Song{{
				ID:          1,
				DisplayID:   "S001",
				Title:       "Active Song",
				IsWorldsend: false,
				IsDeleted:   false,
			}}

			mockRepo.On("FindAllExcludingWorldsend", ctx, mockExec, tt.expectedIncludeDeleted).Return(expectedSongs, nil)

			result, err := uc.GetAllSongsExcludingWorldsend(ctx, tt.includeDeleted, tt.requesterAccountTypeID)

			assert.NoError(t, err)
			assert.Equal(t, expectedSongs, result)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetSongByDisplayID_DeletedSongPermission(t *testing.T) {
	activeSong := &entity.Song{
		ID:        1,
		DisplayID: "S001",
		Title:     "Active Song",
		IsDeleted: false,
		Charts:    []*entity.Chart{},
	}
	deletedSong := &entity.Song{
		ID:        2,
		DisplayID: "S002",
		Title:     "Deleted Song",
		IsDeleted: true,
		Charts:    []*entity.Chart{},
	}

	tests := []struct {
		name                   string
		displayID              string
		requesterAccountTypeID *int
		repoReturn             *entity.Song
		repoErr                error
		wantResult             *entity.Song
		wantErr                error
	}{
		{
			name:                   "通常楽曲は誰でも取得できる",
			displayID:              "S001",
			requesterAccountTypeID: nil,
			repoReturn:             activeSong,
			wantResult:             activeSong,
		},
		{
			name:                   "削除済み楽曲はeditorなら取得できる",
			displayID:              "S002",
			requesterAccountTypeID: intPtr(info.AccountTypeEditor),
			repoReturn:             deletedSong,
			wantResult:             deletedSong,
		},
		{
			name:                   "削除済み楽曲はplayerなら見つからない扱いになる",
			displayID:              "S002",
			requesterAccountTypeID: intPtr(info.AccountTypePlayer),
			repoReturn:             deletedSong,
			wantErr:                repository.ErrSongNotFound,
		},
		{
			name:                   "削除済み楽曲は未知の権限なら見つからない扱いになる",
			displayID:              "S002",
			requesterAccountTypeID: intPtr(4),
			repoReturn:             deletedSong,
			wantErr:                repository.ErrSongNotFound,
		},
		{
			name:                   "削除済み楽曲は権限なしなら見つからない扱いになる",
			displayID:              "S002",
			requesterAccountTypeID: nil,
			repoReturn:             deletedSong,
			wantErr:                repository.ErrSongNotFound,
		},
		{
			name:                   "存在しない楽曲は見つからないエラーを返す",
			displayID:              "S999",
			requesterAccountTypeID: intPtr(info.AccountTypeAdmin),
			repoErr:                repository.ErrSongNotFound,
			wantErr:                repository.ErrSongNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSongRepository)
			mockMasterCache := new(MockSongMasterProvider)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)

			uc := NewSongService(mockRepo, mockMasterCache, mockTM, mockExec)
			ctx := context.Background()

			mockRepo.On("FindByDisplayID", ctx, mockExec, tt.displayID).Return(tt.repoReturn, tt.repoErr)

			result, err := uc.GetSongByDisplayID(ctx, tt.displayID, tt.requesterAccountTypeID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetSongsLastUpdatedAt_WithDeletedSongs_RequiresEditorPermission(t *testing.T) {
	tests := []struct {
		name                   string
		includeDeleted         bool
		requesterAccountTypeID *int
		expectedIncludeDeleted bool
	}{
		{
			name:                   "editor権限なら削除済みを含めて最終更新日時を取得する",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeEditor),
			expectedIncludeDeleted: true,
		},
		{
			name:                   "権限なしでは削除済みを含めずに最終更新日時を取得する",
			includeDeleted:         true,
			requesterAccountTypeID: nil,
			expectedIncludeDeleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSongRepository)
			mockMasterCache := new(MockSongMasterProvider)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)

			uc := NewSongService(mockRepo, mockMasterCache, mockTM, mockExec)
			ctx := context.Background()
			expectedUpdatedAt := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)

			mockRepo.On("GetLatestUpdatedAtExcludingWorldsend", ctx, mockExec, tt.expectedIncludeDeleted).Return(expectedUpdatedAt, nil)

			result, err := uc.GetSongsLastUpdatedAt(ctx, tt.includeDeleted, tt.requesterAccountTypeID)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, expectedUpdatedAt.Equal(*result))
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteSong_SavesDeletedState(t *testing.T) {
	mockRepo := new(MockSongRepository)
	mockMasterCache := new(MockSongMasterProvider)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := NewSongService(mockRepo, mockMasterCache, tm, mockExec)
	ctx := context.Background()

	song := &entity.Song{
		ID:        10,
		DisplayID: "S010",
		IsDeleted: false,
		Charts:    []*entity.Chart{},
	}
	mockRepo.On("FindByDisplayID", ctx, mockExec, "S010").Return(song, nil).Once()
	mockRepo.On("Save", ctx, mockExec, mock.MatchedBy(func(saved *entity.Song) bool {
		return saved == song && saved.IsDeleted
	})).Return(nil).Once()

	err := uc.DeleteSong(ctx, "S010")

	assert.NoError(t, err)
	assert.True(t, song.IsDeleted)
	mockRepo.AssertExpectations(t)
}

func TestRestoreSong_SavesRestoredState(t *testing.T) {
	mockRepo := new(MockSongRepository)
	mockMasterCache := new(MockSongMasterProvider)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := NewSongService(mockRepo, mockMasterCache, tm, mockExec)
	ctx := context.Background()

	song := &entity.Song{
		ID:        11,
		DisplayID: "S011",
		IsDeleted: true,
		Charts:    []*entity.Chart{},
	}
	mockRepo.On("FindByDisplayID", ctx, mockExec, "S011").Return(song, nil).Once()
	mockRepo.On("Save", ctx, mockExec, mock.MatchedBy(func(saved *entity.Song) bool {
		return saved == song && !saved.IsDeleted
	})).Return(nil).Once()

	err := uc.RestoreSong(ctx, "S011")

	assert.NoError(t, err)
	assert.False(t, song.IsDeleted)
	mockRepo.AssertExpectations(t)
}
