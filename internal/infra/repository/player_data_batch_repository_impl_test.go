package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPlayerBatchRepository(t *testing.T) *sqlx.DB {
	t.Helper()
	db := setupPlayerRepositorySQLite(t)
	setupPlayerBatchSchema(t, db)
	return db
}

func setupPlayerBatchSchema(t *testing.T, db *sqlx.DB) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TABLE slots (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`INSERT INTO slots VALUES (1, 'none'), (2, 'best')`,
		`CREATE TABLE player_records (player_id INTEGER NOT NULL, chart_id INTEGER NOT NULL, score INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL, slot_id INTEGER NOT NULL, slot_order INTEGER NULL)`,
		`CREATE TABLE player_locked_songs (player_id INTEGER NOT NULL, song_id INTEGER NOT NULL, is_ultima BOOLEAN NOT NULL)`,
		`INSERT INTO player_records VALUES (1, 10, 1000000, 1, 2, 1)`,
	} {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}
}

func TestPlayerDataBatchRepository_ProcessPlayer_最新集約を保存する(t *testing.T) {
	db := setupPlayerBatchRepository(t)
	now := seedPlayerWithHonors(t, db, 1, false)
	ctx := context.Background()
	repo := NewPlayerDataBatchRepository(db)
	keys, err := repo.ListPlayerKeys(ctx, 0, 1, 10)
	require.NoError(t, err)
	require.Len(t, keys, 1)

	// 取得日時を変えない更新も、バッチのロック後に読み直す必要があります。
	playerRepo := NewPlayerRepository(db)
	tx, err := db.Beginx()
	require.NoError(t, err)
	player, err := playerRepo.FindByUserIDForUpdate(ctx, tx, 20)
	require.NoError(t, err)
	require.NotNil(t, player)
	player.Name = playername.MustNewPlayerName("最新の名前")
	require.NoError(t, playerRepo.Save(ctx, tx, player))
	_, err = tx.Exec(`INSERT INTO player_locked_songs VALUES (1, 100, true)`)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	build := func(data domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error) {
		assert.Equal(t, &now, data.DataCollectedAt)
		assert.Len(t, data.Records, 1)
		assert.Equal(t, []domainrepo.PlayerBatchLockedSong{{SongID: 100, IsUltima: true}}, data.LockedSongs)
		return domainrepo.PlayerBatchUpdate{PlayerRating: 16.5, BestAverage: 16.7, NewAverage: 16.2, Overpower: 12345}, nil
	}
	for range 2 {
		status, err := repo.ProcessPlayer(ctx, keys[0], build)
		require.NoError(t, err)
		assert.Equal(t, domainrepo.PlayerBatchUpdated, status)
	}
	saved, err := playerRepo.FindByID(ctx, db, 1)
	require.NoError(t, err)
	assert.Equal(t, "最新の名前", saved.Name.String())
	assert.Equal(t, player.OfficialRating, saved.OfficialRating)
	assert.Equal(t, player.OfficialOverpower, saved.OfficialOverpower)
	assert.Equal(t, player.OfficialOverpowerPercent, saved.OfficialOverpowerPercent)
	assert.Equal(t, player.DataCollectedAt, saved.DataCollectedAt)
	assert.Equal(t, player.UpdatedAt, saved.UpdatedAt)
	assert.Equal(t, player.CreatedAt, saved.CreatedAt)
	require.NotNil(t, saved.CalculatedRating)
	assert.Equal(t, 16.5, *saved.CalculatedRating)
	require.NotNil(t, saved.BestAverageRating)
	assert.Equal(t, 16.7, *saved.BestAverageRating)
	require.NotNil(t, saved.NewAverageRating)
	assert.Equal(t, 16.2, *saved.NewAverageRating)
	require.NotNil(t, saved.OverpowerValue)
	assert.Equal(t, 12345.0, *saved.OverpowerValue)
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM player_metric_histories`))
	assert.Zero(t, count)
}

func TestPlayerDataBatchRepository_ProcessPlayer_競合と削除を計算前に検出する(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		name := "取得日時の競合"
		if deleted {
			name = "削除済み"
		}
		t.Run(name, func(t *testing.T) {
			db := setupPlayerBatchRepository(t)
			now := seedPlayerWithHonors(t, db, 1, false)
			if deleted {
				_, err := db.Exec(`DELETE FROM players WHERE id = 1`)
				require.NoError(t, err)
			}
			older := now.Add(-time.Hour)
			called := false
			status, err := NewPlayerDataBatchRepository(db).ProcessPlayer(context.Background(), domainrepo.PlayerBatchKey{ID: 1, DataCollectedAt: &older}, func(domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error) {
				called = true
				return domainrepo.PlayerBatchUpdate{}, nil
			})
			require.NoError(t, err)
			assert.False(t, called)
			expected := domainrepo.PlayerBatchConflict
			if deleted {
				expected = domainrepo.PlayerBatchDeleted
			}
			assert.Equal(t, expected, status)
			var slot int
			require.NoError(t, db.Get(&slot, `SELECT slot_id FROM player_records WHERE player_id = 1`))
			assert.Equal(t, 2, slot)
		})
	}
}

func TestPlayerDataBatchRepository_ProcessPlayer_保存失敗時はスロットも戻す(t *testing.T) {
	db := setupPlayerBatchRepository(t)
	now := seedPlayerWithHonors(t, db, 1, false)
	_, err := db.Exec(`CREATE TRIGGER fail_player_save BEFORE UPDATE ON players BEGIN SELECT RAISE(FAIL, 'save failed'); END`)
	require.NoError(t, err)
	_, err = NewPlayerDataBatchRepository(db).ProcessPlayer(context.Background(), domainrepo.PlayerBatchKey{ID: 1, DataCollectedAt: &now}, func(domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error) {
		return domainrepo.PlayerBatchUpdate{ResetSlots: true, PlayerRating: 16.5}, nil
	})
	require.ErrorContains(t, err, "save failed")
	var slot int
	require.NoError(t, db.Get(&slot, `SELECT slot_id FROM player_records WHERE player_id = 1`))
	assert.Equal(t, 2, slot)
}

func TestPlayerDataBatchRepository_ProcessPlayer_計算エラーを返す(t *testing.T) {
	db := setupPlayerBatchRepository(t)
	now := seedPlayerWithHonors(t, db, 1, false)
	expected := errors.New("計算失敗")
	_, err := NewPlayerDataBatchRepository(db).ProcessPlayer(context.Background(), domainrepo.PlayerBatchKey{ID: 1, DataCollectedAt: &now}, func(domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error) {
		return domainrepo.PlayerBatchUpdate{}, expected
	})
	assert.ErrorIs(t, err, expected)
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM players`))
	assert.Equal(t, 1, count)
}
