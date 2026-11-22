package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode/teamcodetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTeam(t *testing.T) {
	collectedAt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		// Given
		teamName string
		color    string
		// Then
		wantErr       error
		expectedName  string
		expectedColor *string
	}{
		{
			name:          "チーム名とカラーを保持する",
			teamName:      "チーム名",
			color:         "green",
			expectedName:  "チーム名",
			expectedColor: new("green"),
		},
		{
			name:          "チーム名の前後の空白を除去する",
			teamName:      "  チーム名\n",
			color:         "gold",
			expectedName:  "チーム名",
			expectedColor: new("gold"),
		},
		{
			name:          "カラーが空の場合は未取得として扱う",
			teamName:      "チーム名",
			color:         "",
			expectedName:  "チーム名",
			expectedColor: nil,
		},
		{
			name:     "チーム名が空白のみの場合はエラー",
			teamName: "   ",
			color:    "green",
			wantErr:  ErrTeamNameInvalid,
		},
		{
			name:     "チーム名が上限を超える場合はエラー",
			teamName: strings.Repeat("あ", TeamNameMaxLength+1),
			color:    "green",
			wantErr:  ErrTeamNameInvalid,
		},
		{
			name:     "カラーが上限を超える場合はエラー",
			teamName: "チーム名",
			color:    strings.Repeat("a", TeamColorMaxLength+1),
			wantErr:  ErrTeamColorInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			team, err := NewTeam(teamcodetest.New(t, "AB12CD"), tt.teamName, tt.color, collectedAt)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, team)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 0, team.ID)
			assert.Equal(t, "AB12CD", team.Code.String())
			assert.Equal(t, tt.expectedName, team.Name)
			assert.Equal(t, tt.expectedColor, team.Color)
			assert.Equal(t, collectedAt, team.DataCollectedAt)
		})
	}
}

func TestTeam_ApplyObservation(t *testing.T) {
	collectedAt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		// Given
		observedName string
		observedAt   time.Time
		// Then
		expectedChanged bool
		expectedName    string
		expectedAt      time.Time
	}{
		{
			name:            "より新しい取得日時の名前変更を反映する",
			observedName:    "新しい名前",
			observedAt:      collectedAt.Add(time.Hour),
			expectedChanged: true,
			expectedName:    "新しい名前",
			expectedAt:      collectedAt.Add(time.Hour),
		},
		{
			name:            "内容が同じでも新しい取得日時へ進める",
			observedName:    "チーム名",
			observedAt:      collectedAt.Add(time.Hour),
			expectedChanged: true,
			expectedName:    "チーム名",
			expectedAt:      collectedAt.Add(time.Hour),
		},
		{
			name:            "古い取得日時の観測は反映しない",
			observedName:    "古い名前",
			observedAt:      collectedAt.Add(-time.Hour),
			expectedChanged: false,
			expectedName:    "チーム名",
			expectedAt:      collectedAt,
		},
		{
			name:            "同じ取得日時の観測は反映しない",
			observedName:    "別の名前",
			observedAt:      collectedAt,
			expectedChanged: false,
			expectedName:    "チーム名",
			expectedAt:      collectedAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			team, err := NewTeam(teamcodetest.New(t, "AB12CD"), "チーム名", "green", collectedAt)
			require.NoError(t, err)
			observed, err := NewTeam(teamcodetest.New(t, "AB12CD"), tt.observedName, "green", tt.observedAt)
			require.NoError(t, err)

			// When
			changed := team.ApplyObservation(observed)

			// Then
			assert.Equal(t, tt.expectedChanged, changed)
			assert.Equal(t, tt.expectedName, team.Name)
			assert.Equal(t, tt.expectedAt, team.DataCollectedAt)
		})
	}
}

func TestTeam_ApplyObservation_カラーの変更を反映する(t *testing.T) {
	// Given
	collectedAt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	team, err := NewTeam(teamcodetest.New(t, "AB12CD"), "チーム名", "green", collectedAt)
	require.NoError(t, err)
	observed, err := NewTeam(teamcodetest.New(t, "AB12CD"), "チーム名", "rainbow", collectedAt.Add(time.Hour))
	require.NoError(t, err)

	// When
	changed := team.ApplyObservation(observed)

	// Then
	assert.True(t, changed)
	assert.Equal(t, new("rainbow"), team.Color)
}

func TestPlayer_ChangeTeam(t *testing.T) {
	tests := []struct {
		name string
		// Given
		current *int
		next    *int
		// Then
		expected *int
	}{
		{name: "未所属からチームへ所属する", current: nil, next: new(10), expected: new(10)},
		{name: "別のチームへ移る", current: new(10), next: new(20), expected: new(20)},
		{name: "チームを抜ける", current: new(10), next: nil, expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			player := &Player{TeamID: tt.current}

			// When
			player.ChangeTeam(tt.next)

			// Then
			assert.Equal(t, tt.expected, player.TeamID)
		})
	}
}
