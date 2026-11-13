package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDataTransferRepository_移行先の非空条件を独立に拒否する(t *testing.T) {
	for _, tt := range []struct {
		name      string
		seed      string
		wantEmpty bool
	}{
		{"他人のデータだけなら空", "SELECT 1", true},
		{"player_idの関連付け", "UPDATE users SET player_id = 77 WHERE id = 1", false},
		{"関連付け前の所有プレイヤー", "INSERT INTO players (id, user_id, player_name) VALUES (88, 1, '既存プレイヤー')", false},
		{"目標だけが存在", "INSERT INTO goals (id, user_id, title) VALUES (88, 1, '既存目標')", false},
		{"目標グループだけが存在", "INSERT INTO goal_groups (id, user_id, name) VALUES (88, 1, '既存グループ')", false},
		{"フィルタだけが存在", "INSERT INTO record_filters (id, user_id, name) VALUES (x'88', 1, '既存フィルタ')", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 他人のデータは空判定の対象外。各行で本人の条件を1つだけ成立させる。
			db := newTransferRepositoryTestDB(t)
			_, err := db.Exec(`INSERT INTO users (id, player_id) VALUES (1, NULL), (2, NULL);
				INSERT INTO players (id, user_id, player_name) VALUES (77, 2, '他人のプレイヤー');
				INSERT INTO goals (id, user_id, title) VALUES (77, 2, '他人の目標');
				INSERT INTO goal_groups (id, user_id, name) VALUES (77, 2, '他人のグループ');
				INSERT INTO record_filters (id, user_id, name) VALUES (x'77', 2, '他人のフィルタ');`)
			require.NoError(t, err)
			_, err = db.Exec(tt.seed)
			require.NoError(t, err)
			const stateQuery = `SELECT 'user:' || id || ':' || COALESCE(player_id, 'NULL') AS state FROM users
				UNION ALL SELECT 'player:' || id || ':' || user_id || ':' || player_name FROM players
				UNION ALL SELECT 'goal:' || id || ':' || user_id || ':' || title FROM goals
				UNION ALL SELECT 'group:' || id || ':' || user_id || ':' || name FROM goal_groups
				UNION ALL SELECT 'filter:' || hex(id) || ':' || user_id || ':' || name FROM record_filters
				ORDER BY state`
			var before []string
			require.NoError(t, db.Select(&before, stateQuery))
			repo := NewUserDataTransferRepository(db)

			// When
			empty, err := repo.IsDestinationEmpty(context.Background(), 1)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEmpty, empty)
			playerID, err := repo.ImportSnapshot(context.Background(), 1, emptyTransferRepositorySnapshot(t))

			// Then
			if tt.wantEmpty {
				require.NoError(t, err)
				assert.Positive(t, playerID)
				return
			}
			require.ErrorIs(t, err, usecase.ErrDataTransferDestinationNotEmpty)
			assert.Zero(t, playerID)
			var after []string
			require.NoError(t, db.Select(&after, stateQuery))
			assert.Equal(t, before, after, "拒否後も既存の関連付け・所有データを変更しない")
		})
	}
}

func TestUserDataTransferRepository_記録と履歴を内部IDへ解決して全件インポートする(t *testing.T) {
	for _, count := range []int{3000, 3001} {
		t.Run(fmt.Sprintf("履歴%d件", count), func(t *testing.T) {
			// Given: 外向きの識別子とDB内のID、各マスターのIDを意図的に異ならせる。
			db := newTransferRepositoryTestDB(t)
			_, err := db.Exec(`INSERT INTO users (id, player_id) VALUES (1, NULL);
				INSERT INTO songs (id, official_idx, is_deleted) VALUES (11, '100', 0), (12, '200', 0), (13, '300', 0), (14, '400', 0);
				INSERT INTO difficulties (id, name, sort_order) VALUES (4, 'MASTER', 4), (3, 'EXPERT', 3);
				INSERT INTO charts (id, song_id, difficulty_id, const) VALUES (101, 11, 4, 14.5), (102, 12, 3, 13.0);
				INSERT INTO worldsend_charts (id, song_id) VALUES (201, 13), (202, 14);
				INSERT INTO clear_lamp_types (id, name) VALUES (5, 'ABSOLUTE'), (4, 'HARD');
				INSERT INTO combo_lamp_types (id, name) VALUES (3, 'ALL JUSTICE'), (2, 'FULL COMBO');
				INSERT INTO full_chain_types (id, name) VALUES (2, 'FULL CHAIN'), (1, 'NONE');
				INSERT INTO slots (id, name) VALUES (8, 'best'), (9, 'none');
				CREATE UNIQUE INDEX history_key ON player_record_histories (player_id, chart_id, updated_at);
				CREATE UNIQUE INDEX worldsend_history_key ON player_worldsend_record_histories (player_id, worldsend_chart_id, updated_at);`)
			require.NoError(t, err)
			// 各譜面50件の履歴上限を守ったスナップショットでチャンク末尾を踏む。
			for group := range (count + 49) / 50 {
				_, err := db.Exec(`INSERT INTO songs (id, official_idx, is_deleted) VALUES (?, ?, 0), (?, ?, 0)`,
					1000+group, fmt.Sprint(10000+group), 2000+group, fmt.Sprint(20000+group))
				require.NoError(t, err)
				_, err = db.Exec(`INSERT INTO charts (id, song_id, difficulty_id, const) VALUES (?, ?, 3, 13.0)`, 1000+group, 1000+group)
				require.NoError(t, err)
				_, err = db.Exec(`INSERT INTO worldsend_charts (id, song_id) VALUES (?, ?)`, 2000+group, 2000+group)
				require.NoError(t, err)
			}
			snapshot := emptyTransferRepositorySnapshot(t)
			currentAt := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
			historyAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
			order := 7
			currentScore, err := score.NewScore(1001234)
			require.NoError(t, err)
			worldsendScore, err := score.NewScore(987654)
			require.NoError(t, err)
			snapshot.Records = []entity.UserDataTransferRecord{{SongOfficialIdx: "100", Difficulty: "MASTER", Score: currentScore,
				ClearLampName: "ABSOLUTE", ComboLampName: "ALL JUSTICE", FullChainName: "FULL CHAIN", SlotName: "best", SlotOrder: &order, UpdatedAt: currentAt},
				{SongOfficialIdx: "200", Difficulty: "EXPERT", Score: worldsendScore,
					ClearLampName: "HARD", ComboLampName: "FULL COMBO", FullChainName: "NONE", SlotName: "none", UpdatedAt: currentAt.Add(time.Hour)}}
			snapshot.WorldsendRecords = []entity.UserDataTransferWorldsendRecord{{SongOfficialIdx: "300", Score: worldsendScore,
				ClearLampName: "HARD", ComboLampName: "FULL COMBO", FullChainName: "NONE", UpdatedAt: currentAt.Add(2 * time.Hour)}}
			for i := range count {
				historyScore, err := score.NewScore(uint32(900000 + i))
				require.NoError(t, err)
				worldsendHistoryScore, err := score.NewScore(uint32(1000000 + i))
				require.NoError(t, err)
				snapshot.RecordHistories = append(snapshot.RecordHistories, entity.UserDataTransferRecordHistory{
					SongOfficialIdx: fmt.Sprint(10000 + i/50), Difficulty: "EXPERT", Score: historyScore, ClearLampName: "HARD", ComboLampName: "FULL COMBO", FullChainName: "NONE", UpdatedAt: historyAt.Add(time.Duration(i) * time.Second)})
				snapshot.WorldsendRecordHistories = append(snapshot.WorldsendRecordHistories, entity.UserDataTransferWorldsendRecordHistory{
					SongOfficialIdx: fmt.Sprint(20000 + i/50), Score: worldsendHistoryScore, ClearLampName: "ABSOLUTE", ComboLampName: "ALL JUSTICE", FullChainName: "FULL CHAIN", UpdatedAt: historyAt.Add(time.Duration(i) * time.Minute)})
			}

			require.NoError(t, snapshot.Validate())

			// When: 非公開の保存部品ではなく、トランザクションを開始する公開メソッドを呼ぶ。
			playerID, err := NewUserDataTransferRepository(db).ImportSnapshot(context.Background(), 1, snapshot)

			// Then: Exportによる往復比較には頼らず、保存列を直接確認する。
			require.NoError(t, err)
			require.Positive(t, playerID)
			var linkedID int
			require.NoError(t, db.Get(&linkedID, `SELECT player_id FROM users WHERE id = 1`))
			assert.Equal(t, playerID, linkedID)
			type storedRecord struct {
				PlayerID    int       `db:"player_id"`
				ChartID     int       `db:"chart_id"`
				Score       int       `db:"score"`
				ClearLampID int       `db:"clear_lamp_id"`
				ComboLampID int       `db:"combo_lamp_id"`
				FullChainID int       `db:"full_chain_id"`
				SlotID      int       `db:"slot_id"`
				SlotOrder   *int      `db:"slot_order"`
				UpdatedAt   time.Time `db:"updated_at"`
			}
			var current []storedRecord
			require.NoError(t, db.Select(&current, `SELECT player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, slot_id, slot_order, updated_at FROM player_records ORDER BY chart_id`))
			assert.Equal(t, []storedRecord{
				{playerID, 101, 1001234, 5, 3, 2, 8, &order, currentAt},
				{playerID, 102, 987654, 4, 2, 1, 9, nil, currentAt.Add(time.Hour)},
			}, current)
			var worldsend []storedRecord
			require.NoError(t, db.Select(&worldsend, `SELECT player_id, worldsend_chart_id AS chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at FROM player_worldsend_records`))
			assert.Equal(t, []storedRecord{{PlayerID: playerID, ChartID: 201, Score: 987654, ClearLampID: 4, ComboLampID: 2, FullChainID: 1, UpdatedAt: currentAt.Add(2 * time.Hour)}}, worldsend)
			for _, tt := range []struct {
				table  string
				column string
				want   storedRecord
			}{
				{"player_record_histories", "chart_id", storedRecord{PlayerID: playerID, ChartID: 1000 + (count-1)/50, Score: 899999 + count, ClearLampID: 4, ComboLampID: 2, FullChainID: 1, UpdatedAt: historyAt.Add(time.Duration(count-1) * time.Second)}},
				{"player_worldsend_record_histories", "worldsend_chart_id", storedRecord{PlayerID: playerID, ChartID: 2000 + (count-1)/50, Score: 999999 + count, ClearLampID: 5, ComboLampID: 3, FullChainID: 2, UpdatedAt: historyAt.Add(time.Duration(count-1) * time.Minute)}},
			} {
				t.Run(tt.table, func(t *testing.T) {
					var actualCount int
					require.NoError(t, db.Get(&actualCount, "SELECT COUNT(*) FROM "+tt.table))
					assert.Equal(t, count, actualCount)
					var last storedRecord
					require.NoError(t, db.Get(&last, "SELECT player_id, "+tt.column+" AS chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at FROM "+tt.table+" ORDER BY updated_at DESC LIMIT 1"))
					assert.Equal(t, tt.want, last)
				})
			}
		})
	}
}
