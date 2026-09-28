package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/chartstatsbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupChartStatsBatchRepositorySQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	// :memory: は接続ごとに別のDBになるため、トランザクションとテスト側の確認を同じ接続で行う
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec(`
		CREATE TABLE rating_bands (
			id INTEGER NOT NULL PRIMARY KEY,
			label TEXT NOT NULL,
			min_inclusive REAL NULL,
			max_exclusive REAL NULL,
			sort_order INTEGER NOT NULL
		);
		CREATE TABLE users (
			id INTEGER NOT NULL PRIMARY KEY,
			is_suspicious INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE players (
			id INTEGER NOT NULL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			best_average_rating REAL NULL
		);
		CREATE TABLE slots (
			id INTEGER NOT NULL PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE songs (
			id INTEGER NOT NULL PRIMARY KEY,
			is_deleted INTEGER NOT NULL DEFAULT 0,
			is_worldsend INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE charts (
			id INTEGER NOT NULL PRIMARY KEY,
			song_id INTEGER NOT NULL
		);
		CREATE TABLE worldsend_charts (
			id INTEGER NOT NULL PRIMARY KEY,
			song_id INTEGER NOT NULL
		);
		CREATE TABLE player_records (
			player_id INTEGER NOT NULL,
			chart_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			slot_id INTEGER NOT NULL,
			PRIMARY KEY (player_id, chart_id)
		);
		CREATE TABLE player_worldsend_records (
			player_id INTEGER NOT NULL,
			worldsend_chart_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			PRIMARY KEY (player_id, worldsend_chart_id)
		);
		CREATE TABLE chart_stats_by_rating_band (
			chart_id INTEGER NOT NULL,
			rating_band_id INTEGER NOT NULL,
			rank_aaal INTEGER NOT NULL, rank_s INTEGER NOT NULL, rank_sp INTEGER NOT NULL, rank_ss INTEGER NOT NULL,
			rank_ssp INTEGER NOT NULL, rank_sss INTEGER NOT NULL, rank_sssp INTEGER NOT NULL, rank_max INTEGER NOT NULL,
			combo_none INTEGER NOT NULL, combo_fc INTEGER NOT NULL, combo_aj INTEGER NOT NULL, combo_ajc INTEGER NOT NULL,
			clear_failed INTEGER NOT NULL, clear_clear INTEGER NOT NULL, clear_hard INTEGER NOT NULL,
			clear_brave INTEGER NOT NULL, clear_absolute INTEGER NOT NULL, clear_catastrophy INTEGER NOT NULL,
			average_score REAL NULL, median_score REAL NULL, player_count INTEGER NOT NULL,
			PRIMARY KEY (chart_id, rating_band_id)
		);
		CREATE TABLE worldsend_chart_stats_by_rating_band (
			worldsend_chart_id INTEGER NOT NULL,
			rating_band_id INTEGER NOT NULL,
			rank_aaal INTEGER NOT NULL, rank_s INTEGER NOT NULL, rank_sp INTEGER NOT NULL, rank_ss INTEGER NOT NULL,
			rank_ssp INTEGER NOT NULL, rank_sss INTEGER NOT NULL, rank_sssp INTEGER NOT NULL, rank_max INTEGER NOT NULL,
			combo_none INTEGER NOT NULL, combo_fc INTEGER NOT NULL, combo_aj INTEGER NOT NULL, combo_ajc INTEGER NOT NULL,
			clear_failed INTEGER NOT NULL, clear_clear INTEGER NOT NULL, clear_hard INTEGER NOT NULL,
			clear_brave INTEGER NOT NULL, clear_absolute INTEGER NOT NULL, clear_catastrophy INTEGER NOT NULL,
			average_score REAL NULL, median_score REAL NULL, player_count INTEGER NOT NULL,
			PRIMARY KEY (worldsend_chart_id, rating_band_id)
		);
		CREATE TABLE chart_best_slot_stats_by_rating_band (
			chart_id INTEGER NOT NULL,
			rating_band_id INTEGER NOT NULL,
			best_player_count INTEGER NOT NULL,
			eligible_player_count INTEGER NOT NULL,
			best_player_percentage REAL NULL CHECK (
				best_player_percentage IS NULL OR (best_player_percentage >= 0 AND best_player_percentage <= 100)
			),
			PRIMARY KEY (chart_id, rating_band_id)
		);
	`)
	require.NoError(t, err)
	return db
}

// chartStatsBatchSourceResult は StreamSource がコールバックへ渡した値です。
type chartStatsBatchSourceResult struct {
	chartRecords     []chartstatsbatch.ChartRecord
	worldsendRecords []chartstatsbatch.ChartRecord
	eligiblePlayers  []chartstatsbatch.EligiblePlayer
	bestSlotRecords  []chartstatsbatch.BestSlotRecord
	bestSlotCharts   []int
}

func streamChartStatsBatchSource(t *testing.T, repo domainrepo.ChartStatsBatchRepository) chartStatsBatchSourceResult {
	t.Helper()
	var result chartStatsBatchSourceResult
	err := repo.StreamSource(context.Background(), domainrepo.ChartStatsBatchSourceCallbacks{
		ChartRecord: func(record chartstatsbatch.ChartRecord) error {
			result.chartRecords = append(result.chartRecords, record)
			return nil
		},
		WorldsendChartRecord: func(record chartstatsbatch.ChartRecord) error {
			result.worldsendRecords = append(result.worldsendRecords, record)
			return nil
		},
		EligiblePlayer: func(player chartstatsbatch.EligiblePlayer) error {
			result.eligiblePlayers = append(result.eligiblePlayers, player)
			return nil
		},
		BestSlotRecord: func(record chartstatsbatch.BestSlotRecord) error {
			result.bestSlotRecords = append(result.bestSlotRecords, record)
			return nil
		},
		BestSlotChart: func(chartID int) error {
			result.bestSlotCharts = append(result.bestSlotCharts, chartID)
			return nil
		},
	})
	require.NoError(t, err)
	return result
}

func TestChartStatsBatchRepository_FindRatingBands(t *testing.T) {
	// Given
	db := setupChartStatsBatchRepositorySQLite(t)
	_, err := db.Exec(`
		INSERT INTO rating_bands (id, label, min_inclusive, max_exclusive, sort_order) VALUES
			(22, '17.00', 17.0, 17.1, 2),
			(0, 'ALL', NULL, NULL, 1)
	`)
	require.NoError(t, err)
	lower, upper := 17.0, 17.1

	// When
	bands, err := NewChartStatsBatchRepository(db).FindRatingBands(context.Background())

	// Then
	require.NoError(t, err)
	assert.Equal(t, []*ratingband.RatingBand{
		{ID: 0, Label: "ALL", SortOrder: 1},
		{ID: 22, Label: "17.00", MinInclusive: &lower, MaxExclusive: &upper, SortOrder: 2},
	}, bands)
}

func TestChartStatsBatchRepository_StreamSource(t *testing.T) {
	// Given
	db := setupChartStatsBatchRepositorySQLite(t)
	_, err := db.Exec(`
		INSERT INTO users (id, is_suspicious) VALUES (1, 0), (2, 0), (3, 1), (4, 0);
		-- プレイヤー3は不審ユーザー、プレイヤー4はベスト枠平均レーティングが未計算のため、どの統計でも集計対象外
		INSERT INTO players (id, user_id, best_average_rating) VALUES (1, 1, 17.0), (2, 2, 16.5), (3, 3, 17.2), (4, 4, NULL);
		INSERT INTO slots (id, name) VALUES (1, 'best'), (2, 'new'), (3, 'other');
		-- 楽曲2と楽曲4は削除済みのため、その譜面はどの統計でも集計対象外
		INSERT INTO songs (id, is_deleted, is_worldsend) VALUES (1, 0, 0), (2, 1, 0), (3, 0, 1), (4, 1, 1);
		INSERT INTO charts (id, song_id) VALUES (10, 1), (20, 2), (30, 3), (11, 1);
		INSERT INTO worldsend_charts (id, song_id) VALUES (5, 3), (6, 4);
		INSERT INTO player_records (player_id, chart_id, score, clear_lamp_id, combo_lamp_id, slot_id) VALUES
			(1, 20, 1000000, 2, 1, 2),
			(2, 11, 1000000, 2, 2, 1),
			(1, 11, 1005000, 3, 3, 2),
			(1, 10, 990000, 1, 1, 1),
			(3, 10, 1009000, 2, 1, 1),
			(4, 10, 1010000, 2, 3, 1);
		INSERT INTO player_worldsend_records (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id) VALUES
			(2, 5, 980000, 2, 1),
			(1, 5, 1010000, 6, 3),
			(3, 5, 1009000, 2, 1),
			(4, 5, 1000000, 2, 1),
			(1, 6, 1000000, 2, 1);
	`)
	require.NoError(t, err)

	// When
	result := streamChartStatsBatchSource(t, NewChartStatsBatchRepository(db))

	// Then: 譜面統計は不審ユーザーと削除済み楽曲を除外し、譜面ID順、同じ譜面内はプレイヤーID順に返す
	assert.Equal(t, []chartstatsbatch.ChartRecord{
		{ChartID: 10, BestAverageRating: 17.0, Score: 990000, ClearLampID: 1, ComboLampID: 1},
		{ChartID: 11, BestAverageRating: 17.0, Score: 1005000, ClearLampID: 3, ComboLampID: 3},
		{ChartID: 11, BestAverageRating: 16.5, Score: 1000000, ClearLampID: 2, ComboLampID: 2},
	}, result.chartRecords)
	assert.Equal(t, []chartstatsbatch.ChartRecord{
		{ChartID: 5, BestAverageRating: 17.0, Score: 1010000, ClearLampID: 6, ComboLampID: 3},
		{ChartID: 5, BestAverageRating: 16.5, Score: 980000, ClearLampID: 2, ComboLampID: 1},
	}, result.worldsendRecords)
	// Then: ベスト枠採用率は不審ユーザーを除外し、ベスト枠の記録だけを対象にする
	assert.ElementsMatch(t, []chartstatsbatch.EligiblePlayer{{BestAverageRating: 17.0}, {BestAverageRating: 16.5}}, result.eligiblePlayers)
	assert.ElementsMatch(t, []chartstatsbatch.BestSlotRecord{
		{ChartID: 10, BestAverageRating: 17.0},
		{ChartID: 11, BestAverageRating: 16.5},
	}, result.bestSlotRecords)
	// Then: 削除済み楽曲と WORLD'S END の譜面は採用率の対象外
	assert.Equal(t, []int{10, 11}, result.bestSlotCharts)
}

func TestChartStatsBatchRepository_ReplaceAll_統計テーブルを入れ替える(t *testing.T) {
	// Given: 前回の集計結果
	db := setupChartStatsBatchRepositorySQLite(t)
	_, err := db.Exec(`
		INSERT INTO chart_stats_by_rating_band VALUES (999, 0, 0,0,0,0,0,0,0,0, 0,0,0,0, 0,0,0,0,0,0, NULL, NULL, 0);
		INSERT INTO worldsend_chart_stats_by_rating_band VALUES (999, 0, 0,0,0,0,0,0,0,0, 0,0,0,0, 0,0,0,0,0,0, NULL, NULL, 0);
		INSERT INTO chart_best_slot_stats_by_rating_band VALUES (999, 0, 1, 1, 100);
	`)
	require.NoError(t, err)
	average, median, percentage := 1004000.5, 1004500.0, 25.0
	snapshot := &chartstatsbatch.Snapshot{
		ChartStats: []*entity.ChartStatsByRatingBand{{
			ChartID: 10, RatingBandID: 22,
			Rank:         entity.ChartRankStats{AAAL: 1, S: 2, SP: 3, SS: 4, SSP: 5, SSS: 6, SSSP: 7, Max: 8},
			Combo:        entity.ChartComboStats{None: 9, FC: 10, AJ: 11, AJC: 12},
			Clear:        entity.ChartClearStats{Failed: 13, Clear: 14, Hard: 15, Brave: 16, Absolute: 17, Catastrophy: 18},
			AverageScore: &average, MedianScore: &median, PlayerCount: 19,
		}},
		WorldsendChartStats: []*entity.ChartStatsByRatingBand{{ChartID: 5, RatingBandID: 0, Clear: entity.ChartClearStats{Failed: 1}, PlayerCount: 1}},
		BestSlotStats: []*entity.ChartBestSlotStatsByRatingBand{
			{ChartID: 10, RatingBandID: 0, BestPlayerCount: 1, EligiblePlayerCount: 4, BestPlayerPercentage: &percentage},
			{ChartID: 10, RatingBandID: 99},
		},
	}

	// When
	err = NewChartStatsBatchRepository(db).ReplaceAll(context.Background(), snapshot)

	// Then
	require.NoError(t, err)
	chartStats, err := NewChartStatsRepository(db).FindChartStatsByChartIDs(context.Background(), db, []int{10, 999})
	require.NoError(t, err)
	assert.Equal(t, snapshot.ChartStats, chartStats)
	worldsendStats, err := NewChartStatsRepository(db).FindWorldsendChartStatsByChartIDs(context.Background(), db, []int{5, 999})
	require.NoError(t, err)
	assert.Equal(t, snapshot.WorldsendChartStats, worldsendStats)
	bestSlotStats, err := NewChartStatsRepository(db).FindChartBestSlotStatsByChartIDs(context.Background(), db, []int{10, 999})
	require.NoError(t, err)
	assert.ElementsMatch(t, snapshot.BestSlotStats, bestSlotStats)
}

func TestChartStatsBatchRepository_ReplaceAll_挿入件数が1文の上限を超えても全件挿入する(t *testing.T) {
	// Given
	db := setupChartStatsBatchRepositorySQLite(t)
	stats := make([]*entity.ChartBestSlotStatsByRatingBand, 0, info.ChartStatsBatchInsertChunkSize+1)
	for chartID := range info.ChartStatsBatchInsertChunkSize + 1 {
		stats = append(stats, &entity.ChartBestSlotStatsByRatingBand{ChartID: chartID, RatingBandID: 0})
	}

	// When
	err := NewChartStatsBatchRepository(db).ReplaceAll(context.Background(), &chartstatsbatch.Snapshot{BestSlotStats: stats})

	// Then
	require.NoError(t, err)
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM chart_best_slot_stats_by_rating_band`))
	assert.Equal(t, info.ChartStatsBatchInsertChunkSize+1, count)
}

func TestChartStatsBatchRepository_ReplaceAll_失敗した場合は既存の統計を維持する(t *testing.T) {
	// Given: 前回の集計結果
	db := setupChartStatsBatchRepositorySQLite(t)
	_, err := db.Exec(`
		INSERT INTO chart_stats_by_rating_band VALUES (999, 0, 0,0,0,0,0,0,0,0, 0,0,0,0, 0,0,0,0,0,0, NULL, NULL, 0);
		INSERT INTO worldsend_chart_stats_by_rating_band VALUES (999, 0, 0,0,0,0,0,0,0,0, 0,0,0,0, 0,0,0,0,0,0, NULL, NULL, 0);
		INSERT INTO chart_best_slot_stats_by_rating_band VALUES (999, 0, 1, 1, 100);
	`)
	require.NoError(t, err)
	invalidPercentage := 200.0
	snapshot := &chartstatsbatch.Snapshot{
		ChartStats: []*entity.ChartStatsByRatingBand{{ChartID: 10, RatingBandID: 0, PlayerCount: 1}},
		// 最後に挿入するベスト枠採用率が制約違反になる
		BestSlotStats: []*entity.ChartBestSlotStatsByRatingBand{
			{ChartID: 10, RatingBandID: 0, BestPlayerCount: 1, EligiblePlayerCount: 1, BestPlayerPercentage: &invalidPercentage},
		},
	}

	// When
	err = NewChartStatsBatchRepository(db).ReplaceAll(context.Background(), snapshot)

	// Then
	assert.Error(t, err)
	for _, query := range []string{
		`SELECT chart_id FROM chart_stats_by_rating_band`,
		`SELECT worldsend_chart_id FROM worldsend_chart_stats_by_rating_band`,
		`SELECT chart_id FROM chart_best_slot_stats_by_rating_band`,
	} {
		var chartIDs []int
		require.NoError(t, db.Select(&chartIDs, query))
		assert.Equal(t, []int{999}, chartIDs, query)
	}
}
