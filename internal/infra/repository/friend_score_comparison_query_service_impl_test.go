package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFriendScoreComparisonQueryService_FindAcceptedFriendPair_双方向acceptedだけを返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupFriendScoreComparisonDB(t, db)
	insertFriendScoreComparisonUsers(t, db)
	query := NewFriendScoreComparisonQueryService(db)

	tests := []struct {
		name       string
		username   string
		wantFound  bool
		wantPlayer *int
	}{
		{name: "双方向accepted", username: "frienduser", wantFound: true, wantPlayer: intPtr(102)},
		{name: "片方向accepted", username: "onewayuser", wantFound: false},
		{name: "pending", username: "pendinguser", wantFound: false},
		{name: "不存在", username: "missinguser", wantFound: false},
		{name: "自分自身", username: "myuser", wantFound: false},
		{name: "未連携フレンド", username: "unlinked", wantFound: true, wantPlayer: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			pair, err := query.FindAcceptedFriendPair(context.Background(), 1, tt.username)

			// Then
			require.NoError(t, err)
			if !tt.wantFound {
				assert.Nil(t, pair)
				return
			}
			require.NotNil(t, pair)
			assert.Equal(t, "myuser", pair.Self.Username)
			assert.Equal(t, tt.username, pair.Friend.Username)
			assert.Equal(t, 101, *pair.Self.PlayerID)
			if tt.wantPlayer == nil {
				assert.Nil(t, pair.Friend.PlayerID)
				return
			}
			require.NotNil(t, pair.Friend.PlayerID)
			assert.Equal(t, *tt.wantPlayer, *pair.Friend.PlayerID)
		})
	}
}

func TestFriendScoreComparisonQueryService_ListChartRecords_指定難易度の有効譜面だけをマスタ順で返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupFriendScoreComparisonDB(t, db)
	insertFriendScoreComparisonCharts(t, db)
	query := NewFriendScoreComparisonQueryService(db)

	// When
	records, err := query.ListChartRecords(context.Background(), 101, 102, "MASTER")

	// Then
	require.NoError(t, err)
	require.Len(t, records, 4)
	assert.Equal(t, []string{"0000000000000009", "0000000000000005", "0000000000000001", "0000000000000007"}, []string{
		records[0].SongDisplayID, records[1].SongDisplayID, records[2].SongDisplayID, records[3].SongDisplayID,
	})

	assert.Nil(t, records[0].Self)
	assert.Nil(t, records[0].Friend)
	assert.False(t, records[0].IsConstUnknown)

	require.NotNil(t, records[1].Self)
	assert.Nil(t, records[1].Friend)
	assert.Equal(t, uint32(0), records[1].Self.Score)
	assert.Equal(t, "NONE", records[1].Self.ClearLamp)

	require.NotNil(t, records[2].Self)
	require.NotNil(t, records[2].Friend)
	assert.Equal(t, uint32(1009000), records[2].Self.Score)
	assert.Equal(t, uint32(1007500), records[2].Friend.Score)
	assert.Equal(t, "CLEAR", records[2].Self.ClearLamp)
	assert.Equal(t, "FULL COMBO", records[2].Self.ComboLamp)
	assert.Equal(t, "NONE", records[2].Friend.FullChain)
	assert.InDelta(t, 14.5, records[2].ChartConst.Float64(), 0.001)
	assert.True(t, records[2].IsConstUnknown)

	require.Nil(t, records[3].Self)
	require.NotNil(t, records[3].Friend)
	assert.Equal(t, uint32(1005000), records[3].Friend.Score)
}

func TestFriendScoreComparisonQueryService_ListChartRecords_他難易度と対象外楽曲を除外する(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupFriendScoreComparisonDB(t, db)
	insertFriendScoreComparisonCharts(t, db)
	query := NewFriendScoreComparisonQueryService(db)

	// When
	expertRecords, err := query.ListChartRecords(context.Background(), 101, 102, "EXPERT")
	require.NoError(t, err)
	ultimaRecords, err := query.ListChartRecords(context.Background(), 101, 102, "ULTIMA")

	// Then
	require.NoError(t, err)
	require.Len(t, expertRecords, 1)
	assert.Equal(t, "0000000000000004", expertRecords[0].SongDisplayID)
	assert.Empty(t, ultimaRecords)
}

func setupFriendScoreComparisonDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
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
		INSERT INTO clear_lamp_types (id, name) VALUES (1, 'CLEAR'), (2, 'NONE');
		INSERT INTO combo_lamp_types (id, name) VALUES (1, 'NONE'), (2, 'FULL COMBO');
		INSERT INTO full_chain_types (id, name) VALUES (1, 'NONE');
	`)
	require.NoError(t, err)
}

func insertFriendScoreComparisonUsers(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO users (id, username, player_id) VALUES
			(1, 'myuser', 101),
			(2, 'frienduser', 102),
			(3, 'onewayuser', 103),
			(4, 'pendinguser', 104),
			(5, 'unlinked', NULL);
		INSERT INTO players (id, user_id, player_name) VALUES
			(101, 1, 'MY PLAYER'),
			(102, 2, 'FRIEND'),
			(103, 3, 'ONEWAY'),
			(104, 4, 'PENDING');
		INSERT INTO friendships (user_id, friend_user_id, status_id) VALUES
			(1, 2, 2), (2, 1, 2),
			(1, 3, 2),
			(1, 4, 1), (4, 1, 1),
			(1, 5, 2), (5, 1, 2);
	`)
	require.NoError(t, err)
}

func insertFriendScoreComparisonCharts(t *testing.T, db *sqlx.DB) {
	t.Helper()
	updatedAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted) VALUES
			(30, '0000000000000001', '楽曲名', 'アーティスト名', 1, '30', 0, 0),
			(10, '0000000000000009', '未プレイ楽曲', 'アーティスト名', 1, '10', 0, 0),
			(20, '0000000000000005', 'スコア0楽曲', 'アーティスト名', 1, '20', 0, 0),
			(40, '0000000000000007', '相手のみ楽曲', 'アーティスト名', 1, '40', 0, 0),
			(50, '0000000000000008', '削除楽曲', 'アーティスト名', 1, '50', 0, 1),
			(60, '0000000000000006', 'WE楽曲', 'アーティスト名', 1, '60', 1, 0),
			(70, '0000000000000004', 'EXPERT楽曲', 'アーティスト名', 1, '70', 0, 0);
		INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown) VALUES
			(30, 30, 4, 14.5, 1),
			(10, 10, 4, 13.0, 0),
			(20, 20, 4, 12.0, 0),
			(40, 40, 4, 13.5, 0),
			(50, 50, 4, 14.0, 0),
			(60, 60, 4, 14.0, 0),
			(70, 70, 3, 11.0, 0);
		INSERT INTO player_records (
			player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at
		) VALUES
			(101, 30, 1009000, 1, 2, 1, ?),
			(102, 30, 1007500, 1, 1, 1, ?),
			(101, 20, 0, 2, 1, 1, ?),
			(102, 40, 1005000, 1, 1, 1, ?),
			(101, 50, 1010000, 1, 2, 1, ?),
			(101, 60, 1010000, 1, 2, 1, ?),
			(101, 70, 1000000, 1, 1, 1, ?);
	`, updatedAt, updatedAt, updatedAt, updatedAt, updatedAt, updatedAt, updatedAt)
	require.NoError(t, err)
}

func intPtr(value int) *int {
	return &value
}
