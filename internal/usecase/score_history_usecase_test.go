package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockScoreHistoryRepository struct {
	mock.Mock
}

func (m *mockScoreHistoryRepository) BulkInsertStandard(ctx context.Context, exec repository.Executor, rows []repository.PlayerRecordHistory) error {
	args := m.Called(ctx, exec, rows)
	return args.Error(0)
}

func (m *mockScoreHistoryRepository) BulkInsertWorldsend(ctx context.Context, exec repository.Executor, rows []repository.PlayerWorldsendRecordHistory) error {
	args := m.Called(ctx, exec, rows)
	return args.Error(0)
}

func (m *mockScoreHistoryRepository) PruneStandardOverLimit(ctx context.Context, exec repository.Executor, playerID int, chartIDs []int) error {
	args := m.Called(ctx, exec, playerID, chartIDs)
	return args.Error(0)
}

func (m *mockScoreHistoryRepository) PruneWorldsendOverLimit(ctx context.Context, exec repository.Executor, playerID int, chartIDs []int) error {
	args := m.Called(ctx, exec, playerID, chartIDs)
	return args.Error(0)
}

func (m *mockScoreHistoryRepository) FindStandardTimeline(ctx context.Context, playerID, chartID int) ([]entity.ScoreHistoryEntry, error) {
	args := m.Called(ctx, playerID, chartID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.ScoreHistoryEntry), args.Error(1)
}

func (m *mockScoreHistoryRepository) FindWorldsendTimeline(ctx context.Context, playerID, worldsendChartID int) ([]entity.ScoreHistoryEntry, error) {
	args := m.Called(ctx, playerID, worldsendChartID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.ScoreHistoryEntry), args.Error(1)
}

type scoreHistoryReadMasterProvider struct {
	masters *domainmasterdata.PlayerDataMasters
}

func (p *scoreHistoryReadMasterProvider) PlayerDataMasters() *domainmasterdata.PlayerDataMasters {
	return p.masters
}

func newScoreHistoryReadMasterProvider() repository.PlayerDataMasterProvider {
	return &scoreHistoryReadMasterProvider{
		masters: &domainmasterdata.PlayerDataMasters{
			Difficulties: map[string]master.ChartDifficulty{
				"EXPERT": {ID: 3},
			},
		},
	}
}

func newScoreHistoryReadUser(t *testing.T) *entity.User {
	t.Helper()
	playerID := 10
	user := testUser(t, 1, "testuser")
	user.PlayerID = &playerID
	return user
}

func setScoreHistoryReadUserExpectation(userRepo *MockUserRepository, exec repository.Executor, user *entity.User) {
	userRepo.On("FindByUsername", mock.Anything, exec, "testuser").Return(user, nil).Once()
}

func TestScoreHistoryUsecase_GetStandard_公開中の通常楽曲だけ履歴を取得する(t *testing.T) {
	ctx := context.Background()
	exec := &MockExecutor{}
	userRepo := new(MockUserRepository)
	songRepo := new(MockSongRepository)
	historyRepo := new(mockScoreHistoryRepository)
	user := newScoreHistoryReadUser(t)
	song := &entity.Song{
		ID:          20,
		IsWorldsend: false,
		Charts:      []*entity.Chart{{ID: 30, DifficultyID: 3}},
	}
	wantRows := []entity.ScoreHistoryEntry{{Score: 1_000_000, UpdatedAt: time.Now().UTC()}}

	setScoreHistoryReadUserExpectation(userRepo, exec, user)
	songRepo.On("FindByDisplayID", ctx, exec, "0123456789abcdef").Return(song, nil).Once()
	historyRepo.On("FindStandardTimeline", ctx, 10, 30).Return(wantRows, nil).Once()
	uc := NewScoreHistoryUsecase(exec, userRepo, songRepo, nil, historyRepo, newScoreHistoryReadMasterProvider())

	got, err := uc.GetStandard(ctx, "testuser", nil, "0123456789abcdef", "EXPERT")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, wantRows[0].Score, got[0].Score)
	userRepo.AssertExpectations(t)
	songRepo.AssertExpectations(t)
	historyRepo.AssertExpectations(t)
}

func TestScoreHistoryUsecase_GetStandard_公開対象外の楽曲は404へ正規化する(t *testing.T) {
	tests := []struct {
		name string
		song *entity.Song
	}{
		{
			name: "削除済み通常楽曲",
			song: &entity.Song{IsDeleted: true},
		},
		{
			name: "WORLD'S END楽曲",
			song: &entity.Song{IsWorldsend: true},
		},
		{
			name: "楽曲がnil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			exec := &MockExecutor{}
			userRepo := new(MockUserRepository)
			songRepo := new(MockSongRepository)
			historyRepo := new(mockScoreHistoryRepository)
			user := newScoreHistoryReadUser(t)

			setScoreHistoryReadUserExpectation(userRepo, exec, user)
			songRepo.On("FindByDisplayID", ctx, exec, "0123456789abcdef").Return(tt.song, nil).Once()
			uc := NewScoreHistoryUsecase(exec, userRepo, songRepo, nil, historyRepo, newScoreHistoryReadMasterProvider())

			got, err := uc.GetStandard(ctx, "testuser", nil, "0123456789abcdef", "EXPERT")

			assert.Nil(t, got)
			require.ErrorIs(t, err, repository.ErrSongNotFound)
			historyRepo.AssertNotCalled(t, "FindStandardTimeline", mock.Anything, mock.Anything, mock.Anything)
			userRepo.AssertExpectations(t)
			songRepo.AssertExpectations(t)
		})
	}
}

func TestScoreHistoryUsecase_GetWorldsend_公開中のWORLDsend楽曲だけ履歴を取得する(t *testing.T) {
	ctx := context.Background()
	exec := &MockExecutor{}
	userRepo := new(MockUserRepository)
	worldsendRepo := new(MockWorldsendChartRepository)
	historyRepo := new(mockScoreHistoryRepository)
	user := newScoreHistoryReadUser(t)
	song := &entity.WorldsendSongWithChart{
		Song:  &entity.Song{ID: 20, IsWorldsend: true},
		Chart: &entity.WorldsendChart{ID: 40},
	}
	wantRows := []entity.ScoreHistoryEntry{{Score: 990_000, UpdatedAt: time.Now().UTC()}}

	setScoreHistoryReadUserExpectation(userRepo, exec, user)
	worldsendRepo.On("FindByDisplayID", ctx, exec, "fedcba9876543210").Return(song, nil).Once()
	historyRepo.On("FindWorldsendTimeline", ctx, 10, 40).Return(wantRows, nil).Once()
	uc := NewScoreHistoryUsecase(exec, userRepo, nil, worldsendRepo, historyRepo, newScoreHistoryReadMasterProvider())

	got, err := uc.GetWorldsend(ctx, "testuser", nil, "fedcba9876543210")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, wantRows[0].Score, got[0].Score)
	userRepo.AssertExpectations(t)
	worldsendRepo.AssertExpectations(t)
	historyRepo.AssertExpectations(t)
}

func TestScoreHistoryUsecase_GetWorldsend_公開対象外の楽曲は404へ正規化する(t *testing.T) {
	tests := []struct {
		name      string
		songChart *entity.WorldsendSongWithChart
		wantErr   error
	}{
		{
			name: "削除済みWORLD'S END楽曲",
			songChart: &entity.WorldsendSongWithChart{
				Song:  &entity.Song{IsWorldsend: true, IsDeleted: true},
				Chart: &entity.WorldsendChart{ID: 40},
			},
			wantErr: repository.ErrSongNotFound,
		},
		{
			name: "通常楽曲",
			songChart: &entity.WorldsendSongWithChart{
				Song:  &entity.Song{IsWorldsend: false},
				Chart: &entity.WorldsendChart{ID: 40},
			},
			wantErr: repository.ErrSongNotFound,
		},
		{
			name: "楽曲がnil",
			songChart: &entity.WorldsendSongWithChart{
				Chart: &entity.WorldsendChart{ID: 40},
			},
			wantErr: repository.ErrSongNotFound,
		},
		{
			name: "譜面がnil",
			songChart: &entity.WorldsendSongWithChart{
				Song: &entity.Song{IsWorldsend: true},
			},
			wantErr: ErrChartNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			exec := &MockExecutor{}
			userRepo := new(MockUserRepository)
			worldsendRepo := new(MockWorldsendChartRepository)
			historyRepo := new(mockScoreHistoryRepository)
			user := newScoreHistoryReadUser(t)

			setScoreHistoryReadUserExpectation(userRepo, exec, user)
			worldsendRepo.On("FindByDisplayID", ctx, exec, "fedcba9876543210").Return(tt.songChart, nil).Once()
			uc := NewScoreHistoryUsecase(exec, userRepo, nil, worldsendRepo, historyRepo, newScoreHistoryReadMasterProvider())

			got, err := uc.GetWorldsend(ctx, "testuser", nil, "fedcba9876543210")

			assert.Nil(t, got)
			require.ErrorIs(t, err, tt.wantErr)
			historyRepo.AssertNotCalled(t, "FindWorldsendTimeline", mock.Anything, mock.Anything, mock.Anything)
			userRepo.AssertExpectations(t)
			worldsendRepo.AssertExpectations(t)
		})
	}
}
