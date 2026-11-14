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
	uc := NewScoreHistoryUsecase(exec, userRepo, songRepo, nil, historyRepo, &stubFriendshipRepo{}, newScoreHistoryReadMasterProvider())

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
			uc := NewScoreHistoryUsecase(exec, userRepo, songRepo, nil, historyRepo, &stubFriendshipRepo{}, newScoreHistoryReadMasterProvider())

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
	uc := NewScoreHistoryUsecase(exec, userRepo, nil, worldsendRepo, historyRepo, &stubFriendshipRepo{}, newScoreHistoryReadMasterProvider())

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
			uc := NewScoreHistoryUsecase(exec, userRepo, nil, worldsendRepo, historyRepo, &stubFriendshipRepo{}, newScoreHistoryReadMasterProvider())

			got, err := uc.GetWorldsend(ctx, "testuser", nil, "fedcba9876543210")

			assert.Nil(t, got)
			require.ErrorIs(t, err, tt.wantErr)
			historyRepo.AssertNotCalled(t, "FindWorldsendTimeline", mock.Anything, mock.Anything, mock.Anything)
			userRepo.AssertExpectations(t)
			worldsendRepo.AssertExpectations(t)
		})
	}
}

func TestScoreHistoryUsecase_GetStandard_履歴を読む前の分岐ごとのエラー(t *testing.T) {
	tests := []struct {
		name string
		// Given: 指定する難易度、プレイヤー連携の有無、見つかる楽曲（nilは楽曲を検索しない段階で止まる）
		difficulty   string
		playerLinked bool
		lookupUser   bool
		song         *entity.Song
		// Then: 期待するエラー
		wantErr error
	}{
		{
			name:       "履歴を持たないBASICは利用者の確認より先に対象外として弾く",
			difficulty: "BASIC",
			wantErr:    ErrScoreHistoryUnsupportedDifficulty,
		},
		{
			name:       "プレイヤー未連携は楽曲を検索せず履歴なしとする",
			difficulty: "EXPERT",
			lookupUser: true,
			wantErr:    ErrScoreHistoryNotFound,
		},
		{
			// テスト用マスタはEXPERTだけを持つため、MASTERはマスタに無い難易度になる。
			name:         "マスタに無い難易度は不正とする",
			difficulty:   "MASTER",
			playerLinked: true,
			lookupUser:   true,
			song:         &entity.Song{ID: 20, Charts: []*entity.Chart{{ID: 30, DifficultyID: 3}}},
			wantErr:      ErrInvalidDifficulty,
		},
		{
			name:         "楽曲に指定難易度の譜面が無ければ譜面なしとする",
			difficulty:   "EXPERT",
			playerLinked: true,
			lookupUser:   true,
			song:         &entity.Song{ID: 20, Charts: []*entity.Chart{}},
			wantErr:      ErrChartNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			// 期待を設定していないモックの呼び出しはテストを失敗させるため、
			// 止まるはずの段階より後ろの読み出し（楽曲・履歴）が呼ばれないことも同時に確かめられる。
			ctx := context.Background()
			exec := &MockExecutor{}
			userRepo := new(MockUserRepository)
			songRepo := new(MockSongRepository)
			historyRepo := new(mockScoreHistoryRepository)
			if tt.lookupUser {
				user := testUser(t, 1, "testuser")
				if tt.playerLinked {
					user = newScoreHistoryReadUser(t)
				}
				userRepo.On("FindByUsername", mock.Anything, exec, "testuser").Return(user, nil).Once()
			}
			if tt.song != nil {
				songRepo.On("FindByDisplayID", ctx, exec, "0123456789abcdef").Return(tt.song, nil).Once()
			}
			uc := NewScoreHistoryUsecase(exec, userRepo, songRepo, nil, historyRepo, &stubFriendshipRepo{}, newScoreHistoryReadMasterProvider())

			// When
			got, err := uc.GetStandard(ctx, "testuser", nil, "0123456789abcdef", tt.difficulty)

			// Then
			assert.Nil(t, got)
			assert.ErrorIs(t, err, tt.wantErr)
			userRepo.AssertExpectations(t)
			songRepo.AssertExpectations(t)
		})
	}
}

func TestScoreHistoryUsecase_GetWorldsend_履歴が0件の場合は履歴なしエラーを返す(t *testing.T) {
	// Given: 履歴0件は空配列ではなく404相当のエラーになる。通常譜面と共通の変換のためWORLD'S END側だけで確かめる。
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
	setScoreHistoryReadUserExpectation(userRepo, exec, user)
	worldsendRepo.On("FindByDisplayID", ctx, exec, "fedcba9876543210").Return(song, nil).Once()
	historyRepo.On("FindWorldsendTimeline", ctx, 10, 40).Return([]entity.ScoreHistoryEntry{}, nil).Once()
	uc := NewScoreHistoryUsecase(exec, userRepo, nil, worldsendRepo, historyRepo, &stubFriendshipRepo{}, newScoreHistoryReadMasterProvider())

	// When
	got, err := uc.GetWorldsend(ctx, "testuser", nil, "fedcba9876543210")

	// Then
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrScoreHistoryNotFound)
	historyRepo.AssertExpectations(t)
}
