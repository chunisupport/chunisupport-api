package repository

import (
	"context"
	"testing"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestChartStatsExportQueryService_Get_未プレイ譜面を含めて一括取得する(t *testing.T) {
	// Given
	database, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer database.Close()

	_, err = database.Exec(`
		CREATE TABLE songs (id INTEGER PRIMARY KEY, display_id TEXT, title TEXT, is_worldsend INTEGER, is_deleted INTEGER);
		CREATE TABLE difficulties (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE charts (id INTEGER PRIMARY KEY, song_id INTEGER, difficulty_id INTEGER, const REAL, is_const_unknown INTEGER);
		CREATE TABLE worldsend_charts (id INTEGER PRIMARY KEY, song_id INTEGER, level_star INTEGER, attribute TEXT);
		CREATE TABLE rating_bands (id INTEGER PRIMARY KEY, label TEXT, sort_order INTEGER);
		CREATE TABLE chart_stats_by_rating_band (
			chart_id INTEGER, rating_band_id INTEGER, rank_aaal INTEGER, rank_s INTEGER, rank_sp INTEGER,
			rank_ss INTEGER, rank_ssp INTEGER, rank_sss INTEGER, rank_sssp INTEGER, rank_max INTEGER,
			combo_none INTEGER, combo_fc INTEGER, combo_aj INTEGER, combo_ajc INTEGER,
			clear_failed INTEGER, clear_clear INTEGER, clear_hard INTEGER, clear_brave INTEGER,
			clear_absolute INTEGER, clear_catastrophy INTEGER, player_count INTEGER,
			average_score REAL, median_score REAL
		);
		CREATE TABLE worldsend_chart_stats_by_rating_band (
			worldsend_chart_id INTEGER, rating_band_id INTEGER, rank_aaal INTEGER, rank_s INTEGER, rank_sp INTEGER,
			rank_ss INTEGER, rank_ssp INTEGER, rank_sss INTEGER, rank_sssp INTEGER, rank_max INTEGER,
			combo_none INTEGER, combo_fc INTEGER, combo_aj INTEGER, combo_ajc INTEGER,
			clear_failed INTEGER, clear_clear INTEGER, clear_hard INTEGER, clear_brave INTEGER,
			clear_absolute INTEGER, clear_catastrophy INTEGER, player_count INTEGER,
			average_score REAL, median_score REAL
		);
		INSERT INTO difficulties VALUES (1, 'BASIC'), (4, 'MASTER');
		INSERT INTO rating_bands VALUES (0, 'ALL', 0), (1, '-14.9', 1), (2, '15.0', 2);
		INSERT INTO songs VALUES
			(1, '0000000000000001', 'プレイ済み', 0, 0),
			(2, '0000000000000002', '未プレイ', 0, 0),
			(3, '0000000000000003', '削除済み', 0, 1),
			(4, '0000000000000004', 'WE', 1, 0);
		INSERT INTO charts VALUES (11, 1, 4, 12.7, 1), (12, 2, 1, 3.0, 0), (13, 3, 4, 14.0, 0);
		INSERT INTO worldsend_charts VALUES (21, 4, 5, '狂');
		INSERT INTO chart_stats_by_rating_band VALUES
			(11, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 1007000.5, 1008000),
			(11, 1, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 1005000, 1006000);
		INSERT INTO worldsend_chart_stats_by_rating_band VALUES
			(21, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 1009000.5, 1010000);
	`)
	require.NoError(t, err)
	service := NewChartStatsExportQueryService(database)

	// When
	snapshot, err := service.Get(context.Background())

	// Then
	require.NoError(t, err)
	require.Len(t, snapshot.Charts, 2)
	assert.Equal(t, "BASIC", snapshot.Charts[0].Difficulty)
	assert.Equal(t, 0, snapshot.Charts[0].PlayerCount)
	assert.Equal(t, []domainrepo.ChartStatsExportScore{{RatingBand: "ALL"}, {RatingBand: "-14.9"}, {RatingBand: "15.0"}}, snapshot.Charts[0].Scores)
	assert.Equal(t, "MASTER", snapshot.Charts[1].Difficulty)
	assert.Equal(t, 19, snapshot.Charts[1].PlayerCount)
	assert.Equal(t, 8, snapshot.Charts[1].Rank.Max)
	assert.Equal(t, 12, snapshot.Charts[1].Combo.AJC)
	assert.Equal(t, domainrepo.ChartStatsExportClear{Failed: 13, Clear: 14, Hard: 15, Brave: 16, Absolute: 17, Catastrophy: 18}, snapshot.Charts[1].Clear)
	require.Len(t, snapshot.Charts[1].Scores, 3)
	assert.Equal(t, "ALL", snapshot.Charts[1].Scores[0].RatingBand)
	assert.Equal(t, 1007000.5, *snapshot.Charts[1].Scores[0].AverageScore)
	assert.Equal(t, 1008000.0, *snapshot.Charts[1].Scores[0].MedianScore)
	assert.Equal(t, "-14.9", snapshot.Charts[1].Scores[1].RatingBand)
	assert.Equal(t, 1005000.0, *snapshot.Charts[1].Scores[1].AverageScore)
	assert.Nil(t, snapshot.Charts[1].Scores[2].AverageScore)
	require.Len(t, snapshot.WorldsendCharts, 1)
	assert.Equal(t, 5, *snapshot.WorldsendCharts[0].LevelStar)
	assert.Equal(t, "狂", *snapshot.WorldsendCharts[0].Attribute)
	assert.Equal(t, 19, snapshot.WorldsendCharts[0].PlayerCount)
	assert.Equal(t, domainrepo.ChartStatsExportClear{Failed: 13, Clear: 14, Hard: 15, Brave: 16, Absolute: 17, Catastrophy: 18}, snapshot.WorldsendCharts[0].Clear)
	require.Len(t, snapshot.WorldsendCharts[0].Scores, 3)
	assert.Equal(t, 1009000.5, *snapshot.WorldsendCharts[0].Scores[0].AverageScore)
	assert.Equal(t, 1010000.0, *snapshot.WorldsendCharts[0].Scores[0].MedianScore)
	assert.Nil(t, snapshot.WorldsendCharts[0].Scores[1].MedianScore)
	var _ domainrepo.ChartStatsExportQueryService = service
}
