package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	dtoapi "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
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

func (m *MockWorldsendChartRepository) SaveSong(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	args := m.Called(ctx, exec, song)
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

type stubWorldsendSongMasterProvider struct {
	masters *domainmasterdata.SongMasters
}

func (s *stubWorldsendSongMasterProvider) SongMasters() *domainmasterdata.SongMasters {
	return s.masters
}

func newWorldsendUsecaseForTest(repo repository.WorldsendChartRepository, tm TransactionManager, exec repository.Executor) WorldsendUsecase {
	return NewWorldsendUsecase(repo, &stubWorldsendSongMasterProvider{
		masters: &domainmasterdata.SongMasters{
			Genres: map[string]domainmasterdata.Item{
				"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"},
			},
		},
	}, tm, exec)
}

func TestGetAllWorldsendSongs_WithDeletedSongs_RequiresEditorPermission(t *testing.T) {
	tests := []struct {
		name                   string
		includeDeleted         bool
		requesterAccountTypeID *int
		expectedIncludeDeleted bool
	}{
		{
			name:                   "EDITOR権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeEditor),
			expectedIncludeDeleted: true,
		},
		{
			name:                   "ADMIN権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeAdmin),
			expectedIncludeDeleted: true,
		},
		{
			name:                   "PLAYER権限のみ_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypePlayer),
			expectedIncludeDeleted: false,
		},
		{
			name:                   "権限なし_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: nil,
			expectedIncludeDeleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockWorldsendChartRepository)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)
			uc := newWorldsendUsecaseForTest(mockRepo, mockTM, mockExec)
			ctx := context.Background()

			expectedSongs := []*repository.WorldsendSongWithChart{{
				Song:  &entity.Song{ID: 1, DisplayID: "WE001", IsWorldsend: true, IsDeleted: false},
				Chart: &entity.WorldsendChart{ID: 1, SongID: 1},
			}}
			mockRepo.On("FindAll", ctx, mockExec, tt.expectedIncludeDeleted).Return(expectedSongs, nil)

			// When
			result, err := uc.GetAllWorldsendSongs(ctx, tt.includeDeleted, tt.requesterAccountTypeID)

			// Then
			assert.NoError(t, err)
			assert.Equal(t, expectedSongs, result)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetWorldsendSongByDisplayID_DeletedSongPermission(t *testing.T) {
	activeSong := &repository.WorldsendSongWithChart{
		Song:  &entity.Song{ID: 1, DisplayID: "WE001", IsWorldsend: true, IsDeleted: false},
		Chart: &entity.WorldsendChart{ID: 1, SongID: 1},
	}
	deletedSong := &repository.WorldsendSongWithChart{
		Song:  &entity.Song{ID: 2, DisplayID: "WE002", IsWorldsend: true, IsDeleted: true},
		Chart: &entity.WorldsendChart{ID: 2, SongID: 2},
	}

	tests := []struct {
		name                   string
		displayID              string
		requesterAccountTypeID *int
		repoReturn             *repository.WorldsendSongWithChart
		repoErr                error
		wantResult             *repository.WorldsendSongWithChart
		wantErr                error
	}{
		{
			name:                   "有効な楽曲は誰でも取得できる",
			displayID:              "WE001",
			requesterAccountTypeID: nil,
			repoReturn:             activeSong,
			wantResult:             activeSong,
		},
		{
			name:                   "削除済み楽曲はEDITOR権限で取得できる",
			displayID:              "WE002",
			requesterAccountTypeID: intPtr(info.AccountTypeEditor),
			repoReturn:             deletedSong,
			wantResult:             deletedSong,
		},
		{
			name:                   "削除済み楽曲はPLAYER権限ではErrSongNotFoundになる",
			displayID:              "WE002",
			requesterAccountTypeID: intPtr(info.AccountTypePlayer),
			repoReturn:             deletedSong,
			wantErr:                repository.ErrSongNotFound,
		},
		{
			name:                   "存在しない楽曲はErrSongNotFoundを返す",
			displayID:              "WE999",
			requesterAccountTypeID: intPtr(info.AccountTypeAdmin),
			repoErr:                repository.ErrSongNotFound,
			wantErr:                repository.ErrSongNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockWorldsendChartRepository)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)
			uc := newWorldsendUsecaseForTest(mockRepo, mockTM, mockExec)
			ctx := context.Background()
			mockRepo.On("FindByDisplayID", ctx, mockExec, tt.displayID).Return(tt.repoReturn, tt.repoErr)

			// When
			result, err := uc.GetWorldsendSongByDisplayID(ctx, tt.displayID, tt.requesterAccountTypeID)

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

func TestDeleteWorldsendSong_SavesDeletedState(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	ctx := context.Background()

	songWithChart := &repository.WorldsendSongWithChart{
		Song:  &entity.Song{ID: 21, DisplayID: "WE021", IsWorldsend: true, IsDeleted: false, Charts: []*entity.Chart{}},
		Chart: &entity.WorldsendChart{ID: 210, SongID: 21},
	}
	mockRepo.On("FindByDisplayID", ctx, mockExec, "WE021").Return(songWithChart, nil).Once()
	mockRepo.On("SaveSong", ctx, mockExec, mock.MatchedBy(func(song *entity.Song) bool {
		return song == songWithChart.Song && song.IsDeleted
	})).Return(nil).Once()

	// When
	err := uc.DeleteWorldsendSong(ctx, "WE021")

	// Then
	assert.NoError(t, err)
	assert.True(t, songWithChart.Song.IsDeleted)
	mockRepo.AssertExpectations(t)
}

func TestRestoreWorldsendSong_SavesRestoredState(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	ctx := context.Background()

	songWithChart := &repository.WorldsendSongWithChart{
		Song:  &entity.Song{ID: 22, DisplayID: "WE022", IsWorldsend: true, IsDeleted: true, Charts: []*entity.Chart{}},
		Chart: &entity.WorldsendChart{ID: 220, SongID: 22},
	}
	mockRepo.On("FindByDisplayID", ctx, mockExec, "WE022").Return(songWithChart, nil).Once()
	mockRepo.On("SaveSong", ctx, mockExec, mock.MatchedBy(func(song *entity.Song) bool {
		return song == songWithChart.Song && !song.IsDeleted
	})).Return(nil).Once()

	// When
	err := uc.RestoreWorldsendSong(ctx, "WE022")

	// Then
	assert.NoError(t, err)
	assert.False(t, songWithChart.Song.IsDeleted)
	mockRepo.AssertExpectations(t)
}

func TestUpdateWorldsendSongs_ConvertsAndSaves(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	ctx := context.Background()

	notes := 2000
	level := 3
	attribute := "狂"
	genre := "POPS & ANIME"
	requests := []*dtoapi.UpdateWorldsendSongRequest{
		{DisplayID: "1234567890abcdef", Title: "A", Artist: "AR", Genre: &genre},
		{DisplayID: "abcdef1234567890", Title: "B", Artist: "BR", Charts: map[string]*dtoapi.UpdateWorldsendChartRequest{
			"WORLDSEND": {Notes: &notes, LevelStar: &level, Attribute: &attribute},
		}},
	}

	mockRepo.On("UpdateSongs", ctx, mockExec, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		songs := args.Get(2).([]*entity.Song)
		charts := args.Get(3).([]*entity.WorldsendChart)
		assert.Len(t, songs, 2)
		assert.Len(t, charts, 2)
		assert.Equal(t, "1234567890abcdef", songs[0].DisplayID)
		if assert.NotNil(t, songs[0].GenreID) {
			assert.Equal(t, 1, *songs[0].GenreID)
		}
		assert.Nil(t, charts[0])
		assert.Equal(t, "abcdef1234567890", songs[1].DisplayID)
		if assert.NotNil(t, charts[1]) {
			if assert.NotNil(t, charts[1].Notes) {
				assert.Equal(t, 2000, int(*charts[1].Notes))
			}
		}
	}).Return(nil).Once()

	// When
	err := uc.UpdateWorldsendSongs(ctx, requests)

	// Then
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateWorldsendSongs_InvalidInputReturnsError(t *testing.T) {
	tests := []struct {
		name     string
		requests []*dtoapi.UpdateWorldsendSongRequest
	}{
		{
			name: "WORLDSEND以外のchartsキーはエラー",
			requests: []*dtoapi.UpdateWorldsendSongRequest{{
				DisplayID: "1234567890abcdef",
				Title:     "A",
				Artist:    "AR",
				Charts: map[string]*dtoapi.UpdateWorldsendChartRequest{
					"MASTER": {},
				},
			}},
		},
		{
			name: "存在しないgenreはエラー",
			requests: []*dtoapi.UpdateWorldsendSongRequest{{
				DisplayID: "1234567890abcdef",
				Title:     "A",
				Artist:    "AR",
				Genre:     strPtr("UNKNOWN"),
			}},
		},
		{
			name: "notesの値オブジェクト生成失敗はエラー",
			requests: []*dtoapi.UpdateWorldsendSongRequest{{
				DisplayID: "1234567890abcdef",
				Title:     "A",
				Artist:    "AR",
				Charts: map[string]*dtoapi.UpdateWorldsendChartRequest{
					"WORLDSEND": {Notes: intPtr(-1)},
				},
			}},
		},
		{
			name: "リクエスト内に重複したdisplay_idがある場合はエラー",
			requests: []*dtoapi.UpdateWorldsendSongRequest{
				{DisplayID: "1234567890abcdef", Title: "A", Artist: "AR"},
				{DisplayID: "1234567890abcdef", Title: "B", Artist: "BR"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockWorldsendChartRepository)
			mockExec := new(MockExecutor)
			tm := &passthroughTransactionManager{tx: mockExec}
			uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)

			// When
			err := uc.UpdateWorldsendSongs(context.Background(), tt.requests)

			// Then
			assert.ErrorIs(t, err, ErrInvalidWorldsendInput)
			mockRepo.AssertNotCalled(t, "UpdateSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateWorldsendSongs_InvalidChartReturnsError(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	invalidLevel := 0
	requests := []*dtoapi.UpdateWorldsendSongRequest{{
		DisplayID: "1234567890abcdef",
		Title:     "A",
		Artist:    "AR",
		Charts: map[string]*dtoapi.UpdateWorldsendChartRequest{
			"WORLDSEND": {LevelStar: &invalidLevel},
		},
	}}

	// When
	err := uc.UpdateWorldsendSongs(context.Background(), requests)

	// Then
	assert.ErrorIs(t, err, ErrInvalidWorldsendInput)
	mockRepo.AssertNotCalled(t, "UpdateSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestUpdateWorldsendSongs_DuplicateDisplayIDIsMappedToValidationError はリポジトリ防衛チェックが
// ErrDuplicateDisplayID を返した際に ErrInvalidWorldsendInput へ変換されることを検証します。
func TestUpdateWorldsendSongs_DuplicateDisplayIDIsMappedToValidationError(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	requests := []*dtoapi.UpdateWorldsendSongRequest{{
		DisplayID: "1234567890abcdef",
		Title:     "A",
		Artist:    "AR",
	}}

	mockRepo.On("UpdateSongs", mock.Anything, mockExec, mock.Anything, mock.Anything).
		Return(repository.ErrDuplicateDisplayID).
		Once()

	// When
	err := uc.UpdateWorldsendSongs(context.Background(), requests)

	// Then
	assert.ErrorIs(t, err, ErrInvalidWorldsendInput)
	assert.ErrorIs(t, err, repository.ErrDuplicateDisplayID)
	mockRepo.AssertExpectations(t)
}

// intPtr はint値へのポインタを返すヘルパー関数です。
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
