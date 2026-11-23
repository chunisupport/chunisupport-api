package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode/teamcodetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlayerTeamObservation(t *testing.T) {
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
			observation, err := newPlayerTeamObservation(tt.team)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResolved, observation.resolved)
			if tt.expectedCode == "" {
				assert.Nil(t, observation.team)
				return
			}
			require.NotNil(t, observation.team)
			assert.Equal(t, tt.expectedCode, observation.team.Code().String())
			assert.Equal(t, tt.expectedName, observation.team.Name())
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
			_, err := newPlayerTeamObservation(tt.team)

			// Then
			var validationErr *PlayerDataValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, tt.field, validationErr.Field)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestPlayerDataUsecase_Register_所属チームを保存する(t *testing.T) {
	newTeam := func(t *testing.T, code, name string) *entity.PlayerTeam {
		team, err := entity.NewPlayerTeam(teamcodetest.New(t, code), name, "green")
		require.NoError(t, err)
		return team
	}
	tests := []struct {
		name string
		// Given
		currentTeam func(t *testing.T) *entity.PlayerTeam
		team        PlayerDataTeamPayload
		// Then
		expectedCode string
		expectedName string
	}{
		{
			name:         "チームコードがある場合は所属させる",
			currentTeam:  func(*testing.T) *entity.PlayerTeam { return nil },
			team:         PlayerDataTeamPayload{Name: "チーム名", Color: "green", Code: "AB12CD"},
			expectedCode: "AB12CD",
			expectedName: "チーム名",
		},
		{
			name:         "所属チームの名前変更を反映する",
			currentTeam:  func(t *testing.T) *entity.PlayerTeam { return newTeam(t, "AB12CD", "旧名") },
			team:         PlayerDataTeamPayload{Name: "新名", Color: "green", Code: "AB12CD"},
			expectedCode: "AB12CD",
			expectedName: "新名",
		},
		{
			name:        "チーム情報がない場合は未所属にする",
			currentTeam: func(t *testing.T) *entity.PlayerTeam { return newTeam(t, "AB12CD", "チーム名") },
			team:        PlayerDataTeamPayload{},
		},
		{
			name:         "チームコードを取得できなかった場合は既存の所属を維持する",
			currentTeam:  func(t *testing.T) *entity.PlayerTeam { return newTeam(t, "AB12CD", "チーム名") },
			team:         PlayerDataTeamPayload{Name: "別名", Color: "green"},
			expectedCode: "AB12CD",
			expectedName: "チーム名",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			fx := newRegistrationFixture(t)
			fx.player.player.Team = tt.currentTeam(t)
			fx.payload.Team = tt.team

			// When
			_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "hash")

			// Then
			require.NoError(t, err)
			saved := fx.player.player.Team
			if tt.expectedCode == "" {
				assert.Nil(t, saved)
				return
			}
			require.NotNil(t, saved)
			assert.Equal(t, tt.expectedCode, saved.Code().String())
			assert.Equal(t, tt.expectedName, saved.Name())
		})
	}
}

func TestPlayerDataUsecase_Register_不正なチームコードは登録前に拒否する(t *testing.T) {
	// Given
	fx := newRegistrationFixture(t)
	fx.payload.Team = PlayerDataTeamPayload{Name: "チーム名", Code: "AB12C"}

	// When
	_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "hash")

	// Then
	var validationErr *PlayerDataValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "team.code", validationErr.Field)
	assert.False(t, fx.player.saved)
}
