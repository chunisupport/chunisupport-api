package repository

import (
	"context"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryPruneMySQL_上限と対象範囲(t *testing.T) {
	db := mysqltest.NewDatabase(t, "")
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO users (id, username) VALUES (1, 'history_one'), (2, 'history_two');
		INSERT INTO players (id, user_id, player_name, player_level, official_player_rating, official_overpower, official_overpower_percent, data_collected_at)
			VALUES (1, 1, '対象', 10, 16.25, 12345.67, 98.76, '2026-11-01 00:08:20'), (2, 2, '他人', 20, 15.00, 10000.00, NULL, '2026-11-01 00:08:20');
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend) VALUES
			(11, '0000000000000011', '通常1', '作者', 1, '11', 0), (12, '0000000000000012', '通常2', '作者', 1, '12', 0),
			(13, '0000000000000013', 'WE1', '作者', 1, '13', 1), (14, '0000000000000014', 'WE2', '作者', 1, '14', 1);
		INSERT INTO charts (id, song_id, difficulty_id, const) VALUES (101, 11, 4, 14.0), (102, 12, 4, 14.5);
		INSERT INTO worldsend_charts (id, song_id) VALUES (101, 13), (102, 14);`)
	require.NoError(t, err)
	ctx := context.Background()

	for _, variant := range historyBulkVariants() {
		t.Run(variant.name+"のDELETE_JOIN", func(t *testing.T) {
			tx, err := db.Beginx()
			require.NoError(t, err)
			defer tx.Rollback()
			repo := NewScoreHistoryRepository(db)
			for _, pair := range [][2]int{{1, 101}, {1, 102}, {2, 101}} {
				rows := historyBulkRows(51)
				for i := range rows {
					rows[i].PlayerID, rows[i].ChartID = pair[0], pair[1]
				}
				require.NoError(t, variant.insert(ctx, repo, tx, rows))
			}
			// 本番INSERTの主キー衝突からドメインエラーとMySQLの原因へ到達できる。
			duplicate := historyBulkRows(1)
			duplicate[0].PlayerID, duplicate[0].ChartID = 1, 101
			err = variant.insert(ctx, repo, tx, duplicate)
			require.ErrorIs(t, err, domainrepo.ErrScoreHistoryTimestampConflict)
			var mysqlErr *mysql.MySQLError
			require.ErrorAs(t, err, &mysqlErr)
			assert.Equal(t, uint16(1062), mysqlErr.Number)

			prune := repo.PruneStandardOverLimit
			if variant.name == "WORLDSEND" {
				prune = repo.PruneWorldsendOverLimit
			}

			// 51件から50件へ。別譜面・別プレイヤーの同時刻の履歴は削除しない。
			require.NoError(t, prune(ctx, tx, 1, []int{101}))
			type summary struct {
				PlayerID int `db:"player_id"`
				ChartID  int `db:"chart_id"`
				Count    int `db:"count"`
				MinScore int `db:"min_score"`
				MaxScore int `db:"max_score"`
			}
			query := "SELECT player_id, " + variant.chartColumn + " AS chart_id, COUNT(*) AS count, MIN(score) AS min_score, MAX(score) AS max_score FROM " + variant.table + " GROUP BY player_id, " + variant.chartColumn + " ORDER BY player_id, chart_id"
			var got []summary
			require.NoError(t, tx.Select(&got, query))
			assert.Equal(t, []summary{{1, 101, 50, 900001, 900050}, {1, 102, 51, 900000, 900050}, {2, 101, 51, 900000, 900050}}, got)

			// 既に50件の譜面は減らさず、追加指定した譜面だけを上限へ収める。
			require.NoError(t, prune(ctx, tx, 1, []int{101, 102}))
			require.NoError(t, tx.Select(&got, query))
			assert.Equal(t, []summary{{1, 101, 50, 900001, 900050}, {1, 102, 50, 900001, 900050}, {2, 101, 51, 900000, 900050}}, got)
		})
	}

	t.Run("PlayerRepository_Save経由で指標履歴を500件に保つ", func(t *testing.T) {
		tx, err := db.Beginx()
		require.NoError(t, err)
		defer tx.Rollback()
		// 他人にも同時刻の履歴を置き、DELETE JOINのplayer_id条件を検証する。
		for i := range 501 {
			_, err := tx.Exec(`INSERT INTO player_metric_histories
				(player_id, official_rating, official_overpower, official_overpower_percent, data_collected_at)
				VALUES (2, 15.00, 10000.00, NULL, ?)`, base.Add(time.Duration(i)*time.Second))
			require.NoError(t, err)
			if i < 500 {
				_, err = tx.Exec(`INSERT INTO player_metric_histories
					(player_id, official_rating, official_overpower, official_overpower_percent, data_collected_at)
					VALUES (1, 16.00, 12000.00, 97.00, ?)`, base.Add(time.Duration(i)*time.Second))
				require.NoError(t, err)
			}
		}
		repo := NewPlayerRepository(db)
		player, err := repo.FindByID(ctx, tx, 1)
		require.NoError(t, err)
		require.NotNil(t, player)
		require.NoError(t, player.ChangeOfficialMetrics(16.50, 13000, 99, base.Add(501*time.Second)))

		require.NoError(t, repo.Save(ctx, tx, player))

		var dates []time.Time
		require.NoError(t, tx.Select(&dates, `SELECT data_collected_at FROM player_metric_histories WHERE player_id = 1 ORDER BY data_collected_at`))
		require.Len(t, dates, 500)
		assert.Equal(t, base.Add(time.Second), dates[0])
		assert.Equal(t, base.Add(500*time.Second), dates[499])
		var otherDates []time.Time
		require.NoError(t, tx.Select(&otherDates, `SELECT data_collected_at FROM player_metric_histories WHERE player_id = 2 ORDER BY data_collected_at`))
		require.Len(t, otherDates, 501)
		assert.Equal(t, base, otherDates[0])
		assert.Equal(t, base.Add(500*time.Second), otherDates[500])
		var stored struct {
			Rating    float64 `db:"official_rating"`
			Overpower float64 `db:"official_overpower"`
			Percent   float64 `db:"official_overpower_percent"`
		}
		require.NoError(t, tx.Get(&stored, `SELECT official_rating, official_overpower, official_overpower_percent FROM player_metric_histories WHERE player_id = 1 AND data_collected_at = ?`, base.Add(500*time.Second)))
		assert.Equal(t, 16.25, stored.Rating)
		assert.Equal(t, 12345.67, stored.Overpower)
		assert.Equal(t, 98.76, stored.Percent)
		saved, err := repo.FindByID(ctx, tx, 1)
		require.NoError(t, err)
		require.NotNil(t, saved)
		require.NotNil(t, saved.DataCollectedAt)
		assert.Equal(t, 16.50, saved.OfficialRating)
		assert.Equal(t, base.Add(501*time.Second), *saved.DataCollectedAt)
	})
}
