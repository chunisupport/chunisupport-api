package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/masterfingerprint"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername/playernametest"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
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
		ID: 10, UserID: 1, Name: playernametest.New(t, "変更前"), Level: 40,
		OfficialRating: 17.24, OfficialOverpower: 12340.12,
		DataCollectedAt: &collectedAt, CreatedAt: collectedAt.Add(-24 * time.Hour), UpdatedAt: collectedAt,
	}}
	userRepo := new(MockUserRepository)
	uc := &playerDataUsecase{playerRepo: playerRepo, userRepo: userRepo}
	playerID := 10
	user := &entity.User{ID: 1, Username: usernametest.New(t, "playerdatatest"), PlayerID: &playerID}
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
		ID: 10, UserID: 1, Name: playernametest.New(t, "変更前"), Level: 1,
		OfficialRating: 0, OfficialOverpower: 0, CreatedAt: createdAt, UpdatedAt: createdAt,
	}}
	userRepo := new(MockUserRepository)
	uc := &playerDataUsecase{playerRepo: playerRepo, userRepo: userRepo}
	playerID := 10
	user := &entity.User{ID: 1, Username: usernametest.New(t, "playerdatatest"), PlayerID: &playerID}
	userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	_, _, err := uc.ensurePlayer(context.Background(), nil, user, &PlayerDataSummaryInput{
		Name: "変更後", Level: 1, OfficialRating: 17.25, OfficialOverpower: 12345.67,
	}, createdAt.Add(time.Hour))

	require.NoError(t, err)
	assert.Nil(t, playerRepo.savedPlayer.PendingMetricHistory())
}

func TestEnsurePlayer_指定したポゼッションを保存する(t *testing.T) {
	createdAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	playerRepo := &stubPlayerRepositoryForPlayerData{foundPlayer: &entity.Player{
		ID: 10, UserID: 1, Name: playernametest.New(t, "変更前"), Level: 1,
		PossessionID: entity.DefaultPossessionID, CreatedAt: createdAt, UpdatedAt: createdAt,
	}}
	userRepo := new(MockUserRepository)
	uc := &playerDataUsecase{playerRepo: playerRepo, userRepo: userRepo}
	playerID := 10
	user := &entity.User{ID: 1, Username: usernametest.New(t, "playerdatatest"), PlayerID: &playerID}
	userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	_, _, err := uc.ensurePlayer(context.Background(), nil, user, &PlayerDataSummaryInput{
		Name: "変更後", Level: 1, PossessionID: 5, OfficialRating: 17.25, OfficialOverpower: 12345.67,
	}, createdAt.Add(time.Hour))

	require.NoError(t, err)
	assert.Equal(t, 5, playerRepo.savedPlayer.PossessionID)
}

func TestEnsurePlayer_未指定のポゼッションはnormalで初期化する(t *testing.T) {
	playerRepo := &stubPlayerRepositoryForPlayerData{}
	userRepo := new(MockUserRepository)
	uc := &playerDataUsecase{playerRepo: playerRepo, userRepo: userRepo}
	user := &entity.User{ID: 1, Username: usernametest.New(t, "playerdatatest")}
	userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	_, _, err := uc.ensurePlayer(context.Background(), nil, user, &PlayerDataSummaryInput{
		Name: "新規", Level: 1, OfficialRating: 17.25, OfficialOverpower: 12345.67,
	}, time.Date(2026, 8, 7, 11, 0, 0, 0, time.UTC))

	require.NoError(t, err)
	assert.Equal(t, entity.DefaultPossessionID, playerRepo.savedPlayer.PossessionID)
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

func (s *stubPlayerRepositoryForPlayerData) FindByIDForUpdate(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	return s.FindByID(ctx, exec, id)
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
		ID: 10, UserID: 1, Name: playernametest.New(t, "登録済み"), Level: 50,
		OfficialRating: 17.25, OfficialOverpower: 12345.67, OfficialOverpowerPercent: &officialOverpowerPercent,
		DataCollectedAt: &updatedAt, CreatedAt: updatedAt.Add(-24 * time.Hour), UpdatedAt: updatedAt,
	}}
	latestUpdate, err := entity.NewPlayerLatestUpdate(10, 1, []byte("gzip-payload"), updatedAt, updatedAt.Add(time.Minute), "saved-hash")
	require.NoError(t, err)
	playerDataRepo := &stubPlayerDataRepositoryForApplyScoresTest{latestUpdate: latestUpdate}
	txExecutor := &sqlx.DB{}
	tm := &passthroughTransactionManagerForPlayerDataIdentity{exec: txExecutor}
	user := &entity.User{ID: 1, Username: usernametest.New(t, "playerdatatest"), PlayerID: &playerID}
	userRepo := new(MockUserRepository)
	userRepo.On("FindByIDForUpdate", mock.Anything, txExecutor, 1).Return(user, nil).Once()
	uc := &playerDataUsecase{
		tm: tm, userRepo: userRepo, playerRepo: playerRepo,
		playerDataRepo: playerDataRepo, masterCache: stubPlayerDataMasterProviderForIdentity{},
	}
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
	userRepo.AssertExpectations(t)
}

func (s *stubPlayerRepositoryForPlayerData) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}

func TestPlayerDataUsecaseDelete_ロック後の最新ユーザーを保存する(t *testing.T) {
	// Given
	txExecutor := &sqlx.DB{}
	tm := &passthroughTransactionManagerForPlayerDataIdentity{exec: txExecutor}
	playerID := 10
	staleUser := &entity.User{ID: 1, Username: usernametest.New(t, "oldname"), PlayerID: &playerID}
	lockedUser := &entity.User{ID: 1, Username: usernametest.New(t, "newname"), PlayerID: &playerID}
	userRepo := new(MockUserRepository)
	userRepo.On("FindByIDForUpdate", mock.Anything, txExecutor, 1).Return(lockedUser, nil).Once()
	userRepo.On("Save", mock.Anything, txExecutor, mock.MatchedBy(func(savedUser *entity.User) bool {
		return savedUser.Username.String() == "newname" && !savedUser.HasLinkedPlayer()
	})).Return(nil).Once()
	uc := &playerDataUsecase{
		tm:         tm,
		userRepo:   userRepo,
		playerRepo: &stubPlayerRepositoryForPlayerData{},
	}

	// When
	err := uc.Delete(context.Background(), staleUser)

	// Then
	require.NoError(t, err)
	assert.Equal(t, "oldname", staleUser.Username.String())
	userRepo.AssertExpectations(t)
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
				Username: usernametest.New(t, "playerdatatest"),
			}
			updatedAt := time.Date(2026, 3, 16, 15, 28, 53, 0, time.FixedZone("JST", 9*60*60))
			before := time.Now()
			officialRating := 16.25
			playerName := playernametest.New(t, "テストプレイヤー")
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

func TestEnsurePlayer_登録時は再計算済みの記録を無効にする(t *testing.T) {
	// Given
	createdAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	fingerprint := masterfingerprint.Compute([]byte("master"))
	playerRepo := &stubPlayerRepositoryForPlayerData{foundPlayer: &entity.Player{
		ID: 10, UserID: 1, Name: playernametest.New(t, "変更前"), Level: 1,
		CreatedAt: createdAt, UpdatedAt: createdAt, RecalculatedMasterFingerprint: &fingerprint,
	}}
	userRepo := new(MockUserRepository)
	uc := &playerDataUsecase{playerRepo: playerRepo, userRepo: userRepo}
	playerID := 10
	user := &entity.User{ID: 1, Username: usernametest.New(t, "playerdatatest"), PlayerID: &playerID}
	userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	// When
	_, _, err := uc.ensurePlayer(context.Background(), nil, user, &PlayerDataSummaryInput{
		Name: "変更後", Level: 1, OfficialRating: 17.25, OfficialOverpower: 12345.67,
	}, createdAt.Add(time.Hour))

	// Then
	require.NoError(t, err)
	assert.Nil(t, playerRepo.savedPlayer.RecalculatedMasterFingerprint)
}

// hookHonorRepositoryForRegisterTest は Register 経由で称号適用の成否を差し替えるための最小スタブです。
// 称号の削除呼び出しだけを差し替え、それ以外の HonorRepository の振る舞いには依存しません。
type hookHonorRepositoryForRegisterTest struct {
	repository.HonorRepository
	hookErr error
}

func (r *hookHonorRepositoryForRegisterTest) DeletePlayerHonorsExceptSlots(context.Context, repository.Executor, int, []int) error {
	return r.hookErr
}

// saveHookPlayerDataRepositoryForRegisterTest は Register 経由で最新更新日時保存の結果を差し替えるための最小スタブです。
type saveHookPlayerDataRepositoryForRegisterTest struct {
	*registrationDataRepository
	saveErr error
}

func (r *saveHookPlayerDataRepositoryForRegisterTest) SaveLatestUpdate(context.Context, repository.Executor, *entity.PlayerLatestUpdate) error {
	return r.saveErr
}

func TestRegister_成功時は指標の前後値と読み飛ばし件数を固定値で返す(t *testing.T) {
	// Given: 登録前の計算レート15.0とOVER POWER値100.0を持つ既存プレイヤーで、スコアも称号も有効なものがない登録を受け付ける。
	// 登録後は集計対象が無いため計算レート0.0・OVER POWER値0.0・OP%0.0になり、最大OP合計200.0に対する登録前OP%は50.0になる。
	fixture := newRegistrationFixture(t)
	beforeRating := 15.0
	beforeOverpower := 100.0
	fixture.player.player.CalculatedRating = &beforeRating
	fixture.player.player.OverpowerValue = &beforeOverpower
	fixture.data.overpowerStats = &repository.OverpowerTargetStats{MaxOverpowerTotal: 200.0}
	fixture.uc.honorRepo = &hookHonorRepositoryForRegisterTest{}
	fixture.payload.Honors = map[string]PlayerDataHonorPayload{
		"0": {Title: "範囲外", Class: "normal"},
	}

	// When
	result, err := fixture.uc.Register(context.Background(), fixture.user, fixture.payload, "result-contents-hash")

	// Then: 指標の登録前と登録後は互いに異なる固定値で返り、取り違えるとこのテストが落ちる。
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 77, result.PlayerID)
	require.NotNil(t, result.MetricDiffs.Rating.Before)
	assert.Equal(t, 15.0, *result.MetricDiffs.Rating.Before)
	require.NotNil(t, result.MetricDiffs.Rating.After)
	assert.Equal(t, 0.0, *result.MetricDiffs.Rating.After)
	require.NotNil(t, result.MetricDiffs.Rating.Delta)
	assert.Equal(t, -15.0, *result.MetricDiffs.Rating.Delta)
	require.NotNil(t, result.MetricDiffs.OverpowerValue.Before)
	assert.Equal(t, 100.0, *result.MetricDiffs.OverpowerValue.Before)
	require.NotNil(t, result.MetricDiffs.OverpowerValue.After)
	assert.Equal(t, 0.0, *result.MetricDiffs.OverpowerValue.After)
	require.NotNil(t, result.MetricDiffs.OverpowerValue.Delta)
	assert.Equal(t, -100.0, *result.MetricDiffs.OverpowerValue.Delta)
	require.NotNil(t, result.MetricDiffs.OverpowerPercent)
	require.NotNil(t, result.MetricDiffs.OverpowerPercent.Before)
	assert.Equal(t, 50.0, *result.MetricDiffs.OverpowerPercent.Before)
	require.NotNil(t, result.MetricDiffs.OverpowerPercent.After)
	assert.Equal(t, 0.0, *result.MetricDiffs.OverpowerPercent.After)
	require.NotNil(t, result.MetricDiffs.OverpowerPercent.Delta)
	assert.Equal(t, -50.0, *result.MetricDiffs.OverpowerPercent.Delta)
	// Then: 登録結果の表示側にも登録後の値が入り、読み飛ばした称号1件が件数と詳細の両方へ転記される。
	require.NotNil(t, result.Profile.Rating)
	assert.Equal(t, 0.0, *result.Profile.Rating)
	require.NotNil(t, result.Summary.Rating)
	assert.Equal(t, 0.0, *result.Summary.Rating)
	assert.Equal(t, 1, result.Counts.HonorsSkipped)
	require.Len(t, result.SkippedRecords, 1)
	assert.Equal(t, "honor", result.SkippedRecords[0].RecordType)
	assert.Equal(t, "slot out of range: 0", result.SkippedRecords[0].Reason)
}

func TestRegister_下位エラーの扱いを種類ごとに守る(t *testing.T) {
	// Given: 下位の処理が返すエラーと、Register が守るべき扱い。
	validationSentinel := &PlayerDataValidationError{Field: "honor", Message: "hook validation"}
	notFoundSentinel := &PlayerDataNotFoundError{Resource: "honor", Key: "hook"}
	conflictSentinel := &PlayerDataConflictError{Reason: "hook conflict"}
	genericCause := errors.New("hook failure")
	saveConflictCause := fmt.Errorf("latest update save failed: %w", entity.ErrConflictingPlayerDataBody)
	tests := []struct {
		name string
		// Given: 仕掛けるフック。称号適用と最新更新保存のどちらで失敗させるか。
		honorHookErr error
		saveHookErr  error
		// Then: 期待する扱い。同一値で返す場合は wantSame、競合へ変換する場合は wantConflict を設定し、
		// どちらでもなければ登録失敗として包まれることを確かめる。
		wantSame     error
		wantConflict bool
	}{
		{
			name:         "検証エラーは包まずにそのまま返す",
			honorHookErr: validationSentinel,
			wantSame:     validationSentinel,
		},
		{
			name:         "未発見エラーは包まずにそのまま返す",
			honorHookErr: notFoundSentinel,
			wantSame:     notFoundSentinel,
		},
		{
			name:         "競合エラーは包まずにそのまま返す",
			honorHookErr: conflictSentinel,
			wantSame:     conflictSentinel,
		},
		{
			name:         "想定外のエラーは登録失敗として包む",
			honorHookErr: genericCause,
		},
		{
			name:         "最新更新の保存競合は競合エラーへ変換する",
			saveHookErr:  saveConflictCause,
			wantConflict: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			fixture := newRegistrationFixture(t)
			fixture.payload.Honors = map[string]PlayerDataHonorPayload{}
			fixture.uc.honorRepo = &hookHonorRepositoryForRegisterTest{hookErr: tt.honorHookErr}
			if tt.saveHookErr != nil {
				fixture.uc.playerDataRepo = &saveHookPlayerDataRepositoryForRegisterTest{registrationDataRepository: fixture.data, saveErr: tt.saveHookErr}
			}

			// When
			result, err := fixture.uc.Register(context.Background(), fixture.user, fixture.payload, "hook-hash")

			// Then
			require.Error(t, err)
			assert.Nil(t, result)
			if tt.wantSame != nil {
				assert.Same(t, tt.wantSame, err)
				return
			}
			if tt.wantConflict {
				var conflictErr *PlayerDataConflictError
				require.ErrorAs(t, err, &conflictErr)
				assert.ErrorIs(t, err, entity.ErrConflictingPlayerDataBody)
				assert.ErrorIs(t, err, tt.saveHookErr)
				return
			}
			assert.ErrorIs(t, err, tt.honorHookErr)
			assert.Equal(t, "player data registration failed (user_id=7, player_id=77): hook failure", err.Error())
		})
	}
}
