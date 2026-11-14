package repository

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/masterfingerprint"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerDataMySQL_枠だけの変更は最終更新日時を進めない(t *testing.T) {
	type row struct {
		PlayerID  int       `db:"player_id"`
		ChartID   int       `db:"chart_id"`
		Score     int       `db:"score"`
		SlotID    int       `db:"slot_id"`
		Position  *int      `db:"slot_order"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	before := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name     string
		batch    bool
		expected []row
	}{
		{name: "登録時の枠解除", expected: []row{
			{1, 10, 1_000_000, 1, nil, before}, {1, 11, 990_000, 1, nil, before},
			{1, 12, 980_000, 1, nil, before}, {2, 10, 1_005_000, 2, new(1), before},
		}},
		{name: "再計算時の解除と割当", batch: true, expected: []row{
			{1, 10, 1_000_000, 1, nil, before}, {1, 11, 990_000, 4, new(2), before},
			{1, 12, 980_000, 2, new(3), before}, {2, 10, 1_005_000, 2, new(1), before},
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := setupPlayerPersistenceMySQL(t)
			// 接続の現在時刻を固定し、sleepなしでON UPDATEの発火を見分ける。
			db.SetMaxOpenConns(1)
			_, err := db.Exec("SET timestamp = ?", time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC).Unix())
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO charts (id,song_id,difficulty_id,const) VALUES (11,1,3,12),(12,1,2,9);
    INSERT INTO users (id,username) VALUES (21,'other_player');
    INSERT INTO players (id,user_id,player_name,player_level,official_player_rating,official_overpower,data_collected_at) VALUES (2,21,'別のプレイヤー',10,15.00,10000.00,'2026-09-01 10:00:00');
    INSERT INTO player_records (player_id,chart_id,score,slot_id,slot_order,updated_at) VALUES
     (1,11,990000,1,NULL,'2026-09-01 10:00:00'),(1,12,980000,2,3,'2026-09-01 10:00:00'),
     (2,10,1005000,2,1,'2026-09-01 10:00:00');`)
			require.NoError(t, err)
			ctx := context.Background()

			if tt.batch {
				player, err := NewPlayerRepository(db).FindByID(ctx, db, 1)
				require.NoError(t, err)
				status, err := NewPlayerDataBatchRepository(db).ProcessPlayer(ctx, domainrepo.PlayerBatchKey{ID: 1, DataCollectedAt: player.DataCollectedAt}, func(domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error) {
					return domainrepo.PlayerBatchUpdate{
						ClearChartIDs: []int{10}, Assignments: []domainrepo.PlayerBatchSlotAssignment{{ChartID: 11, SlotID: 4, Position: 2}},
						MasterFingerprint: masterfingerprint.Compute([]byte("slot-contract")),
					}, nil
				})
				require.NoError(t, err)
				assert.Equal(t, domainrepo.PlayerBatchUpdated, status)
			} else {
				tx, err := db.BeginTxx(ctx, nil)
				require.NoError(t, err)
				defer tx.Rollback()
				repo := NewPlayerDataRepository(db)
				ids, err := repo.FindRankedPlayerRecordChartIDs(ctx, tx, 1)
				require.NoError(t, err)
				assert.Equal(t, []int{10, 12}, ids)
				var slots []domainrepo.PlayerRecordSlotUpdate
				for _, id := range ids {
					slots = append(slots, domainrepo.PlayerRecordSlotUpdate{PlayerID: 1, ChartID: id, SlotID: 1})
				}
				require.NoError(t, repo.SavePlayerData(ctx, tx, domainrepo.PlayerDataSaveInput{FullRecordSlotsToUpdate: slots}))
				require.NoError(t, tx.Commit())
			}

			var actual []row
			require.NoError(t, db.Select(&actual, `SELECT player_id,chart_id,score,slot_id,slot_order,updated_at FROM player_records ORDER BY player_id,chart_id`))
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestChartStatsBatchJobMySQL_同値の保存も既存行として扱う(t *testing.T) {
	db := mysqltest.NewDatabase(t, "")
	repo := NewChartStatsBatchJobRepository(db)
	ctx := context.Background()
	startedAt := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	job := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), startedAt)
	require.NoError(t, repo.Save(ctx, job))

	// clientFoundRowsが無効なら、同値UPDATEを0件と誤認して主キー重複になる。
	require.NoError(t, repo.Save(ctx, job))
	actual, err := repo.FindByID(ctx, job.ID())
	require.NoError(t, err)
	assert.Equal(t, job, actual)
	require.NoError(t, job.Complete(startedAt.Add(time.Minute)))
	require.NoError(t, repo.Save(ctx, job))
	actual, err = repo.FindByID(ctx, job.ID())
	require.NoError(t, err)
	assert.Equal(t, job, actual)
	var count int
	require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM chart_stats_batch_jobs"))
	assert.Equal(t, 1, count)
}
