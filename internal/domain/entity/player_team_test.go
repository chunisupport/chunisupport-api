package entity

import (
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode/teamcodetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlayerTeam(t *testing.T) {
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
		{name: "チーム名とカラーを保持する", teamName: "チーム名", color: "green", expectedName: "チーム名", expectedColor: new("green")},
		{name: "チーム名の前後の空白を除去する", teamName: "  チーム名\n", color: "gold", expectedName: "チーム名", expectedColor: new("gold")},
		{name: "カラーが空の場合は未取得として扱う", teamName: "チーム名", color: "", expectedName: "チーム名", expectedColor: nil},
		{name: "チーム名が上限ちょうどの場合は有効", teamName: strings.Repeat("あ", TeamNameMaxLength), color: "green", expectedName: strings.Repeat("あ", TeamNameMaxLength), expectedColor: new("green")},
		{name: "チーム名が空白のみの場合はエラー", teamName: "   ", color: "green", wantErr: ErrTeamNameInvalid},
		{name: "チーム名が上限を超える場合はエラー", teamName: strings.Repeat("あ", TeamNameMaxLength+1), color: "green", wantErr: ErrTeamNameInvalid},
		{name: "カラーが非ASCII文字を含む場合はエラー", teamName: "チーム名", color: "緑", wantErr: ErrTeamColorInvalid},
		{name: "カラーが大文字を含む場合はエラー", teamName: "チーム名", color: "Green", wantErr: ErrTeamColorInvalid},
		{name: "カラーが上限を超える場合はエラー", teamName: "チーム名", color: strings.Repeat("a", TeamColorMaxLength+1), wantErr: ErrTeamColorInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			team, err := NewPlayerTeam(teamcodetest.New(t, "AB12CD"), tt.teamName, tt.color)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, team)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "AB12CD", team.Code().String())
			assert.Equal(t, tt.expectedName, team.Name())
			assert.Equal(t, tt.expectedColor, team.Color())
		})
	}
}
