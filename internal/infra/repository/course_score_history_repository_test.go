package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCourseScoreHistoryTestDB(t *testing.T) *scoreHistoryRepository {
	t.Helper()
	db := setupTestDB(t)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err := db.Exec(`
		CREATE TABLE player_course_records (
			player_id INTEGER NOT NULL, course_id INTEGER NOT NULL, score INTEGER NOT NULL,
			is_clear INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL, updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, course_id)
		);
		CREATE TABLE player_course_record_histories (
			player_id INTEGER NOT NULL, course_id INTEGER NOT NULL, score INTEGER NOT NULL,
			is_clear INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL, updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (player_id, course_id, updated_at)
		);
	`)
	require.NoError(t, err)
	return &scoreHistoryRepository{db: db}
}

func courseHistoryRows(playerID, courseID, count int, base time.Time) []domainrepo.PlayerCourseRecordHistory {
	rows := make([]domainrepo.PlayerCourseRecordHistory, count)
	for i := range rows {
		rows[i] = domainrepo.PlayerCourseRecordHistory{PlayerID: playerID, CourseID: courseID, State: domainrepo.CourseRecordState{
			Score: 2_900_000 + i, IsClear: i%2 == 1, ComboLampID: 2, UpdatedAt: base.Add(time.Duration(i) * time.Second),
		}}
	}
	return rows
}

func TestScoreHistoryRepository_FindCourseTimeline(t *testing.T) {
	base := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		// Given
		hasCurrent   bool
		historyCount int
		// Then
		want []entity.CourseScoreHistoryEntry
	}{
		{
			name: "現行値を先頭に過去の成績を新しい順で返す", hasCurrent: true, historyCount: 2,
			want: []entity.CourseScoreHistoryEntry{
				{Score: 3_010_000, IsClear: true, ComboLampID: 3, UpdatedAt: base.Add(-time.Hour)},
				{Score: 2_900_001, IsClear: true, ComboLampID: 2, UpdatedAt: base.Add(time.Second)},
				{Score: 2_900_000, IsClear: false, ComboLampID: 2, UpdatedAt: base},
			},
		},
		{name: "現行値がなければ履歴を返さない", hasCurrent: false, historyCount: 2, want: []entity.CourseScoreHistoryEntry{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 現行値は履歴のどれよりも古い日時を持つ。他プレイヤー・他コースの行は結果に混ざらない。
			repo := setupCourseScoreHistoryTestDB(t)
			ctx := context.Background()
			if tt.hasCurrent {
				_, err := repo.db.Exec(`INSERT INTO player_course_records VALUES (1, 10, 3010000, 1, 3, ?)`, base.Add(-time.Hour))
				require.NoError(t, err)
			}
			_, err := repo.db.Exec(`INSERT INTO player_course_records VALUES (2, 10, 1000000, 0, 1, ?), (1, 11, 1000000, 0, 1, ?)`, base, base)
			require.NoError(t, err)
			require.NoError(t, repo.BulkInsertCourse(ctx, repo.db, courseHistoryRows(1, 10, tt.historyCount, base)))
			require.NoError(t, repo.BulkInsertCourse(ctx, repo.db, courseHistoryRows(2, 10, 1, base)))
			require.NoError(t, repo.BulkInsertCourse(ctx, repo.db, courseHistoryRows(1, 11, 1, base)))

			// When
			entries, err := repo.FindCourseTimeline(ctx, 1, 10)

			// Then
			require.NoError(t, err)
			require.Len(t, entries, len(tt.want))
			for i, want := range tt.want {
				assert.Equal(t, want.Score, entries[i].Score)
				assert.Equal(t, want.IsClear, entries[i].IsClear)
				assert.Equal(t, want.ComboLampID, entries[i].ComboLampID)
				assert.True(t, want.UpdatedAt.Equal(entries[i].UpdatedAt), "%d件目の更新日時", i+1)
			}
		})
	}
}

func TestScoreHistoryRepository_FindCourseTimeline_上限を超えたら最も古い履歴を切り捨てる(t *testing.T) {
	// Given: 現行値1件と、上限より1件多い履歴
	repo := setupCourseScoreHistoryTestDB(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	_, err := repo.db.Exec(`INSERT INTO player_course_records VALUES (1, 10, 3010000, 1, 3, ?)`, base.Add(-time.Hour))
	require.NoError(t, err)
	require.NoError(t, repo.BulkInsertCourse(ctx, repo.db, courseHistoryRows(1, 10, info.MaxScoreHistoryEntriesPerChart+1, base)))

	// When
	entries, err := repo.FindCourseTimeline(ctx, 1, 10)

	// Then: 現行値と新しい履歴50件だけが返る。
	require.NoError(t, err)
	require.Len(t, entries, info.MaxScoreHistoryEntriesPerChart+1)
	assert.Equal(t, 3_010_000, entries[0].Score)
	assert.Equal(t, 2_900_000+info.MaxScoreHistoryEntriesPerChart, entries[1].Score)
	assert.Equal(t, 2_900_001, entries[info.MaxScoreHistoryEntriesPerChart].Score)
}

func TestScoreHistoryRepository_PruneCourseOverLimit(t *testing.T) {
	// Given: 対象コースは上限を2件超え、他コースと他プレイヤーも上限を超えている。
	repo := setupCourseScoreHistoryTestDB(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	over := info.MaxScoreHistoryEntriesPerChart + 2
	for _, pair := range [][2]int{{1, 10}, {1, 11}, {2, 10}} {
		require.NoError(t, repo.BulkInsertCourse(ctx, repo.db, courseHistoryRows(pair[0], pair[1], over, base)))
	}

	// When
	err := repo.PruneCourseOverLimit(ctx, repo.db, 1, []int{10})

	// Then: 指定したプレイヤー・コースだけ古い順に削除される。
	require.NoError(t, err)
	type summary struct {
		PlayerID int `db:"player_id"`
		CourseID int `db:"course_id"`
		Count    int `db:"count"`
		MinScore int `db:"min_score"`
	}
	var got []summary
	require.NoError(t, repo.db.Select(&got, `SELECT player_id, course_id, COUNT(*) AS count, MIN(score) AS min_score
		FROM player_course_record_histories GROUP BY player_id, course_id ORDER BY player_id, course_id`))
	assert.Equal(t, []summary{
		{1, 10, info.MaxScoreHistoryEntriesPerChart, 2_900_002},
		{1, 11, over, 2_900_000},
		{2, 10, over, 2_900_000},
	}, got)
}

func TestCourseScoreHistoryMySQL_保存と上限整理とプレイヤー削除(t *testing.T) {
	// Given: 本番マイグレーションを適用したDB
	db := mysqltest.NewDatabase(t, "")
	_, err := db.Exec(`INSERT INTO users (id, username) VALUES (1, 'course_history_one'), (2, 'course_history_two');
		INSERT INTO players (id, user_id, player_name, player_level, official_player_rating, official_overpower, official_overpower_percent, data_collected_at)
			VALUES (1, 1, '対象', 10, 16.25, 12345.67, 98.76, '2026-11-01 00:08:20'), (2, 2, '他人', 20, 15.00, 10000.00, NULL, '2026-11-01 00:08:20');
		INSERT INTO courses (id, display_id, official_idx, name, course_class_id) VALUES
			(10, '0000000000000010', 'c10', 'コース10', 1), (11, '0000000000000011', 'c11', 'コース11', 1);`)
	require.NoError(t, err)
	ctx := context.Background()
	repo := NewScoreHistoryRepository(db)
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	over := info.MaxScoreHistoryEntriesPerChart + 1
	for _, pair := range [][2]int{{1, 10}, {1, 11}, {2, 10}} {
		require.NoError(t, repo.BulkInsertCourse(ctx, db, courseHistoryRows(pair[0], pair[1], over, base)))
	}

	// When / Then: 同一コース・同一秒の履歴はドメインの競合エラーになる。
	err = repo.BulkInsertCourse(ctx, db, courseHistoryRows(1, 10, 1, base))
	require.ErrorIs(t, err, domainrepo.ErrScoreHistoryTimestampConflict)

	// When / Then: 理論値を超えるスコアはCHECK制約で拒否される。
	invalid := courseHistoryRows(1, 10, 1, base.Add(time.Hour))
	invalid[0].State.Score = 3_030_001
	require.Error(t, repo.BulkInsertCourse(ctx, db, invalid))

	// When: 対象プレイヤーの1コースだけ上限整理する。
	require.NoError(t, repo.PruneCourseOverLimit(ctx, db, 1, []int{10}))

	// Then: DELETE JOINは別コース・別プレイヤーの同時刻の履歴を削除しない。
	type summary struct {
		PlayerID int `db:"player_id"`
		CourseID int `db:"course_id"`
		Count    int `db:"count"`
		MinScore int `db:"min_score"`
	}
	const summaryQuery = `SELECT player_id, course_id, COUNT(*) AS count, MIN(score) AS min_score
		FROM player_course_record_histories GROUP BY player_id, course_id ORDER BY player_id, course_id`
	var got []summary
	require.NoError(t, db.Select(&got, summaryQuery))
	assert.Equal(t, []summary{
		{1, 10, info.MaxScoreHistoryEntriesPerChart, 2_900_001},
		{1, 11, over, 2_900_000},
		{2, 10, over, 2_900_000},
	}, got)

	// When: プレイヤーを削除する。
	_, err = db.Exec(`UPDATE users SET player_id = NULL WHERE id = 1; DELETE FROM players WHERE id = 1`)
	require.NoError(t, err)

	// Then: そのプレイヤーの履歴だけがCASCADEで消える。
	require.NoError(t, db.Select(&got, summaryQuery))
	assert.Equal(t, []summary{{2, 10, over, 2_900_000}}, got)
}
