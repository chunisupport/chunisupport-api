package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/teamcode/teamcodetest"
	infrarepo "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamRepositoryMySQL_SaveAndFind(t *testing.T) {
	// Given
	db := mysqltest.NewDatabase(t, "")
	ctx := context.Background()
	repo := infrarepo.NewTeamRepository()
	collectedAt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	team, err := entity.NewTeam(teamcodetest.New(t, "AB12CD"), "チーム名", "green", collectedAt)
	require.NoError(t, err)

	// When: 新規登録
	require.NoError(t, repo.Save(ctx, db, team))

	// Then
	require.NotZero(t, team.ID)
	found, err := repo.FindByCode(ctx, db, teamcodetest.New(t, "AB12CD"))
	require.NoError(t, err)
	assert.Equal(t, team.ID, found.ID)
	assert.Equal(t, "AB12CD", found.Code.String())
	assert.Equal(t, "チーム名", found.Name)
	assert.Equal(t, new("green"), found.Color)
	assert.True(t, collectedAt.Equal(found.DataCollectedAt))

	// When: 新しい観測で更新
	observed, err := entity.NewTeam(teamcodetest.New(t, "AB12CD"), "新しい名前", "", collectedAt.Add(time.Hour))
	require.NoError(t, err)
	require.True(t, found.ApplyObservation(observed))
	require.NoError(t, repo.Save(ctx, db, found))

	// Then
	updated, err := repo.FindByCodeForUpdate(ctx, db, teamcodetest.New(t, "AB12CD"))
	require.NoError(t, err)
	assert.Equal(t, team.ID, updated.ID)
	assert.Equal(t, "新しい名前", updated.Name)
	assert.Nil(t, updated.Color)
	assert.True(t, collectedAt.Add(time.Hour).Equal(updated.DataCollectedAt))
}

func TestTeamRepositoryMySQL_FindByCode_存在しない場合はErrTeamNotFound(t *testing.T) {
	// Given
	db := mysqltest.NewDatabase(t, "")
	repo := infrarepo.NewTeamRepository()

	// When
	_, err := repo.FindByCode(context.Background(), db, teamcodetest.New(t, "ZZZZZZ"))
	_, errForUpdate := repo.FindByCodeForUpdate(context.Background(), db, teamcodetest.New(t, "ZZZZZZ"))

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrTeamNotFound)
	assert.ErrorIs(t, errForUpdate, domainrepo.ErrTeamNotFound)
}

func TestTeamRepositoryMySQL_Save_同じチームコードの登録はErrTeamAlreadyExists(t *testing.T) {
	// Given
	db := mysqltest.NewDatabase(t, "")
	ctx := context.Background()
	repo := infrarepo.NewTeamRepository()
	collectedAt := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	first, err := entity.NewTeam(teamcodetest.New(t, "AB12CD"), "先行", "green", collectedAt)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, db, first))
	second, err := entity.NewTeam(teamcodetest.New(t, "AB12CD"), "後続", "green", collectedAt)
	require.NoError(t, err)

	// When
	err = repo.Save(ctx, db, second)

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrTeamAlreadyExists)
	assert.Zero(t, second.ID)
}

func TestTeamMigrationMySQL_チームを削除すると所属プレイヤーは未所属になる(t *testing.T) {
	// Given
	db := mysqltest.NewDatabase(t, "")
	_, err := db.Exec(`
		INSERT INTO teams (id, code, name, data_collected_at) VALUES (1, 'AB12CD', 'チーム', '2026-11-22 10:00:00');
		INSERT INTO users (id, username) VALUES (1, 'team_member');
		INSERT INTO players (id, user_id, player_name, player_level, official_player_rating, team_id) VALUES (1, 1, 'プレイヤー', 1, 0, 1);
	`)
	require.NoError(t, err)

	// When
	_, err = db.Exec(`DELETE FROM teams WHERE id = 1`)
	require.NoError(t, err)

	// Then
	var teamID *int
	require.NoError(t, db.Get(&teamID, `SELECT team_id FROM players WHERE id = 1`))
	assert.Nil(t, teamID)
}

func TestTeamMigrationMySQL_チームコードは大文字英数字6桁だけを受け付ける(t *testing.T) {
	tests := []struct {
		name string
		// Given
		code string
		// Then
		wantErr bool
	}{
		{name: "大文字英数字6桁", code: "AB12CD", wantErr: false},
		{name: "小文字を含む", code: "ab12cd", wantErr: true},
		{name: "5桁", code: "AB12C", wantErr: true},
		{name: "記号を含む", code: "AB-2CD", wantErr: true},
	}
	db := mysqltest.NewDatabase(t, "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := db.Exec(`INSERT INTO teams (code, name, data_collected_at) VALUES (?, 'チーム', '2026-11-22 10:00:00')`, tt.code)

			// Then
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
