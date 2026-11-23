package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode/teamcodetest"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerTeamMySQL_所属チームを保存して復元する(t *testing.T) {
	// Given
	db := mysqltest.NewDatabase(t, "")
	ctx := context.Background()
	_, err := db.Exec("INSERT INTO users (id, username) VALUES (20, 'team_member')")
	require.NoError(t, err)
	seedPlayerWithHonors(t, db, 1, false)
	repo := NewPlayerRepository(db)
	player, err := repo.FindByID(ctx, db, 1)
	require.NoError(t, err)
	team, err := entity.NewPlayerTeam(teamcodetest.New(t, "AB12CD"), "チーム名", "green")
	require.NoError(t, err)
	player.ChangeTeam(team)

	// When
	require.NoError(t, repo.Save(ctx, db, player))

	// Then
	saved, err := repo.FindByID(ctx, db, 1)
	require.NoError(t, err)
	assert.Equal(t, team, saved.Team)

	// When: 未所属に戻す
	saved.ChangeTeam(nil)
	require.NoError(t, repo.Save(ctx, db, saved))

	// Then
	cleared, err := repo.FindByID(ctx, db, 1)
	require.NoError(t, err)
	assert.Nil(t, cleared.Team)
}

func TestPlayerTeamMySQL_チーム列の組み合わせをCHECK制約で検証する(t *testing.T) {
	tests := []struct {
		name string
		// Given
		code  *string
		team  *string
		color *string
		// Then
		wantErr bool
	}{
		{name: "未所属はすべてNULL", wantErr: false},
		{name: "チームコードとチーム名がある", code: new("AB12CD"), team: new("チーム"), wantErr: false},
		{name: "チームコード・チーム名・カラーがある", code: new("AB12CD"), team: new("チーム"), color: new("gold"), wantErr: false},
		{name: "チームコードが小文字を含む", code: new("ab12cd"), team: new("チーム"), wantErr: true},
		{name: "チームコードが5桁", code: new("AB12C"), team: new("チーム"), wantErr: true},
		{name: "チームコードがあるのにチーム名がない", code: new("AB12CD"), wantErr: true},
		{name: "チームコードがないのにチーム名がある", team: new("チーム"), wantErr: true},
		{name: "チームコードがないのにカラーがある", color: new("gold"), wantErr: true},
	}
	db := mysqltest.NewDatabase(t, "")
	_, err := db.Exec("INSERT INTO users (id, username) VALUES (20, 'team_member')")
	require.NoError(t, err)
	seedPlayerWithHonors(t, db, 1, false)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := db.Exec(`UPDATE players SET team_code = ?, team_name = ?, team_color = ? WHERE id = 1`, tt.code, tt.team, tt.color)

			// Then
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
