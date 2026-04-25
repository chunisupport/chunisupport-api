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

func (m *MockSongRepository) FindLatestUpdatedAt(ctx context.Context, exec repository.Executor) (*time.Time, error) {
	args := m.Called(ctx, exec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *MockSongRepository) Save(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	args := m.Called(ctx, exec, song)
	return args.Error(0)
}

func (m *MockSongRepository) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song) error {
	args := m.Called(ctx, exec, songs)
	return args.Error(0)
}

func (m *MockSongRepository) Create(ctx context.Context, exec repository.Executor, song *entity.Song) (*entity.Song, error) {
	args := m.Called(ctx, exec, song)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Song), args.Error(1)
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
		description            string
	}{
		{
			name:                   "EDITOR権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: new(info.AccountTypeEditor),
			expectedIncludeDeleted: true,
			description:            "EDITOR権限がある場合、includeDeleted=trueで削除済み楽曲を取得できる",
		},
		{
			name:                   "ADMIN権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: new(info.AccountTypeAdmin),
			expectedIncludeDeleted: true,
			description:            "ADMIN権限がある場合、includeDeleted=trueで削除済み楽曲を取得できる",
		},
		{
			name:                   "PLAYER権限のみ_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: new(info.AccountTypePlayer),
			expectedIncludeDeleted: false,
			description:            "PLAYER権限の場合、includeDeleted=trueでも削除済み楽曲は除外される",
		},
		{
			name:                   "未知ロール_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: new(4),
			expectedIncludeDeleted: false,
			description:            "未知ロールIDは権限なしとして扱われ、削除済み楽曲は除外される",
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
			// Given
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

			// When
			result, err := usecase.GetAllSongsExcludingWorldsend(ctx, tt.includeDeleted, tt.requesterAccountTypeID)

			// Then
			assert.NoError(t, err)
			assert.NotNil(t, result)
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
		name string
		// Given
		displayID              string
		requesterAccountTypeID *int
		repoReturn             *entity.Song
		repoErr                error
		// Then
		wantResult *entity.Song
		wantErr    error
	}{
		{
			name:                   "有効な楽曲は誰でも取得できる",
			displayID:              "S001",
			requesterAccountTypeID: nil,
			repoReturn:             activeSong,
			wantResult:             activeSong,
		},
		{
			name:                   "削除済み楽曲はEDITOR権限で取得できる",
			displayID:              "S002",
			requesterAccountTypeID: new(info.AccountTypeEditor),
			repoReturn:             deletedSong,
			wantResult:             deletedSong,
		},
		{
			name:                   "削除済み楽曲はPLAYER権限ではErrSongNotFoundになる",
			displayID:              "S002",
			requesterAccountTypeID: new(info.AccountTypePlayer),
			repoReturn:             deletedSong,
			wantErr:                repository.ErrSongNotFound,
		},
		{
			name:                   "削除済み楽曲は未知ロールではErrSongNotFoundになる",
			displayID:              "S002",
			requesterAccountTypeID: new(4),
			repoReturn:             deletedSong,
			wantErr:                repository.ErrSongNotFound,
		},
		{
			name:                   "削除済み楽曲は権限なしではErrSongNotFoundになる",
			displayID:              "S002",
			requesterAccountTypeID: nil,
			repoReturn:             deletedSong,
			wantErr:                repository.ErrSongNotFound,
		},
		{
			name:                   "存在しない楽曲はErrSongNotFoundを返す",
			displayID:              "S999",
			requesterAccountTypeID: new(info.AccountTypeAdmin),
			repoErr:                repository.ErrSongNotFound,
			wantErr:                repository.ErrSongNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockSongRepository)
			mockMasterCache := new(MockSongMasterProvider)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)

			uc := NewSongService(mockRepo, mockMasterCache, mockTM, mockExec)
			ctx := context.Background()

			mockRepo.On("FindByDisplayID", ctx, mockExec, tt.displayID).Return(tt.repoReturn, tt.repoErr)

			// When
			result, err := uc.GetSongByDisplayID(ctx, tt.displayID, tt.requesterAccountTypeID)

			// Then
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

func TestDeleteSong_SavesDeletedState(t *testing.T) {
	// Given
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

	// When
	err := uc.DeleteSong(ctx, "S010")

	// Then
	assert.NoError(t, err)
	assert.True(t, song.IsDeleted)
	mockRepo.AssertExpectations(t)
}

func TestRestoreSong_SavesRestoredState(t *testing.T) {
	// Given
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

	// When
	err := uc.RestoreSong(ctx, "S011")

	// Then
	assert.NoError(t, err)
	assert.False(t, song.IsDeleted)
	mockRepo.AssertExpectations(t)
}

func TestGetSongsUpdatedAt_ReturnsRepositoryValue(t *testing.T) {
	// Given
	mockRepo := new(MockSongRepository)
	mockMasterCache := new(MockSongMasterProvider)
	mockTM := new(MockTransactionManager)
	mockExec := new(MockExecutor)
	uc := NewSongService(mockRepo, mockMasterCache, mockTM, mockExec)
	ctx := context.Background()
	expected := time.Date(2026, 4, 9, 12, 34, 56, 0, time.UTC)

	mockRepo.On("FindLatestUpdatedAt", ctx, mockExec).Return(&expected, nil).Once()

	// When
	result, err := uc.GetSongsUpdatedAt(ctx)

	// Then
	assert.NoError(t, err)
	if assert.NotNil(t, result) {
		assert.True(t, expected.Equal(*result))
	}
	mockRepo.AssertExpectations(t)
}
