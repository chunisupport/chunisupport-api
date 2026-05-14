package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	domainservice "github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubPlayerLockedSongPlayerRepository struct {
	player    *entity.Player
	gotUserID int
	saved     *entity.Player
}

func (s *stubPlayerLockedSongPlayerRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	return nil, nil
}

func (s *stubPlayerLockedSongPlayerRepository) FindByIDWithHonors(ctx context.Context, exec repository.Executor, id int) (*repository.PlayerWithHonors, error) {
	return nil, nil
}

func (s *stubPlayerLockedSongPlayerRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	s.gotUserID = userID
	return s.player, nil
}

func (s *stubPlayerLockedSongPlayerRepository) FindHonorsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerHonor, error) {
	return nil, nil
}

func (s *stubPlayerLockedSongPlayerRepository) UpdateCalculatedRatings(ctx context.Context, exec repository.Executor, playerID int, calculatedRating, bestAverage, newAverage float64) error {
	return nil
}

func (s *stubPlayerLockedSongPlayerRepository) Save(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	s.saved = player
	return nil
}

func (s *stubPlayerLockedSongPlayerRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}

type spyPlayerLockedSongRepository struct {
	createCalled bool
	deleteCalled bool
	lockedSongs  []*entity.PlayerLockedSong
}

func (s *spyPlayerLockedSongRepository) ListByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerLockedSong, error) {
	return s.lockedSongs, nil
}

func (s *spyPlayerLockedSongRepository) Create(ctx context.Context, exec repository.Executor, lockedSong *entity.PlayerLockedSong) error {
	s.createCalled = true
	return nil
}

func (s *spyPlayerLockedSongRepository) BulkCreate(ctx context.Context, exec repository.Executor, lockedSongs []*entity.PlayerLockedSong) error {
	s.createCalled = true
	return nil
}

func (s *spyPlayerLockedSongRepository) Delete(ctx context.Context, exec repository.Executor, playerID int, songID int, isUltima bool) error {
	s.deleteCalled = true
	return nil
}

func (s *spyPlayerLockedSongRepository) BulkDelete(ctx context.Context, exec repository.Executor, playerID int, songIDs []int, isUltimaFlags []bool) error {
	s.deleteCalled = true
	return nil
}

type stubPlayerSongIDResolver struct {
	songID *int
}

func (s *stubPlayerSongIDResolver) ResolveSongIDByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*int, error) {
	return s.songID, nil
}

type stubPlayerRecordRepositoryForLockedSong struct {
	records []*entity.PlayerRecord
}

func (s *stubPlayerRecordRepositoryForLockedSong) FindByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerRecord, error) {
	return s.records, nil
}
func (s *stubPlayerRecordRepositoryForLockedSong) FindByPlayerIDForRating(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerRecord, error) {
	return nil, nil
}
func (s *stubPlayerRecordRepositoryForLockedSong) GetLastScoreUpdate(ctx context.Context, exec repository.Executor, playerID int) (*time.Time, error) {
	return nil, nil
}

type stubPlayerDataRepositoryForLockedSong struct{}

func (s *stubPlayerDataRepositoryForLockedSong) LoadMasterData(ctx context.Context, officialIdxList []string) (*repository.PlayerDataMaster, error) {
	return nil, nil
}
func (s *stubPlayerDataRepositoryForLockedSong) SavePlayerData(ctx context.Context, exec repository.Executor, input repository.PlayerDataSaveInput) error {
	return nil
}
func (s *stubPlayerDataRepositoryForLockedSong) GetOverpowerTargetStats(ctx context.Context, filter repository.OverpowerTargetFilter) (*repository.OverpowerTargetStats, error) {
	return &repository.OverpowerTargetStats{MaxOverpowerTotal: 100}, nil
}

type stubPlayerLockedSongQueryService struct {
	gotPlayerID int
	rows        []*PlayerLockedSongReadModel
}

func (s *stubPlayerLockedSongQueryService) ListWithSongDisplayIDAndTitleByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*PlayerLockedSongReadModel, error) {
	s.gotPlayerID = playerID
	return s.rows, nil
}

func TestPlayerLockedSongList(t *testing.T) {
	tests := []struct {
		name        string
		targetUser  *entity.User
		player      *entity.Player
		requester   *entity.User
		wantErr     error
		wantRowsHit bool
	}{
		{
			name:        "公開ユーザーの未解禁曲を取得できる",
			targetUser:  &entity.User{ID: 100},
			player:      &entity.Player{ID: 10},
			wantRowsHit: true,
		},
		{
			name:        "非公開ユーザー本人は未解禁曲を取得できる",
			targetUser:  &entity.User{ID: 100, IsPrivate: true},
			player:      &entity.Player{ID: 10},
			requester:   &entity.User{ID: 100},
			wantRowsHit: true,
		},
		{
			name:       "プレイヤー未連携ユーザーは未連携エラー",
			targetUser: &entity.User{ID: 100},
			wantErr:    ErrPlayerNotLinked,
		},
		{
			name:       "非公開ユーザーを他人が参照すると非公開エラー",
			targetUser: &entity.User{ID: 100, IsPrivate: true},
			requester:  &entity.User{ID: 200},
			wantErr:    ErrUserPrivate,
		},
		{
			name:    "存在しないユーザーは見つからないエラー",
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			userRepo := new(MockUserRepository)
			if tt.targetUser == nil {
				userRepo.On("FindByUsername", mock.Anything, mock.Anything, "testuser").Return(nil, repository.ErrUserNotFound).Once()
			} else {
				userRepo.On("FindByUsername", mock.Anything, mock.Anything, "testuser").Return(tt.targetUser, nil).Once()
			}
			queryService := &stubPlayerLockedSongQueryService{
				rows: []*PlayerLockedSongReadModel{
					{DisplayID: "0123456789abcdef", Title: "テスト楽曲", IsUltima: true},
				},
			}
			playerRepo := &stubPlayerLockedSongPlayerRepository{player: tt.player}
			u := &playerLockedSongUsecase{
				userRepo:     userRepo,
				playerRepo:   playerRepo,
				queryService: queryService,
			}

			// When
			got, err := u.List(context.Background(), "testuser", tt.requester)

			// Then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, "0123456789abcdef", got[0].DisplayID)
				assert.Equal(t, "テスト楽曲", got[0].Title)
				assert.True(t, got[0].IsUltima)
			}
			if tt.wantRowsHit {
				assert.Equal(t, 100, playerRepo.gotUserID)
				assert.Equal(t, 10, queryService.gotPlayerID)
			} else {
				assert.Zero(t, queryService.gotPlayerID)
			}
			userRepo.AssertExpectations(t)
		})
	}
}

func TestPlayerLockedSongLock(t *testing.T) {
	tests := []struct {
		name           string
		song           *entity.Song
		inputIsUltima  bool
		wantErr        error
		wantCreateCall bool
	}{
		{
			name:    "WORLD'S END楽曲は見つからない楽曲として扱う",
			song:    &entity.Song{ID: 1, DisplayID: "0123456789abcdef", IsWorldsend: true, Charts: []*entity.Chart{}},
			wantErr: repository.ErrSongNotFound,
		},
		{
			name:    "削除済み楽曲は見つからない楽曲として扱う",
			song:    &entity.Song{ID: 1, DisplayID: "0123456789abcdef", IsDeleted: true, Charts: []*entity.Chart{}},
			wantErr: repository.ErrSongNotFound,
		},
		{
			name:          "ULTIMA譜面がない楽曲をULTIMA未解禁にできない",
			song:          &entity.Song{ID: 1, DisplayID: "0123456789abcdef", Charts: []*entity.Chart{{DifficultyID: domainservice.DifficultyIDMaster}}},
			inputIsUltima: true,
			wantErr:       ErrChartNotFound,
		},
		{
			name:           "ULTIMA譜面がある楽曲をULTIMA未解禁にできる",
			song:           &entity.Song{ID: 1, DisplayID: "0123456789abcdef", Charts: []*entity.Chart{{DifficultyID: domainservice.DifficultyIDUltima}}},
			inputIsUltima:  true,
			wantCreateCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			displayID, err := displayid.NewDisplayID("0123456789abcdef")
			require.NoError(t, err)
			songRepo := new(MockSongRepository)
			songRepo.On("FindByDisplayID", mock.Anything, mock.Anything, "0123456789abcdef").Return(tt.song, nil).Once()
			lockedRepo := &spyPlayerLockedSongRepository{}
			u := &playerLockedSongUsecase{
				tm:             &passthroughTransactionManager{},
				playerRepo:     &stubPlayerLockedSongPlayerRepository{player: &entity.Player{ID: 10}},
				playerRecRepo:  &stubPlayerRecordRepositoryForLockedSong{records: []*entity.PlayerRecord{}},
				playerDataRepo: &stubPlayerDataRepositoryForLockedSong{},
				songRepo:       songRepo,
				lockedRepo:     lockedRepo,
			}

			// When
			err = u.Lock(context.Background(), 100, &PlayerLockedSongInput{DisplayID: displayID, IsUltima: tt.inputIsUltima})

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantCreateCall, lockedRepo.createCalled)
			songRepo.AssertExpectations(t)
		})
	}
}

func TestPlayerLockedSongInputRequired(t *testing.T) {
	tests := []struct {
		name string
		run  func(*playerLockedSongUsecase) error
	}{
		{
			name: "ロック入力がnilの場合はエラー",
			run: func(u *playerLockedSongUsecase) error {
				return u.Lock(context.Background(), 100, nil)
			},
		},
		{
			name: "ロック解除入力がnilの場合はエラー",
			run: func(u *playerLockedSongUsecase) error {
				return u.Unlock(context.Background(), 100, nil)
			},
		},
		{
			name: "バッチ入力がnilの場合はエラー",
			run: func(u *playerLockedSongUsecase) error {
				return u.Batch(context.Background(), 100, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			u := &playerLockedSongUsecase{}

			// When
			err := tt.run(u)

			// Then
			require.ErrorIs(t, err, errPlayerLockedSongInputRequired)
			assert.EqualError(t, err, "input is required")
		})
	}
}

func TestPlayerLockedSongBatch(t *testing.T) {
	displayID1, err := displayid.NewDisplayID("0123456789abcdef")
	require.NoError(t, err)
	displayID2, err := displayid.NewDisplayID("fedcba9876543210")
	require.NoError(t, err)

	lockedRepo := &spyPlayerLockedSongRepository{}
	songRepo := new(MockSongRepository)
	songRepo.On("FindByDisplayID", mock.Anything, mock.Anything, "0123456789abcdef").Return(&entity.Song{ID: 1, DisplayID: "0123456789abcdef", Charts: []*entity.Chart{}}, nil).Once()

	u := &playerLockedSongUsecase{
		db:             nil,
		tm:             &passthroughTransactionManager{},
		playerRepo:     &stubPlayerLockedSongPlayerRepository{player: &entity.Player{ID: 10}},
		playerRecRepo:  &stubPlayerRecordRepositoryForLockedSong{records: []*entity.PlayerRecord{}},
		playerDataRepo: &stubPlayerDataRepositoryForLockedSong{},
		songRepo:       songRepo,
		lockedRepo:     lockedRepo,
		resolver:       &stubPlayerSongIDResolver{songID: ptrInt(1)},
	}

	err = u.Batch(context.Background(), 100, &PlayerLockedSongBatchInput{
		Add:    []*PlayerLockedSongInput{{DisplayID: displayID1, IsUltima: false}},
		Delete: []*PlayerLockedSongInput{{DisplayID: displayID2, IsUltima: true}},
	})
	require.NoError(t, err)
	assert.True(t, lockedRepo.createCalled)
	assert.True(t, lockedRepo.deleteCalled)
	songRepo.AssertExpectations(t)
}

func ptrInt(v int) *int { return &v }
