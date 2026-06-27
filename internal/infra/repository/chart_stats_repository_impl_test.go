package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChartStatsDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db := setupTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS chart_stats_by_rating_band (
			chart_id INTEGER NOT NULL,
			rating_band_id INTEGER NOT NULL,
			rank_aaal INTEGER NOT NULL,
			rank_s INTEGER NOT NULL,
			rank_sp INTEGER NOT NULL,
			rank_ss INTEGER NOT NULL,
			rank_ssp INTEGER NOT NULL,
			rank_sss INTEGER NOT NULL,
			rank_sssp INTEGER NOT NULL,
			rank_max INTEGER NOT NULL,
			combo_none INTEGER NOT NULL,
			combo_fc INTEGER NOT NULL,
			combo_aj INTEGER NOT NULL,
			combo_ajc INTEGER NOT NULL,
			clear_failed INTEGER NOT NULL,
			clear_clear INTEGER NOT NULL,
			clear_hard INTEGER NOT NULL,
			clear_brave INTEGER NOT NULL,
			clear_absolute INTEGER NOT NULL,
			clear_catastrophy INTEGER NOT NULL,
			average_score REAL,
			player_count INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS worldsend_chart_stats_by_rating_band (
			worldsend_chart_id INTEGER NOT NULL,
			rating_band_id INTEGER NOT NULL,
			rank_aaal INTEGER NOT NULL,
			rank_s INTEGER NOT NULL,
			rank_sp INTEGER NOT NULL,
			rank_ss INTEGER NOT NULL,
			rank_ssp INTEGER NOT NULL,
			rank_sss INTEGER NOT NULL,
			rank_sssp INTEGER NOT NULL,
			rank_max INTEGER NOT NULL,
			combo_none INTEGER NOT NULL,
			combo_fc INTEGER NOT NULL,
			combo_aj INTEGER NOT NULL,
			combo_ajc INTEGER NOT NULL,
			clear_failed INTEGER NOT NULL,
			clear_clear INTEGER NOT NULL,
			clear_hard INTEGER NOT NULL,
			clear_brave INTEGER NOT NULL,
			clear_absolute INTEGER NOT NULL,
			clear_catastrophy INTEGER NOT NULL,
			average_score REAL,
			player_count INTEGER NOT NULL
		);
	`)
	require.NoError(t, err)

	return db
}

func TestFindWorldsendChartStatsByChartIDs_UsesWorldsendTable(t *testing.T) {
	db := setupChartStatsDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO chart_stats_by_rating_band (
			chart_id, rating_band_id, rank_aaal, rank_s, rank_sp, rank_ss, rank_ssp, rank_sss, rank_sssp, rank_max,
			combo_none, combo_fc, combo_aj, combo_ajc, clear_failed, clear_clear, clear_hard, clear_brave, clear_absolute, clear_catastrophy,
			average_score, player_count
		) VALUES (
			100, 1, 10, 11, 12, 13, 14, 15, 16, 17,
			18, 19, 20, 21, 22, 23, 24, 25, 26, 27,
			900000, 27
		);
		INSERT INTO worldsend_chart_stats_by_rating_band (
			worldsend_chart_id, rating_band_id, rank_aaal, rank_s, rank_sp, rank_ss, rank_ssp, rank_sss, rank_sssp, rank_max,
			combo_none, combo_fc, combo_aj, combo_ajc, clear_failed, clear_clear, clear_hard, clear_brave, clear_absolute, clear_catastrophy,
			average_score, player_count
		) VALUES (
			100, 1, 1, 2, 3, 4, 5, 6, 7, 8,
			9, 10, 11, 12, 13, 14, 15, 16, 17, 18,
			950000, 18
		);
	`)
	require.NoError(t, err)

	repo := &chartStatsRepository{db: db}

	chartStats, err := repo.FindChartStatsByChartIDs(context.Background(), db, []int{100})
	require.NoError(t, err)
	require.Len(t, chartStats, 1)
	assert.Equal(t, 20, chartStats[0].Combo.AJ)
	assert.Equal(t, 21, chartStats[0].Combo.AJC)

	stats, err := repo.FindWorldsendChartStatsByChartIDs(context.Background(), db, []int{100})
	require.NoError(t, err)
	require.Len(t, stats, 1)

	assert.Equal(t, 100, stats[0].ChartID)
	assert.Equal(t, 1, stats[0].Rank.AAAL)
	assert.Equal(t, 2, stats[0].Rank.S)
	assert.Equal(t, 9, stats[0].Combo.None)
	assert.Equal(t, 11, stats[0].Combo.AJ)
	assert.Equal(t, 12, stats[0].Combo.AJC)
	assert.Equal(t, 13, stats[0].Clear.Failed)
	if assert.NotNil(t, stats[0].AverageScore) {
		assert.InDelta(t, 950000, *stats[0].AverageScore, 0.001)
	}
	assert.Equal(t, 18, stats[0].PlayerCount)
}
