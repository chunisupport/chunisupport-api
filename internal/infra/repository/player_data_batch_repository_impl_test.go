package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/masterfingerprint"
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
		`CREATE TABLE player_records (player_id INTEGER NOT NULL, chart_id INTEGER NOT NULL, score INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL, slot_id INTEGER NOT NULL, slot_order INTEGER NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE player_locked_songs (player_id INTEGER NOT NULL, song_id INTEGER NOT NULL, is_ultima BOOLEAN NOT NULL)`,
		`INSERT INTO player_records VALUES (1, 10, 1000000, 1, 2, 1, '2026-09-01 10:00:00')`,
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
	keys, err := repo.ListPlayerKeys(ctx, 0, 1, 10, masterfingerprint.Compute([]byte("master")))
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
		return domainrepo.PlayerBatchUpdate{PlayerRating: 16.5, BestAverage: 16.7, NewAverage: 16.2, Overpower: 12345, MasterFingerprint: masterfingerprint.Compute([]byte("master"))}, nil
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
		return domainrepo.PlayerBatchUpdate{ClearChartIDs: []int{10}, PlayerRating: 16.5, MasterFingerprint: masterfingerprint.Compute([]byte("master"))}, nil
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

func TestPlayerDataBatchRepository_ListPlayerKeys_再計算済みのプレイヤーを除外する(t *testing.T) {
	// Given
	db := setupPlayerBatchRepository(t)
	current := masterfingerprint.Compute([]byte("current"))
	for id, fingerprint := range map[int]any{1: nil, 2: current.String(), 3: masterfingerprint.Compute([]byte("old")).String()} {
		seedPlayerForBatchList(t, db, id, fingerprint)
	}

	// When
	keys, err := NewPlayerDataBatchRepository(db).ListPlayerKeys(context.Background(), 0, 3, 10, current)

	// Then
	require.NoError(t, err)
	ids := make([]int, 0, len(keys))
	for _, key := range keys {
		ids = append(ids, key.ID)
	}
	assert.Equal(t, []int{1, 3}, ids)
}

func TestPlayerDataBatchRepository_ProcessPlayer_フィンガープリントを記録する(t *testing.T) {
	// Given
	db := setupPlayerBatchRepository(t)
	now := seedPlayerWithHonors(t, db, 1, false)
	fingerprint := masterfingerprint.Compute([]byte("master"))

	// When
	status, err := NewPlayerDataBatchRepository(db).ProcessPlayer(context.Background(), domainrepo.PlayerBatchKey{ID: 1, DataCollectedAt: &now}, func(domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error) {
		return domainrepo.PlayerBatchUpdate{PlayerRating: 16.5, Overpower: 12345, MasterFingerprint: fingerprint}, nil
	})

	// Then
	require.NoError(t, err)
	assert.Equal(t, domainrepo.PlayerBatchUpdated, status)
	saved, err := NewPlayerRepository(db).FindByID(context.Background(), db, 1)
	require.NoError(t, err)
	require.NotNil(t, saved.RecalculatedMasterFingerprint)
	assert.Equal(t, fingerprint, *saved.RecalculatedMasterFingerprint)
	require.NotNil(t, saved.CalculatedRating)
	assert.Equal(t, 16.5, *saved.CalculatedRating)
}

func TestPlayerDataBatchRepository_ProcessPlayer_枠の差分だけを更新する(t *testing.T) {
	type recordRow struct {
		ChartID   int    `db:"chart_id"`
		SlotID    int    `db:"slot_id"`
		SlotOrder *int   `db:"slot_order"`
		UpdatedAt string `db:"updated_at"`
	}
	order := func(v int) *int { return &v }
	tests := []struct {
		name     string
		update   domainrepo.PlayerBatchUpdate
		expected []recordRow
	}{
		{
			name:   "差分がなければ成績行を変更しない",
			update: domainrepo.PlayerBatchUpdate{ClearChartIDs: []int{}, Assignments: []domainrepo.PlayerBatchSlotAssignment{}},
			expected: []recordRow{
				{ChartID: 10, SlotID: 2, SlotOrder: order(1), UpdatedAt: "2026-09-01 10:00:00"},
				{ChartID: 11, SlotID: 2, SlotOrder: order(2), UpdatedAt: "2026-09-01 10:00:00"},
				{ChartID: 12, SlotID: 1, SlotOrder: nil, UpdatedAt: "2026-09-01 10:00:00"},
			},
		},
		{
			name: "外す譜面と付け替える譜面だけを更新する",
			update: domainrepo.PlayerBatchUpdate{
				ClearChartIDs: []int{11},
				Assignments:   []domainrepo.PlayerBatchSlotAssignment{{ChartID: 12, SlotID: 2, Position: 2}},
			},
			expected: []recordRow{
				{ChartID: 10, SlotID: 2, SlotOrder: order(1), UpdatedAt: "2026-09-01 10:00:00"},
				{ChartID: 11, SlotID: 1, SlotOrder: nil, UpdatedAt: "2026-09-01 10:00:00"},
				{ChartID: 12, SlotID: 2, SlotOrder: order(2), UpdatedAt: "2026-09-01 10:00:00"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupPlayerBatchRepository(t)
			now := seedPlayerWithHonors(t, db, 1, false)
			_, err := db.Exec(`INSERT INTO player_records VALUES (1, 11, 999000, 1, 2, 2, '2026-09-01 10:00:00'), (1, 12, 990000, 1, 1, NULL, '2026-09-01 10:00:00')`)
			require.NoError(t, err)

			// When
			_, err = NewPlayerDataBatchRepository(db).ProcessPlayer(context.Background(), domainrepo.PlayerBatchKey{ID: 1, DataCollectedAt: &now}, func(domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error) {
				update := tt.update
				update.MasterFingerprint = masterfingerprint.Compute([]byte("master"))
				return update, nil
			})

			// Then
			require.NoError(t, err)
			var rows []recordRow
			require.NoError(t, db.Select(&rows, `SELECT chart_id, slot_id, slot_order, updated_at FROM player_records WHERE player_id = 1 ORDER BY chart_id`))
			assert.Equal(t, tt.expected, rows)
		})
	}
}

func TestPlayerBatchRecordQuery_ロック対象を成績行に限定する(t *testing.T) {
	t.Run("MySQLでは共有マスタのslotsをロックしない", func(t *testing.T) {
		assert.Contains(t, playerBatchRecordQuery("mysql"), "FOR UPDATE OF pr")
	})
	t.Run("SQLiteではロック句を付けない", func(t *testing.T) {
		assert.NotContains(t, playerBatchRecordQuery("sqlite"), "FOR UPDATE")
	})
}

func seedPlayerForBatchList(t *testing.T, db *sqlx.DB, playerID int, fingerprint any) {
	t.Helper()
	now := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO players (id, user_id, player_name, player_level, official_player_rating, possession_id, official_overpower, data_collected_at, created_at, updated_at, recalculated_master_fingerprint)
		VALUES (?, ?, 'プレイヤー', 30, 16.25, 1, 1234.5, ?, ?, ?, ?)`,
		playerID, 100+playerID, now, now, now, fingerprint)
	require.NoError(t, err)
}
