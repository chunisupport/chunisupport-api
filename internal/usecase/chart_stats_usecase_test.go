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

type MockChartStatsRepository struct {
	mock.Mock
}

func (m *MockChartStatsRepository) FindRatingBands(ctx context.Context, exec repository.Executor) ([]*entity.RatingBand, error) {
	args := m.Called(ctx, exec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.RatingBand), args.Error(1)
}

func (m *MockChartStatsRepository) FindChartStatsByChartIDs(ctx context.Context, exec repository.Executor, chartIDs []int) ([]*entity.ChartStatsByRatingBand, error) {
	args := m.Called(ctx, exec, chartIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ChartStatsByRatingBand), args.Error(1)
}

type StubChartStatsMasterProvider struct {
	bands []*entity.RatingBand
}

func (s *StubChartStatsMasterProvider) RatingBands() []*entity.RatingBand {
	return s.bands
}

func TestGetSongStatsByDisplayID_SortByRatingBandOrder(t *testing.T) {
	ctx := context.Background()
	mockSongRepo := new(MockSongRepository)
	mockWorldsendRepo := new(MockWorldsendChartRepository)
	mockStatsRepo := new(MockChartStatsRepository)
	mockSongMasterProvider := new(MockSongMasterProvider)
	mockExec := new(MockExecutor)
	stubMasterProvider := &StubChartStatsMasterProvider{bands: []*entity.RatingBand{{ID: 10, SortOrder: 2}, {ID: 20, SortOrder: 1}}}

	u := NewChartStatsUsecase(mockSongRepo, mockWorldsendRepo, mockStatsRepo, mockSongMasterProvider, stubMasterProvider, mockExec, mockExec)

	song := &entity.Song{DisplayID: "S001", Charts: []*entity.Chart{{ID: 101, DifficultyID: 3}}}
	mockSongRepo.On("FindByDisplayID", ctx, mockExec, "S001").Return(song, nil)
	mockSongMasterProvider.On("SongMasters").Return(&masterdata.SongMasters{CommonMasters: masterdata.CommonMasters{DifficultyNamesByID: map[int]string{3: "EXPERT"}}})
	mockStatsRepo.On("FindChartStatsByChartIDs", ctx, mockExec, []int{101}).Return([]*entity.ChartStatsByRatingBand{{ChartID: 101, RatingBandID: 10}, {ChartID: 101, RatingBandID: 20}}, nil)

	result, err := u.GetSongStatsByDisplayID(ctx, "S001", nil)

	assert.NoError(t, err)
	assert.Equal(t, []int{20, 10}, []int{result.Charts["EXPERT"][0].RatingBandID, result.Charts["EXPERT"][1].RatingBandID})
}

func TestGetSongStatsByDisplayID_DeletedSongPermissionBranch(t *testing.T) {
	tests := []struct {
		name               string
		accountTypeID      *int
		setupMocks         func(ctx context.Context, songRepo *MockSongRepository, statsRepo *MockChartStatsRepository, songMasterProvider *MockSongMasterProvider, exec *MockExecutor)
		assertResult       func(t *testing.T, result *entity.SongChartStats)
		expectedErrChecker assert.ErrorAssertionFunc
	}{
		{
			name:          "権限がない場合は削除済み楽曲が取得できない",
			accountTypeID: nil,
			setupMocks: func(ctx context.Context, songRepo *MockSongRepository, _ *MockChartStatsRepository, _ *MockSongMasterProvider, exec *MockExecutor) {
				deletedSong := &entity.Song{DisplayID: "S002", IsDeleted: true, Charts: []*entity.Chart{{ID: 201, DifficultyID: 4}}}
				songRepo.On("FindByDisplayID", ctx, exec, "S002").Return(deletedSong, nil).Once()
			},
			assertResult: func(t *testing.T, result *entity.SongChartStats) {
				assert.Nil(t, result)
			},
			expectedErrChecker: func(t assert.TestingT, err error, _ ...any) bool {
				return assert.ErrorIs(t, err, repository.ErrSongNotFound)
			},
		},
		{
			name: "EDITOR権限がある場合は削除済み楽曲を取得できる",
			accountTypeID: func() *int {
				editor := info.AccountTypeEditor
				return &editor
			}(),
			setupMocks: func(ctx context.Context, songRepo *MockSongRepository, statsRepo *MockChartStatsRepository, songMasterProvider *MockSongMasterProvider, exec *MockExecutor) {
				deletedSong := &entity.Song{DisplayID: "S002", IsDeleted: true, Charts: []*entity.Chart{{ID: 201, DifficultyID: 4}}}
				songRepo.On("FindByDisplayID", ctx, exec, "S002").Return(deletedSong, nil).Once()
				songMasterProvider.On("SongMasters").Return(&masterdata.SongMasters{CommonMasters: masterdata.CommonMasters{DifficultyNamesByID: map[int]string{4: "MASTER"}}}).Once()
				statsRepo.On("FindChartStatsByChartIDs", ctx, exec, []int{201}).Return([]*entity.ChartStatsByRatingBand{}, nil).Once()
			},
			assertResult: func(t *testing.T, result *entity.SongChartStats) {
				assert.NotNil(t, result)
			},
			expectedErrChecker: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSongRepo := new(MockSongRepository)
			mockWorldsendRepo := new(MockWorldsendChartRepository)
			mockStatsRepo := new(MockChartStatsRepository)
			mockSongMasterProvider := new(MockSongMasterProvider)
			mockExec := new(MockExecutor)
			stubMasterProvider := &StubChartStatsMasterProvider{}

			tt.setupMocks(ctx, mockSongRepo, mockStatsRepo, mockSongMasterProvider, mockExec)

			u := NewChartStatsUsecase(mockSongRepo, mockWorldsendRepo, mockStatsRepo, mockSongMasterProvider, stubMasterProvider, mockExec, mockExec)

			result, err := u.GetSongStatsByDisplayID(ctx, "S002", tt.accountTypeID)

			tt.expectedErrChecker(t, err)
			tt.assertResult(t, result)
		})
	}
}

func TestGetChartStatsByDisplayIDAndDifficulty_WorldsendBranch(t *testing.T) {
	ctx := context.Background()
	mockSongRepo := new(MockSongRepository)
	mockWorldsendRepo := new(MockWorldsendChartRepository)
	mockStatsRepo := new(MockChartStatsRepository)
	mockSongMasterProvider := new(MockSongMasterProvider)
	mockExec := new(MockExecutor)
	stubMasterProvider := &StubChartStatsMasterProvider{bands: []*entity.RatingBand{{ID: 1, SortOrder: 2}, {ID: 2, SortOrder: 1}}}

	u := NewChartStatsUsecase(mockSongRepo, mockWorldsendRepo, mockStatsRepo, mockSongMasterProvider, stubMasterProvider, mockExec, mockExec)

	worldsendSong := &entity.Song{DisplayID: "WE001", IsWorldsend: true}
	mockSongRepo.On("FindByDisplayID", ctx, mockExec, "WE001").Return(worldsendSong, nil)
	mockWorldsendRepo.On("FindByDisplayID", ctx, mockExec, "WE001").Return(&repository.WorldsendSongWithChart{Song: worldsendSong, Chart: &entity.WorldsendChart{ID: 301}}, nil)
	mockStatsRepo.On("FindChartStatsByChartIDs", ctx, mockExec, []int{301}).Return([]*entity.ChartStatsByRatingBand{{ChartID: 301, RatingBandID: 1}, {ChartID: 301, RatingBandID: 2}}, nil)

	result, err := u.GetChartStatsByDisplayIDAndDifficulty(ctx, "WE001", info.StatsDifficultyWorldsend, nil)

	assert.NoError(t, err)
	assert.Equal(t, info.StatsDifficultyWorldsend, result.Difficulty)
	assert.Equal(t, []int{2, 1}, []int{result.Stats[0].RatingBandID, result.Stats[1].RatingBandID})
}
