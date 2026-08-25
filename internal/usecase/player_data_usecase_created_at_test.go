package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubPlayerRepositoryForPlayerData struct {
	foundPlayer *entity.Player
	savedPlayer *entity.Player
	saveCalls   int
}

type passthroughTransactionManagerForPlayerDataIdentity struct {
	exec repository.Executor
	err  error
}

func (m *passthroughTransactionManagerForPlayerDataIdentity) Transactional(_ context.Context, fn func(repository.Executor) error) error {
	m.err = fn(m.exec)
	return m.err
}

type stubPlayerDataMasterProviderForIdentity struct{}

func (stubPlayerDataMasterProviderForIdentity) PlayerDataMasters() *masterdata.PlayerDataMasters {
	return &masterdata.PlayerDataMasters{}
}

func (s *stubPlayerRepositoryForPlayerData) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	return nil, nil
}

func TestEnsurePlayer_公式指標が変化した場合は更新前の組を履歴へ保存する(t *testing.T) {
	collectedAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	playerRepo := &stubPlayerRepositoryForPlayerData{foundPlayer: &entity.Player{
		ID: 10, UserID: 1, Name: playername.MustNewPlayerName("変更前"), Level: 40,
		OfficialRating: 17.24, OfficialOverpower: 12340.12,
		DataCollectedAt: &collectedAt, CreatedAt: collectedAt.Add(-24 * time.Hour), UpdatedAt: collectedAt,
	}}
	userRepo := new(MockUserRepository)
	uc := &playerDataUsecase{playerRepo: playerRepo, userRepo: userRepo}
	playerID := 10
	user := &entity.User{ID: 1, Username: username.MustNewUserName("playerdatatest"), PlayerID: &playerID}
	updatedAt := collectedAt.Add(time.Hour)
	userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	_, _, err := uc.ensurePlayer(context.Background(), nil, user, &PlayerDataSummaryInput{
		Name: "変更後", Level: 41, OfficialRating: 17.25, OfficialOverpower: 12345.67,
	}, updatedAt)

	require.NoError(t, err)
	require.NotNil(t, playerRepo.savedPlayer.PendingMetricHistory())
	assert.Equal(t, 17.24, playerRepo.savedPlayer.PendingMetricHistory().OfficialRating)
	assert.Equal(t, 12340.12, playerRepo.savedPlayer.PendingMetricHistory().OfficialOverpower)
	assert.Equal(t, collectedAt, playerRepo.savedPlayer.PendingMetricHistory().DataCollectedAt)
}

func TestEnsurePlayer_取得日時のない既存値は履歴へ保存しない(t *testing.T) {
	createdAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	playerRepo := &stubPlayerRepositoryForPlayerData{foundPlayer: &entity.Player{
		ID: 10, UserID: 1, Name: playername.MustNewPlayerName("変更前"), Level: 1,
		OfficialRating: 0, OfficialOverpower: 0, CreatedAt: createdAt, UpdatedAt: createdAt,
	}}
	userRepo := new(MockUserRepository)
	uc := &playerDataUsecase{playerRepo: playerRepo, userRepo: userRepo}
	playerID := 10
	user := &entity.User{ID: 1, Username: username.MustNewUserName("playerdatatest"), PlayerID: &playerID}
	userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	_, _, err := uc.ensurePlayer(context.Background(), nil, user, &PlayerDataSummaryInput{
		Name: "変更後", Level: 1, OfficialRating: 17.25, OfficialOverpower: 12345.67,
	}, createdAt.Add(time.Hour))

	require.NoError(t, err)
	assert.Nil(t, playerRepo.savedPlayer.PendingMetricHistory())
}

func (s *stubPlayerRepositoryForPlayerData) FindByIDWithHonors(ctx context.Context, exec repository.Executor, id int) (*repository.PlayerWithHonors, error) {
	return nil, nil
}

func (s *stubPlayerRepositoryForPlayerData) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	return s.foundPlayer, nil
}

func (s *stubPlayerRepositoryForPlayerData) FindByUserIDForUpdate(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	return s.foundPlayer, nil
}

func (s *stubPlayerRepositoryForPlayerData) FindHonorsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerHonor, error) {
	return nil, nil
}

func (s *stubPlayerRepositoryForPlayerData) UpdateCalculatedRatings(ctx context.Context, exec repository.Executor, playerID int, calculatedRating, bestAverage, newAverage float64) error {
	return nil
}

func (s *stubPlayerRepositoryForPlayerData) Save(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	s.saveCalls++
	copied := *player
	if copied.ID == 0 {
		copied.ID = 99
		player.ID = copied.ID
	}
	s.savedPlayer = &copied
	return nil
}

func TestRegister_同一取得日時の異なる本文は後続保存前に競合として拒否する(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, 8, 25, 1, 2, 3, 0, time.UTC)
	officialOverpowerPercent := 98.76
	playerID := 10
	playerRepo := &stubPlayerRepositoryForPlayerData{foundPlayer: &entity.Player{
		ID: 10, UserID: 1, Name: playername.MustNewPlayerName("登録済み"), Level: 50,
		OfficialRating: 17.25, OfficialOverpower: 12345.67, OfficialOverpowerPercent: &officialOverpowerPercent,
		DataCollectedAt: &updatedAt, CreatedAt: updatedAt.Add(-24 * time.Hour), UpdatedAt: updatedAt,
	}}
	latestUpdate, err := entity.NewPlayerLatestUpdate(10, 1, []byte("gzip-payload"), updatedAt, updatedAt.Add(time.Minute), "saved-hash")
	require.NoError(t, err)
	playerDataRepo := &stubPlayerDataRepositoryForApplyScoresTest{latestUpdate: latestUpdate}
	txExecutor := &sqlx.DB{}
	tm := &passthroughTransactionManagerForPlayerDataIdentity{exec: txExecutor}
	uc := &playerDataUsecase{
		tm: tm, userRepo: new(MockUserRepository), playerRepo: playerRepo,
		playerDataRepo: playerDataRepo, masterCache: stubPlayerDataMasterProviderForIdentity{},
	}
	user := &entity.User{ID: 1, Username: username.MustNewUserName("playerdatatest"), PlayerID: &playerID}
	rating := 17.25
	overpower := 12345.67
	payload := &PlayerDataPayload{
		AppVersion: "0.1.0", Name: "登録済み", Level: 50, Rating: &rating,
		Overpower: PlayerDataOverpowerPayload{Value: &overpower, Percentage: &officialOverpowerPercent},
		UpdatedAt: updatedAt.Format(time.RFC3339),
	}

	// When
	_, err = uc.Register(context.Background(), user, payload, "different-hash")

	// Then
	var conflictErr *PlayerDataConflictError
	assert.ErrorAs(t, err, &conflictErr)
	assert.Same(t, txExecutor, playerDataRepo.latestUpdateExec)
	assert.Equal(t, 1, playerDataRepo.findLatestUpdateForUpdateCalls)
	assert.Equal(t, 1, playerRepo.saveCalls)
	assert.Equal(t, 0, playerDataRepo.saveCalls)
	assert.Equal(t, 0, playerDataRepo.latestUpdateSaveCalls)
	assert.Same(t, err, tm.err)
}

func (s *stubPlayerRepositoryForPlayerData) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}

func TestEnsurePlayer_新規プレイヤー作成時はCreatedAtをゼロ値にしない(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "新規登録時はupdated_atをcreated_atにも設定して保存する",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			playerRepo := &stubPlayerRepositoryForPlayerData{}
			userRepo := new(MockUserRepository)
			uc := &playerDataUsecase{
				playerRepo: playerRepo,
				userRepo:   userRepo,
			}

			user := &entity.User{
				ID:       1,
				Username: username.MustNewUserName("playerdatatest"),
			}
			updatedAt := time.Date(2026, 3, 16, 15, 28, 53, 0, time.FixedZone("JST", 9*60*60))
			before := time.Now()
			officialRating := 16.25
			playerName := playername.MustNewPlayerName("テストプレイヤー")
			summary := &PlayerDataSummaryInput{
				Name:           playerName.String(),
				Level:          42,
				OfficialRating: officialRating,
			}

			userRepo.
				On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(savedUser *entity.User) bool {
					return savedUser != nil && savedUser.PlayerID != nil && *savedUser.PlayerID == 99
				})).
				Return(nil).
				Once()

			// When
			playerID, previousPlayer, err := uc.ensurePlayer(context.Background(), nil, user, summary, updatedAt)
			after := time.Now()

			// Then
			require.NoError(t, err)
			assert.Equal(t, 99, playerID)
			assert.Nil(t, previousPlayer)
			require.NotNil(t, playerRepo.savedPlayer)
			assert.False(t, playerRepo.savedPlayer.CreatedAt.IsZero())
			assert.False(t, playerRepo.savedPlayer.CreatedAt.Before(before))
			assert.False(t, playerRepo.savedPlayer.CreatedAt.After(after))
			assert.True(t, playerRepo.savedPlayer.UpdatedAt.Equal(updatedAt))
			require.NotNil(t, playerRepo.savedPlayer.DataCollectedAt)
			assert.True(t, playerRepo.savedPlayer.DataCollectedAt.Equal(updatedAt))
			userRepo.AssertExpectations(t)
		})
	}
}
