package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFriendChartRankingQueryService_ListRecords_自分と相互承認済みフレンドのプレイ済みだけを返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupFriendChartRankingDB(t, db)
	insertFriendChartRankingFixtures(t, db)

	query := NewFriendChartRankingQueryService()
	chart, err := query.FindChart(context.Background(), db, "0000000000000001", "MASTER")
	require.NoError(t, err)
	require.NotNil(t, chart)

	// When
	records, err := query.ListRecords(context.Background(), db, 1, chart.ChartID)

	// Then
	require.NoError(t, err)
	require.Len(t, records, 3)
	assert.Equal(t, []int{2, 3, 1}, []int{records[0].UserID, records[1].UserID, records[2].UserID})
	assert.Equal(t, []uint32{1_009_500, 1_009_500, 1_009_000}, []uint32{records[0].Score, records[1].Score, records[2].Score})
}

func TestFriendChartRankingQueryService_ListWorldsendRecords_自分と相互承認済みフレンドのプレイ済みだけを返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupFriendChartRankingDB(t, db)
	insertFriendChartRankingFixtures(t, db)
	insertFriendWorldsendRankingFixtures(t, db)

	query := NewFriendChartRankingQueryService()
	chart, err := query.FindWorldsendChart(context.Background(), db, "0000000000000002")
	require.NoError(t, err)
	require.NotNil(t, chart)
	require.NotNil(t, chart.LevelStar)
	require.NotNil(t, chart.Attribute)
	assert.Equal(t, 5, *chart.LevelStar)
	assert.Equal(t, "狂", *chart.Attribute)

	// When
	records, err := query.ListWorldsendRecords(context.Background(), db, 1, chart.ChartID)

	// Then
	require.NoError(t, err)
	require.Len(t, records, 3)
	assert.Equal(t, []int{2, 3, 1}, []int{records[0].UserID, records[1].UserID, records[2].UserID})
	assert.Equal(t, []uint32{1_009_700, 1_009_700, 1_009_100}, []uint32{records[0].Score, records[1].Score, records[2].Score})
	assert.Equal(t, "ALL JUSTICE", records[0].ComboLamp)
}

func setupFriendChartRankingDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL,
			player_id INTEGER
		);
		CREATE TABLE players (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			player_name TEXT NOT NULL
		);
		CREATE TABLE friendships (
			user_id INTEGER NOT NULL,
			friend_user_id INTEGER NOT NULL,
			status_id INTEGER NOT NULL
		);
		CREATE TABLE player_records (
			player_id INTEGER NOT NULL,
			chart_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE TABLE clear_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE full_chain_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE worldsend_charts (
			id INTEGER PRIMARY KEY,
			song_id INTEGER NOT NULL,
			level_star INTEGER,
			attribute TEXT,
			notes INTEGER,
			notes_designer TEXT,
			updated_at DATETIME
		);
		CREATE TABLE player_worldsend_records (
			player_id INTEGER NOT NULL,
			worldsend_chart_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL
		);
		INSERT INTO clear_lamp_types (id, name) VALUES (1, 'CLEAR');
		INSERT INTO combo_lamp_types (id, name) VALUES (1, 'NONE'), (2, 'FULL COMBO'), (3, 'ALL JUSTICE');
		INSERT INTO full_chain_types (id, name) VALUES (1, 'NONE');
	`)
	require.NoError(t, err)
}

func insertFriendWorldsendRankingFixtures(t *testing.T, db *sqlx.DB) {
	t.Helper()
	updatedAt := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
		VALUES (2, '0000000000000002', 'WE楽曲名', 'WE artist', 1, '2', 1, 0);
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (20, 2, 5, '狂', 2000);
		INSERT INTO player_worldsend_records (
			player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at
		) VALUES
			(101, 20, 1009100, 1, 1, 1, ?),
			(102, 20, 1009700, 1, 3, 1, ?),
			(103, 20, 1009700, 1, 2, 1, ?),
			(104, 20, 1010000, 1, 3, 1, ?)
	`, updatedAt, updatedAt.Add(time.Minute), updatedAt, updatedAt.Add(2*time.Minute))
	require.NoError(t, err)
}

func insertFriendChartRankingFixtures(t *testing.T, db *sqlx.DB) {
	t.Helper()
	updatedAt := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
		VALUES (1, '0000000000000001', '楽曲名', 'artist', 1, '1', 0, 0);
		INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown)
		VALUES (10, 1, 4, 14.5, 0);
		INSERT INTO users (id, username, player_id) VALUES
			(1, 'me', 101),
			(2, 'friend1', 102),
			(3, 'friend2', 103),
			(4, 'oneway', 104),
			(5, 'unplayed', 105);
		INSERT INTO players (id, user_id, player_name) VALUES
			(101, 1, 'ME'),
			(102, 2, 'FRIEND1'),
			(103, 3, 'FRIEND2'),
			(104, 4, 'ONEWAY'),
			(105, 5, 'UNPLAYED');
		INSERT INTO friendships (user_id, friend_user_id, status_id) VALUES
			(1, 2, 2), (2, 1, 2),
			(1, 3, 2), (3, 1, 2),
			(1, 4, 2);
		INSERT INTO player_records (
			player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at
		) VALUES
			(101, 10, 1009000, 1, 1, 1, ?),
			(102, 10, 1009500, 1, 3, 1, ?),
			(103, 10, 1009500, 1, 2, 1, ?),
			(104, 10, 1010000, 1, 3, 1, ?)
	`, updatedAt, updatedAt.Add(time.Minute), updatedAt, updatedAt.Add(2*time.Minute))
	require.NoError(t, err)
}
