package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode/teamcodetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlayerTeamObservation(t *testing.T) {
	collectedAt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		// Given
		team PlayerDataTeamPayload
		// Then
		expectedResolved bool
		expectedCode     string
		expectedName     string
	}{
		{
			name:             "チーム情報がない場合は未所属として確定する",
			team:             PlayerDataTeamPayload{},
			expectedResolved: true,
		},
		{
			name:             "チームコードがない場合は所属を確定しない",
			team:             PlayerDataTeamPayload{Name: "チーム名", Color: "green"},
			expectedResolved: false,
		},
		{
			name:             "チームコードがある場合はチームを確定する",
			team:             PlayerDataTeamPayload{Name: "チーム名", Color: "green", Code: "AB12CD"},
			expectedResolved: true,
			expectedCode:     "AB12CD",
			expectedName:     "チーム名",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			observation, err := newPlayerTeamObservation(tt.team, collectedAt)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResolved, observation.resolved)
			if tt.expectedCode == "" {
				assert.Nil(t, observation.team)
				return
			}
			require.NotNil(t, observation.team)
			assert.Equal(t, tt.expectedCode, observation.team.Code.String())
			assert.Equal(t, tt.expectedName, observation.team.Name)
			assert.Equal(t, collectedAt, observation.team.DataCollectedAt)
		})
	}
}

func TestNewPlayerTeamObservation_不正なチーム情報はフィールド付き検証エラーになる(t *testing.T) {
	tests := []struct {
		name string
		// Given
		team PlayerDataTeamPayload
		// Then
		field   string
		wantErr error
	}{
		{
			name:    "チームコードの形式が不正",
			team:    PlayerDataTeamPayload{Name: "チーム名", Code: "ab12cd"},
			field:   "team.code",
			wantErr: teamcode.ErrInvalidTeamCode,
		},
		{
			name:    "チームコードがあるのにチーム名がない",
			team:    PlayerDataTeamPayload{Code: "AB12CD"},
			field:   "team.name",
			wantErr: entity.ErrTeamNameInvalid,
		},
		{
			name:    "チームカラーが長すぎる",
			team:    PlayerDataTeamPayload{Name: "チーム名", Code: "AB12CD", Color: strings.Repeat("a", entity.TeamColorMaxLength+1)},
			field:   "team.color",
			wantErr: entity.ErrTeamColorInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := newPlayerTeamObservation(tt.team, time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC))

			// Then
			var validationErr *PlayerDataValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, tt.field, validationErr.Field)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// stubTeamRepository はチームコードごとに1件のチームを保持するスタブです。
type stubTeamRepository struct {
	stored *entity.Team
	// concurrentlyInserted は FindByCode では見えず INSERT で重複する、並行登録済みのチームです。
	concurrentlyInserted *entity.Team
	findCalls            int
	findForUpdateCalls   int
	inserted             []*entity.Team
	updated              []*entity.Team
}

func (r *stubTeamRepository) FindByCode(_ context.Context, _ repository.Executor, code teamcode.TeamCode) (*entity.Team, error) {
	r.findCalls++
	if r.stored == nil || r.stored.Code != code {
		return nil, repository.ErrTeamNotFound
	}
	copied := *r.stored
	return &copied, nil
}

func (r *stubTeamRepository) FindByCodeForUpdate(_ context.Context, _ repository.Executor, code teamcode.TeamCode) (*entity.Team, error) {
	r.findForUpdateCalls++
	if r.stored == nil || r.stored.Code != code {
		return nil, repository.ErrTeamNotFound
	}
	copied := *r.stored
	return &copied, nil
}

func (r *stubTeamRepository) Save(_ context.Context, _ repository.Executor, team *entity.Team) error {
	if team.ID != 0 {
		r.updated = append(r.updated, team)
		r.stored = team
		return nil
	}
	if r.concurrentlyInserted != nil {
		r.stored = r.concurrentlyInserted
		r.concurrentlyInserted = nil
		return errors.Join(repository.ErrTeamAlreadyExists, errors.New("duplicate entry"))
	}
	team.ID = 100 + len(r.inserted)
	r.inserted = append(r.inserted, team)
	r.stored = team
	return nil
}

func TestPlayerDataUsecase_EnsureTeam(t *testing.T) {
	collectedAt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	existing := func(t *testing.T, name string, at time.Time) *entity.Team {
		team, err := entity.NewTeam(teamcodetest.New(t, "AB12CD"), name, "green", at)
		require.NoError(t, err)
		team.ID = 5
		return team
	}

	t.Run("未登録のチームは新規登録する", func(t *testing.T) {
		// Given
		repo := &stubTeamRepository{}
		uc := &playerDataUsecase{teamRepo: repo}
		observed, err := entity.NewTeam(teamcodetest.New(t, "AB12CD"), "チーム名", "green", collectedAt)
		require.NoError(t, err)

		// When
		teamID, err := uc.ensureTeam(context.Background(), nil, observed)

		// Then
		require.NoError(t, err)
		assert.Equal(t, 100, teamID)
		assert.Len(t, repo.inserted, 1)
		assert.Empty(t, repo.updated)
		assert.Zero(t, repo.findForUpdateCalls)
	})

	t.Run("登録済みのチームは新しい観測で更新する", func(t *testing.T) {
		// Given
		repo := &stubTeamRepository{stored: existing(t, "旧名", collectedAt.Add(-time.Hour))}
		uc := &playerDataUsecase{teamRepo: repo}
		observed, err := entity.NewTeam(teamcodetest.New(t, "AB12CD"), "新名", "green", collectedAt)
		require.NoError(t, err)

		// When
		teamID, err := uc.ensureTeam(context.Background(), nil, observed)

		// Then
		require.NoError(t, err)
		assert.Equal(t, 5, teamID)
		assert.Empty(t, repo.inserted)
		require.Len(t, repo.updated, 1)
		assert.Equal(t, "新名", repo.updated[0].Name)
		assert.Equal(t, 1, repo.findForUpdateCalls)
	})

	t.Run("登録済みのチームより古い観測では更新しない", func(t *testing.T) {
		// Given
		repo := &stubTeamRepository{stored: existing(t, "新名", collectedAt.Add(time.Hour))}
		uc := &playerDataUsecase{teamRepo: repo}
		observed, err := entity.NewTeam(teamcodetest.New(t, "AB12CD"), "旧名", "green", collectedAt)
		require.NoError(t, err)

		// When
		teamID, err := uc.ensureTeam(context.Background(), nil, observed)

		// Then
		require.NoError(t, err)
		assert.Equal(t, 5, teamID)
		assert.Empty(t, repo.inserted)
		assert.Empty(t, repo.updated)
	})

	t.Run("並行登録で重複した場合は登録済みのチームを更新する", func(t *testing.T) {
		// Given
		repo := &stubTeamRepository{concurrentlyInserted: existing(t, "先行", collectedAt.Add(-time.Hour))}
		uc := &playerDataUsecase{teamRepo: repo}
		observed, err := entity.NewTeam(teamcodetest.New(t, "AB12CD"), "後続", "green", collectedAt)
		require.NoError(t, err)

		// When
		teamID, err := uc.ensureTeam(context.Background(), nil, observed)

		// Then
		require.NoError(t, err)
		assert.Equal(t, 5, teamID)
		assert.Empty(t, repo.inserted)
		require.Len(t, repo.updated, 1)
		assert.Equal(t, "後続", repo.updated[0].Name)
	})
}

func TestPlayerDataUsecase_Register_所属チームを保存する(t *testing.T) {
	tests := []struct {
		name string
		// Given
		currentTeamID *int
		team          PlayerDataTeamPayload
		// Then
		expectedTeamID *int
		expectInserted bool
	}{
		{
			name:           "チームコードがある場合はチームを登録して所属させる",
			currentTeamID:  nil,
			team:           PlayerDataTeamPayload{Name: "チーム名", Color: "green", Code: "AB12CD"},
			expectedTeamID: new(100),
			expectInserted: true,
		},
		{
			name:           "チーム情報がない場合は未所属にする",
			currentTeamID:  new(5),
			team:           PlayerDataTeamPayload{},
			expectedTeamID: nil,
		},
		{
			name:           "チームコードを取得できなかった場合は既存の所属を維持する",
			currentTeamID:  new(5),
			team:           PlayerDataTeamPayload{Name: "チーム名", Color: "green"},
			expectedTeamID: new(5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			fx := newRegistrationFixture(t)
			teams := &stubTeamRepository{}
			fx.uc.teamRepo = teams
			fx.player.player.TeamID = tt.currentTeamID
			fx.payload.Team = tt.team

			// When
			_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "hash")

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedTeamID, fx.player.player.TeamID)
			assert.Equal(t, tt.expectInserted, len(teams.inserted) == 1)
		})
	}
}

func TestPlayerDataUsecase_Register_不正なチームコードは登録前に拒否する(t *testing.T) {
	// Given
	fx := newRegistrationFixture(t)
	fx.uc.teamRepo = &stubTeamRepository{}
	fx.payload.Team = PlayerDataTeamPayload{Name: "チーム名", Code: "AB12C"}

	// When
	_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "hash")

	// Then
	var validationErr *PlayerDataValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "team.code", validationErr.Field)
	assert.False(t, fx.player.saved)
}
