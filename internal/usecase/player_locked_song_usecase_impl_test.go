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
	player              *entity.Player
	lockedPlayer        *entity.Player
	gotUserID           int
	findCalls           int
	findForUpdateCalls  int
	findForUpdateExec   repository.Executor
	saved               *entity.Player
	saveExec            repository.Executor
	mutationCallHistory *[]string
}

func (s *stubPlayerLockedSongPlayerRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	return nil, nil
}

func (s *stubPlayerLockedSongPlayerRepository) FindByIDWithHonors(ctx context.Context, exec repository.Executor, id int) (*repository.PlayerWithHonors, error) {
	return nil, nil
}

func (s *stubPlayerLockedSongPlayerRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	s.findCalls++
	s.gotUserID = userID
	return s.player, nil
}

func (s *stubPlayerLockedSongPlayerRepository) FindByUserIDForUpdate(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	s.findForUpdateCalls++
	s.findForUpdateExec = exec
	s.gotUserID = userID
	if s.mutationCallHistory != nil {
		*s.mutationCallHistory = append(*s.mutationCallHistory, "player_lock")
	}
	if s.lockedPlayer != nil {
		return s.lockedPlayer, nil
	}
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
	s.saveExec = exec
	if s.mutationCallHistory != nil {
		*s.mutationCallHistory = append(*s.mutationCallHistory, "player_save")
	}
	return nil
}

func (s *stubPlayerLockedSongPlayerRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}

type spyPlayerLockedSongRepository struct {
	createCalled     bool
	deleteCalled     bool
	bulkCreateCalled bool
	bulkDeleteCalled bool
	lockedSongs      []*entity.PlayerLockedSong
	callHistory      *[]string
	execs            []repository.Executor
}

func (s *spyPlayerLockedSongRepository) ListByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerLockedSong, error) {
	s.execs = append(s.execs, exec)
	return s.lockedSongs, nil
}

func (s *spyPlayerLockedSongRepository) Create(ctx context.Context, exec repository.Executor, lockedSong *entity.PlayerLockedSong) error {
	s.createCalled = true
	s.execs = append(s.execs, exec)
	if s.callHistory != nil {
		*s.callHistory = append(*s.callHistory, "child_create")
	}
	return nil
}

func (s *spyPlayerLockedSongRepository) BulkCreate(ctx context.Context, exec repository.Executor, lockedSongs []*entity.PlayerLockedSong) error {
	s.bulkCreateCalled = true
	s.execs = append(s.execs, exec)
	if s.callHistory != nil {
		*s.callHistory = append(*s.callHistory, "child_bulk_create")
	}
	return nil
}

func (s *spyPlayerLockedSongRepository) Delete(ctx context.Context, exec repository.Executor, playerID int, songID int, isUltima bool) error {
	s.deleteCalled = true
	s.execs = append(s.execs, exec)
	if s.callHistory != nil {
		*s.callHistory = append(*s.callHistory, "child_delete")
	}
	return nil
}

func (s *spyPlayerLockedSongRepository) BulkDelete(ctx context.Context, exec repository.Executor, playerID int, songIDs []int, isUltimaFlags []bool) error {
	s.bulkDeleteCalled = true
	s.execs = append(s.execs, exec)
	if s.callHistory != nil {
		*s.callHistory = append(*s.callHistory, "child_bulk_delete")
	}
	return nil
}

func (s *spyPlayerLockedSongRepository) DeleteBySongID(ctx context.Context, exec repository.Executor, songID int) error {
	return nil
}

type stubPlayerSongIDResolver struct {
	songID *int
}

func (s *stubPlayerSongIDResolver) ResolveSongIDByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*int, error) {
	return s.songID, nil
}

func (s *stubPlayerSongIDResolver) ResolveSongIDsByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) (map[string]int, error) {
	resolved := make(map[string]int, len(displayIDs))
	if s.songID == nil {
		return resolved, nil
	}
	for _, displayID := range displayIDs {
		resolved[displayID] = *s.songID
	}
	return resolved, nil
}

type stubPlayerRecordRepositoryForLockedSong struct {
	records []*entity.PlayerRecord
	exec    repository.Executor
}

func (s *stubPlayerRecordRepositoryForLockedSong) FindByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerRecord, error) {
	s.exec = exec
	return s.records, nil
}
func (s *stubPlayerRecordRepositoryForLockedSong) FindByPlayerIDAndSongDisplayID(ctx context.Context, exec repository.Executor, playerID int, displayID string) ([]*entity.PlayerRecord, error) {
	return s.records, nil
}
func (s *stubPlayerRecordRepositoryForLockedSong) FindByPlayerIDForRating(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerRecord, error) {
	return nil, nil
}
func (s *stubPlayerRecordRepositoryForLockedSong) FindOPTargetCandidatesByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]repository.PlayerRecordOPTargetCandidate, error) {
	return nil, nil
}
func (s *stubPlayerRecordRepositoryForLockedSong) GetLastScoreUpdate(ctx context.Context, exec repository.Executor, playerID int) (*time.Time, error) {
	return nil, nil
}

type stubPlayerDataRepositoryForLockedSong struct {
	receivedFilter repository.OverpowerTargetFilter
	receivedExec   repository.Executor
}

func (s *stubPlayerDataRepositoryForLockedSong) LoadMasterData(ctx context.Context, officialIdxList []string) (*repository.PlayerDataMaster, error) {
	return nil, nil
}
func (s *stubPlayerDataRepositoryForLockedSong) SavePlayerData(ctx context.Context, exec repository.Executor, input repository.PlayerDataSaveInput) error {
	return nil
}

func (s *stubPlayerDataRepositoryForLockedSong) FindPlayerRecordStatesByChartIDs(ctx context.Context, exec repository.Executor, playerID int, chartIDs []int) (map[int]repository.PlayerRecordState, error) {
	return map[int]repository.PlayerRecordState{}, nil
}

func (s *stubPlayerDataRepositoryForLockedSong) FindWorldsendRecordStatesByChartIDs(ctx context.Context, exec repository.Executor, playerID int, worldsendChartIDs []int) (map[int]repository.WorldsendRecordState, error) {
	return map[int]repository.WorldsendRecordState{}, nil
}
func (s *stubPlayerDataRepositoryForLockedSong) GetOverpowerTargetStats(ctx context.Context, filter repository.OverpowerTargetFilter) (*repository.OverpowerTargetStats, error) {
	s.receivedFilter = filter
	return &repository.OverpowerTargetStats{MaxOverpowerTotal: 100}, nil
}
func (s *stubPlayerDataRepositoryForLockedSong) GetOverpowerTargetStatsWithExecutor(ctx context.Context, exec repository.Executor, filter repository.OverpowerTargetFilter) (*repository.OverpowerTargetStats, error) {
	s.receivedExec = exec
	return s.GetOverpowerTargetStats(ctx, filter)
}

func (s *stubPlayerDataRepositoryForLockedSong) SaveLatestUpdate(_ context.Context, _ repository.Executor, _ *entity.PlayerLatestUpdate) error {
	return nil
}

func (s *stubPlayerDataRepositoryForLockedSong) FindLatestUpdateByPlayerID(_ context.Context, _ int) (*entity.PlayerLatestUpdate, error) {
	return nil, repository.ErrPlayerLatestUpdateNotFound
}

func (s *stubPlayerDataRepositoryForLockedSong) FindLatestUpdateByPlayerIDForUpdate(_ context.Context, _ repository.Executor, _ int) (*entity.PlayerLatestUpdate, error) {
	return nil, repository.ErrPlayerLatestUpdateNotFound
}

type stubPlayerLockedSongQueryService struct {
	gotPlayerID int
	rows        []*PlayerLockedSongReadModel
}

type trackingPlayerLockedSongTransactionManager struct {
	tx          repository.Executor
	callHistory *[]string
}

func (m *trackingPlayerLockedSongTransactionManager) Transactional(ctx context.Context, f func(repository.Executor) error) error {
	*m.callHistory = append(*m.callHistory, "transaction_enter")
	return f(m.tx)
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
		friend      bool
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
			name:        "非公開ユーザーを承認済みフレンドが取得できる",
			targetUser:  &entity.User{ID: 100, IsPrivate: true},
			player:      &entity.Player{ID: 10},
			requester:   &entity.User{ID: 200},
			friend:      true,
			wantRowsHit: true,
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
			friendshipRepo := newStubFriendshipRepo()
			if tt.friend {
				friendshipRepo.exists[[2]int{tt.requester.ID, tt.targetUser.ID}] = true
			}
			u := &playerLockedSongUsecase{
				userRepo:       userRepo,
				playerRepo:     playerRepo,
				friendshipRepo: friendshipRepo,
				queryService:   queryService,
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
			playerDataRepo := &stubPlayerDataRepositoryForLockedSong{}
			u := &playerLockedSongUsecase{
				tm:             &passthroughTransactionManager{},
				playerRepo:     &stubPlayerLockedSongPlayerRepository{player: &entity.Player{ID: 10}},
				playerRecRepo:  &stubPlayerRecordRepositoryForLockedSong{records: []*entity.PlayerRecord{}},
				playerDataRepo: playerDataRepo,
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
			if tt.wantCreateCall {
				assert.Equal(t, repository.OverpowerTargetFilter{ExcludeWorldsend: true, ExcludeDeleted: true, PlayerID: ptrInt(10)}, playerDataRepo.receivedFilter)
			}
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
	// Given
	displayID1, err := displayid.NewDisplayID("0123456789abcdef")
	require.NoError(t, err)
	displayID2, err := displayid.NewDisplayID("fedcba9876543210")
	require.NoError(t, err)

	lockedRepo := &spyPlayerLockedSongRepository{}
	songRepo := new(MockSongRepository)
	songRepo.On("FindByDisplayIDs", mock.Anything, mock.Anything, []string{"0123456789abcdef"}).Return([]*entity.Song{{ID: 1, DisplayID: "0123456789abcdef", Charts: []*entity.Chart{}}}, nil).Once()

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

	// When
	err = u.Batch(context.Background(), 100, &PlayerLockedSongBatchInput{
		Add:    []*PlayerLockedSongInput{{DisplayID: displayID1, IsUltima: false}},
		Delete: []*PlayerLockedSongInput{{DisplayID: displayID2, IsUltima: true}},
	})
	// Then
	require.NoError(t, err)
	assert.True(t, lockedRepo.bulkCreateCalled)
	assert.True(t, lockedRepo.bulkDeleteCalled)
	assert.False(t, lockedRepo.createCalled)
	assert.False(t, lockedRepo.deleteCalled)
	songRepo.AssertExpectations(t)
}

func TestPlayerLockedSongMutationsLockPlayerBeforeChildRows(t *testing.T) {
	displayID, err := displayid.NewDisplayID("0123456789abcdef")
	require.NoError(t, err)

	tests := []struct {
		name        string
		wantHistory []string
		run         func(*playerLockedSongUsecase) error
	}{
		{
			name:        "ロックではPlayerを先にロックする",
			wantHistory: []string{"transaction_enter", "player_lock", "child_create", "player_save"},
			run: func(u *playerLockedSongUsecase) error {
				return u.Lock(context.Background(), 100, &PlayerLockedSongInput{DisplayID: displayID})
			},
		},
		{
			name:        "ロック解除ではPlayerを先にロックする",
			wantHistory: []string{"transaction_enter", "player_lock", "child_delete", "player_save"},
			run: func(u *playerLockedSongUsecase) error {
				return u.Unlock(context.Background(), 100, &PlayerLockedSongInput{DisplayID: displayID})
			},
		},
		{
			name:        "バッチ更新ではPlayerを先にロックする",
			wantHistory: []string{"transaction_enter", "player_lock", "child_bulk_create", "child_bulk_delete", "player_save"},
			run: func(u *playerLockedSongUsecase) error {
				return u.Batch(context.Background(), 100, &PlayerLockedSongBatchInput{
					Add:    []*PlayerLockedSongInput{{DisplayID: displayID}},
					Delete: []*PlayerLockedSongInput{{DisplayID: displayID, IsUltima: true}},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			callHistory := make([]string, 0, len(tt.wantHistory))
			tx := new(MockExecutor)
			lockedPlayer := &entity.Player{ID: 10, UserID: 100, Level: 99}
			playerRepo := &stubPlayerLockedSongPlayerRepository{
				player:              &entity.Player{ID: 10, UserID: 100, Level: 1},
				lockedPlayer:        lockedPlayer,
				mutationCallHistory: &callHistory,
			}
			lockedRepo := &spyPlayerLockedSongRepository{callHistory: &callHistory}
			playerRecRepo := &stubPlayerRecordRepositoryForLockedSong{records: []*entity.PlayerRecord{}}
			playerDataRepo := &stubPlayerDataRepositoryForLockedSong{}
			songRepo := new(MockSongRepository)
			songRepo.On("FindByDisplayID", mock.Anything, mock.Anything, displayID.String()).
				Return(&entity.Song{ID: 1, DisplayID: displayID.String(), Charts: []*entity.Chart{}}, nil).
				Maybe()
			songRepo.On("FindByDisplayIDs", mock.Anything, mock.Anything, []string{displayID.String()}).
				Return([]*entity.Song{{ID: 1, DisplayID: displayID.String(), Charts: []*entity.Chart{}}}, nil).
				Maybe()
			u := &playerLockedSongUsecase{
				db:             new(MockExecutor),
				tm:             &trackingPlayerLockedSongTransactionManager{tx: tx, callHistory: &callHistory},
				playerRepo:     playerRepo,
				playerRecRepo:  playerRecRepo,
				playerDataRepo: playerDataRepo,
				songRepo:       songRepo,
				lockedRepo:     lockedRepo,
				resolver:       &stubPlayerSongIDResolver{songID: ptrInt(1)},
			}

			// When
			err := tt.run(u)

			// Then
			require.NoError(t, err)
			assert.Zero(t, playerRepo.findCalls)
			assert.Equal(t, 1, playerRepo.findForUpdateCalls)
			assert.Same(t, lockedPlayer, playerRepo.saved)
			assert.Same(t, tx, playerRepo.findForUpdateExec)
			assert.Same(t, tx, playerRepo.saveExec)
			assert.Same(t, tx, playerRecRepo.exec)
			assert.Same(t, tx, playerDataRepo.receivedExec)
			for _, exec := range lockedRepo.execs {
				assert.Same(t, tx, exec)
			}
			assert.Equal(t, tt.wantHistory, callHistory)
		})
	}
}

func ptrInt(v int) *int { return &v }
