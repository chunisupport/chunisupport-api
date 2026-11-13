package repository

import (
	"context"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本番と同じON UPDATEを持つMySQLでも、SQLiteと共通の保存契約を検証します。
func TestSavePlayerData_差分書き込み(t *testing.T) {
	t.Run("SQLite", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()
		testRecordWrites(t, db)
	})
	t.Run("MySQL", func(t *testing.T) {
		db := mysqltest.NewDatabase(t, "")
		db.SetMaxOpenConns(1)
		_, err := db.Exec("SET timestamp = ?", time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC).Unix())
		require.NoError(t, err)
		testRecordWrites(t, db)
	})
}

func setupRecordWriteTables(t *testing.T, db *sqlx.DB) {
	t.Helper()
	// 他のマスタに依存しない最小テーブルで、書き込み内容とトランザクション境界を検証します。
	_, err := db.Exec(`DROP TABLE IF EXISTS player_records; DROP TABLE IF EXISTS player_worldsend_records;
 CREATE TABLE player_records (
 player_id INTEGER NOT NULL, chart_id INTEGER NOT NULL, score INTEGER NOT NULL,
 clear_lamp_id INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL, full_chain_id INTEGER NOT NULL,
 slot_id INTEGER NOT NULL, slot_order INTEGER, updated_at TIMESTAMP NOT NULL,
 PRIMARY KEY(player_id,chart_id));
 CREATE TABLE player_worldsend_records (
 player_id INTEGER NOT NULL, worldsend_chart_id INTEGER NOT NULL, score INTEGER NOT NULL,
 clear_lamp_id INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL, full_chain_id INTEGER NOT NULL,
 updated_at TIMESTAMP NOT NULL, PRIMARY KEY(player_id,worldsend_chart_id));`)
	require.NoError(t, err)
	if db.DriverName() == "mysql" {
		_, err = db.Exec(`ALTER TABLE player_records MODIFY updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
  ALTER TABLE player_worldsend_records MODIFY updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;`)
		require.NoError(t, err)
	}
}

func testRecordWrites(t *testing.T, db *sqlx.DB) {
	t.Helper()
	setupRecordWriteTables(t, db)
	ctx := context.Background()
	repo := NewPlayerDataRepository(db)
	before := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	after := before.Add(time.Hour)
	full := []domainrepo.PlayerRecordForUpsert{}
	we := []domainrepo.WorldsendRecordForUpsert{}
	for _, id := range []int{101, 102, 103, 104} {
		full = append(full, domainrepo.PlayerRecordForUpsert{PlayerID: 10, ChartID: id, State: domainrepo.PlayerRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, SlotID: 2, SlotOrder: new(1), UpdatedAt: before}})
	}
	other := full[0]
	other.PlayerID = 20
	full = append(full, other)
	for _, id := range []int{201, 202} {
		we = append(we, domainrepo.WorldsendRecordForUpsert{PlayerID: 10, ChartID: id, State: domainrepo.WorldsendRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, UpdatedAt: before}})
	}
	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, repo.SavePlayerData(ctx, tx, domainrepo.PlayerDataSaveInput{FullRecordsToInsert: full, WorldsendRecordsToInsert: we}))
	require.NoError(t, tx.Commit())
	states, err := repo.FindPlayerRecordStatesByChartIDs(ctx, db, 10, []int{101, 102, 103, 104})
	require.NoError(t, err)
	for _, r := range full[:4] {
		assert.Equal(t, r.State, states[r.ChartID])
	}
	wes, err := repo.FindWorldsendRecordStatesByChartIDs(ctx, db, 10, []int{201, 202})
	require.NoError(t, err)
	for _, r := range we {
		assert.Equal(t, r.State, wes[r.ChartID])
	}
	changed := full[0]
	changed.State = domainrepo.PlayerRecordState{Score: 1005000, ClearLampID: 2, ComboLampID: 3, FullChainID: 2, SlotID: 3, SlotOrder: new(2), UpdatedAt: after}
	changedWE := we[0]
	changedWE.State = domainrepo.WorldsendRecordState{Score: 1005000, ClearLampID: 2, ComboLampID: 3, FullChainID: 2, UpdatedAt: after}
	input := domainrepo.PlayerDataSaveInput{FullRecordsToUpdate: []domainrepo.PlayerRecordForUpsert{changed}, WorldsendRecordsToUpdate: []domainrepo.WorldsendRecordForUpsert{changedWE}, FullRecordSlotsToUpdate: []domainrepo.PlayerRecordSlotUpdate{{PlayerID: 10, ChartID: 102, SlotID: 4, SlotOrder: new(3)}, {PlayerID: 10, ChartID: 103, SlotID: 1}}}
	tx, err = db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, repo.SavePlayerData(ctx, tx, input))
	require.NoError(t, tx.Commit())
	states, err = repo.FindPlayerRecordStatesByChartIDs(ctx, db, 10, []int{101, 102, 103, 104})
	require.NoError(t, err)
	assert.Equal(t, changed.State, states[101])
	wantSlot := full[1].State
	wantSlot.SlotID = 4
	wantSlot.SlotOrder = new(3)
	assert.Equal(t, wantSlot, states[102])
	wantClear := full[2].State
	wantClear.SlotID = 1
	wantClear.SlotOrder = nil
	assert.Equal(t, wantClear, states[103])
	assert.Equal(t, full[3].State, states[104])
	otherStates, err := repo.FindPlayerRecordStatesByChartIDs(ctx, db, 20, []int{101})
	require.NoError(t, err)
	assert.Equal(t, other.State, otherStates[101])
	wes, err = repo.FindWorldsendRecordStatesByChartIDs(ctx, db, 10, []int{201, 202})
	require.NoError(t, err)
	assert.Equal(t, changedWE.State, wes[201])
	assert.Equal(t, we[1].State, wes[202])
	// 先行INSERTの後、後続UPDATEが失敗しても呼び出し元のRollbackで部分反映が残りません。
	_, err = db.Exec("DROP TABLE player_worldsend_records")
	require.NoError(t, err)
	tx, err = db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	newRow := full[0]
	newRow.ChartID = 999
	err = repo.SavePlayerData(ctx, tx, domainrepo.PlayerDataSaveInput{FullRecordsToInsert: []domainrepo.PlayerRecordForUpsert{newRow}, WorldsendRecordsToUpdate: []domainrepo.WorldsendRecordForUpsert{changedWE}})
	require.Error(t, err)
	require.NoError(t, tx.Rollback())
	var count int
	require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM player_records WHERE chart_id = 999"))
	assert.Zero(t, count)
	// 空入力ならテーブルが存在しなくてもSQLを発行しません。
	_, err = db.Exec("DROP TABLE player_records")
	require.NoError(t, err)
	require.NoError(t, repo.SavePlayerData(ctx, db, domainrepo.PlayerDataSaveInput{}))
}

func TestSavePlayerDataMySQL_チャンク境界(t *testing.T) {
	db := mysqltest.NewDatabase(t, "")
	setupRecordWriteTables(t, db)
	ctx := context.Background()
	repo := NewPlayerDataRepository(db)
	full := make([]domainrepo.PlayerRecordForUpsert, info.BulkInsertChunkSize+1)
	we := make([]domainrepo.WorldsendRecordForUpsert, len(full))
	slots := make([]domainrepo.PlayerRecordSlotUpdate, len(full))
	for i := range full {
		full[i] = domainrepo.PlayerRecordForUpsert{PlayerID: 1, ChartID: i + 1, State: domainrepo.PlayerRecordState{Score: 1, SlotID: 1, UpdatedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}}
		we[i] = domainrepo.WorldsendRecordForUpsert{PlayerID: 1, ChartID: i + 1, State: domainrepo.WorldsendRecordState{Score: 1, UpdatedAt: full[i].State.UpdatedAt}}
		slots[i] = domainrepo.PlayerRecordSlotUpdate{PlayerID: 1, ChartID: i + 1, SlotID: 3, SlotOrder: new(i + 1)}
	}
	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, repo.SavePlayerData(ctx, tx, domainrepo.PlayerDataSaveInput{FullRecordsToInsert: full, WorldsendRecordsToInsert: we}))
	for i := range full {
		full[i].State.Score = 2
		we[i].State.Score = 2
	}
	require.NoError(t, repo.SavePlayerData(ctx, tx, domainrepo.PlayerDataSaveInput{FullRecordsToUpdate: full, WorldsendRecordsToUpdate: we}))
	require.NoError(t, repo.SavePlayerData(ctx, tx, domainrepo.PlayerDataSaveInput{FullRecordSlotsToUpdate: slots}))
	require.NoError(t, tx.Commit())
	var count int
	require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM player_records WHERE score=2 AND slot_id=3 AND slot_order=chart_id"))
	assert.Equal(t, len(full), count)
	require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM player_worldsend_records WHERE score=2"))
	assert.Equal(t, len(we), count)
}
