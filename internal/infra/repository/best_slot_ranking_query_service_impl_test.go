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

func TestBestSlotRankingQueryService_List_割合順で楽曲情報を一括結合する(t *testing.T) {
	// Given
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE songs (id INTEGER PRIMARY KEY, display_id TEXT, title TEXT, is_worldsend INTEGER, is_deleted INTEGER);
		CREATE TABLE difficulties (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE charts (id INTEGER PRIMARY KEY, song_id INTEGER, difficulty_id INTEGER, const REAL, is_const_unknown INTEGER);
		INSERT INTO difficulties VALUES (4, 'MASTER');
		INSERT INTO songs VALUES
			(1, '0000000000000001', '楽曲A', 0, 0),
			(2, '0000000000000002', '楽曲B', 0, 0),
			(3, '0000000000000003', '楽曲C', 0, 0),
			(4, '0000000000000004', '削除済み', 0, 1),
			(5, '0000000000000005', 'WORLD''S END', 1, 0);
		INSERT INTO charts VALUES
			(101, 1, 4, 14.8, 0), (102, 2, 4, 14.7, 1), (103, 3, 4, 14.9, 0),
			(104, 4, 4, 15.0, 0), (105, 5, 4, 15.0, 0);
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE chart_best_slot_stats_by_rating_band (
			chart_id INTEGER, rating_band_id INTEGER, best_player_count INTEGER,
			eligible_player_count INTEGER, best_player_percentage REAL
		);
		CREATE TABLE chart_stats_by_rating_band (
			chart_id INTEGER, rating_band_id INTEGER, average_score REAL
		);
		INSERT INTO chart_best_slot_stats_by_rating_band VALUES
			(104, 22, 20, 40, 50.0),
			(105, 22, 20, 40, 50.0),
			(101, 22, 10, 40, 25.0),
			(103, 22, 10, 40, 25.0),
			(102, 22, 8, 40, 20.0);
		INSERT INTO chart_stats_by_rating_band VALUES
			(101, 22, 1007500.5),
			(103, 21, 999999.0),
			(102, 22, 1005000.0);
	`)
	require.NoError(t, err)

	service := NewBestSlotRankingQueryService(db)

	// When
	page, err := service.List(context.Background(), 22, nil, 2)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 40, page.EligiblePlayerCount)
	require.Len(t, page.Items, 2)
	assert.Equal(t, "0000000000000001", page.Items[0].SongDisplayID)
	assert.Equal(t, "楽曲A", page.Items[0].SongTitle)
	assert.Equal(t, 1, page.Items[0].Rank)
	require.NotNil(t, page.Items[0].AverageScore)
	assert.Equal(t, 1007500.5, *page.Items[0].AverageScore)
	assert.Equal(t, "0000000000000003", page.Items[1].SongDisplayID)
	assert.Equal(t, 2, page.Items[1].Rank)
	assert.Nil(t, page.Items[1].AverageScore)
	require.NotNil(t, page.NextCursor)
	assert.Equal(t, "0000000000000003", page.NextCursor.SongDisplayID)
	assert.Equal(t, "MASTER", page.NextCursor.Difficulty)

	// When: 次ページ
	next, err := service.List(context.Background(), 22, page.NextCursor, 2)

	// Then
	require.NoError(t, err)
	require.Len(t, next.Items, 1)
	assert.Equal(t, "0000000000000002", next.Items[0].SongDisplayID)
	assert.Nil(t, next.NextCursor)
	var _ domainrepo.BestSlotRankingQueryService = service
}
