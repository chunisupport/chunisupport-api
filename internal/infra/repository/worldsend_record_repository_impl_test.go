package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWorldsendRecordDB(t *testing.T, db *sqlx.DB) {
	t.Helper()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS clear_lamp_types (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS combo_lamp_types (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS full_chain_types (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS worldsend_charts (
			id INTEGER PRIMARY KEY,
			song_id INTEGER NOT NULL,
			level_star INTEGER,
			attribute TEXT,
			notes INTEGER,
			FOREIGN KEY (song_id) REFERENCES songs(id)
		);
		CREATE TABLE IF NOT EXISTS player_worldsend_records (
			player_id INTEGER NOT NULL,
			worldsend_chart_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL
		);
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO clear_lamp_types (id, name) VALUES (1, 'CLEAR');
		INSERT INTO combo_lamp_types (id, name) VALUES (1, 'NONE');
		INSERT INTO full_chain_types (id, name) VALUES (1, 'NONE');
	`)
	require.NoError(t, err)
}

// setupWorldsendRecordFilteringDB は絞り込み条件を検証するための記録データを用意します。
// プレイヤー10は削除済みでない2曲と削除済み1曲の記録を、プレイヤー20は共有する1曲の記録を持ちます。
func setupWorldsendRecordFilteringDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := setupTestDB(t)
	setupWorldsendRecordDB(t, db)

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
		VALUES
			(1, 'WE001', '曲1', 'artist', 1, 'WEIDX001', 1, 0),
			(2, 'WE002', '曲2', 'artist', 1, 'WEIDX002', 1, 0),
			(3, 'WE003', '削除曲', 'artist', 1, 'WEIDX003', 1, 1);
		INSERT INTO worldsend_charts (id, song_id) VALUES (101, 1), (102, 2), (103, 3);
		INSERT INTO player_worldsend_records
			(player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
		VALUES
			(10, 101, 900000, 1, 1, 1, '2024-01-01 00:00:00'),
			(10, 102, 950000, 1, 1, 1, '2024-02-01 00:00:00'),
			(10, 103, 990000, 1, 1, 1, '2024-03-01 00:00:00'),
			(20, 101, 800000, 1, 1, 1, '2024-04-01 00:00:00');
	`)
	require.NoError(t, err)

	return db
}

func TestWorldsendRecordRepository_FindByPlayerID_他プレイヤーと削除済み楽曲を除く(t *testing.T) {
	// Given
	db := setupWorldsendRecordFilteringDB(t)
	defer db.Close()

	// When
	records, err := (&worldsendRecordRepository{db: db}).FindByPlayerID(context.Background(), db, 10)

	// Then
	// 更新日時の降順で、指定プレイヤーの削除済みでない楽曲の記録だけが返る。
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.NotNil(t, records[0].Song)
	require.NotNil(t, records[1].Song)
	assert.Equal(t, "WE002", records[0].Song.DisplayID)
	assert.Equal(t, "WE001", records[1].Song.DisplayID)
	assert.Equal(t, uint32(950000), uint32(records[0].Score))
	assert.Equal(t, uint32(900000), uint32(records[1].Score))
}

func TestWorldsendRecordRepository_FindByPlayerIDAndSongDisplayID_他プレイヤーと削除済み楽曲を除く(t *testing.T) {
	tests := []struct {
		name string
		// When: 取得条件
		playerID  int
		displayID string
		// Then: 期待する記録の DisplayID とスコアの一覧
		expectedDisplayIDs []string
		expectedScores     []uint32
	}{
		{
			name:               "指定プレイヤーの指定楽曲の記録だけを返す",
			playerID:           10,
			displayID:          "WE001",
			expectedDisplayIDs: []string{"WE001"},
			expectedScores:     []uint32{900000},
		},
		{
			name:               "他プレイヤーを指定した場合はそのプレイヤーの記録だけを返す",
			playerID:           20,
			displayID:          "WE001",
			expectedDisplayIDs: []string{"WE001"},
			expectedScores:     []uint32{800000},
		},
		{
			name:               "削除済み楽曲の記録は返さない",
			playerID:           10,
			displayID:          "WE003",
			expectedDisplayIDs: []string{},
			expectedScores:     []uint32{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupWorldsendRecordFilteringDB(t)
			defer db.Close()

			// When
			records, err := (&worldsendRecordRepository{db: db}).FindByPlayerIDAndSongDisplayID(context.Background(), db, tt.playerID, tt.displayID)

			// Then
			require.NoError(t, err)
			gotIDs := []string{}
			gotScores := []uint32{}
			for _, record := range records {
				require.NotNil(t, record.Song)
				gotIDs = append(gotIDs, record.Song.DisplayID)
				gotScores = append(gotScores, uint32(record.Score))
			}
			assert.Equal(t, tt.expectedDisplayIDs, gotIDs)
			assert.Equal(t, tt.expectedScores, gotScores)
		})
	}
}

func TestFindByPlayerID_ScansLevelStarValueObject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupWorldsendRecordDB(t, db)

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0, 0);
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, 4, '狂', 1200);
		INSERT INTO player_worldsend_records (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
		VALUES (10, 101, 900000, 1, 1, 1, ?);
	`, time.Now().UTC())
	require.NoError(t, err)

	repo := &worldsendRecordRepository{db: db}
	records, err := repo.FindByPlayerID(context.Background(), db, 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.NotNil(t, records[0].WorldsendChart)
	require.NotNil(t, records[0].WorldsendChart.LevelStar)
	assert.Equal(t, 4, records[0].WorldsendChart.LevelStar.Int())
}

func TestFindByPlayerID_ScansNilLevelStarAsNil(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	setupWorldsendRecordDB(t, db)

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0, 0);
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, NULL, '狂', 1200);
		INSERT INTO player_worldsend_records (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
		VALUES (10, 101, 900000, 1, 1, 1, ?);
	`, time.Now().UTC())
	require.NoError(t, err)

	repo := &worldsendRecordRepository{db: db}
	records, err := repo.FindByPlayerID(context.Background(), db, 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.NotNil(t, records[0].WorldsendChart)
	assert.Nil(t, records[0].WorldsendChart.LevelStar)
}

func TestWorldsendRecordRepository_FindByPlayerIDAndSongDisplayID_指定楽曲だけを返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupWorldsendRecordDB(t, db)
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend)
		VALUES
			(1, 'WE001', '曲1', 'artist', 1, 'WEIDX001', 1),
			(2, 'WE002', '曲2', 'artist', 1, 'WEIDX002', 1);
		INSERT INTO worldsend_charts (id, song_id) VALUES (101, 1), (102, 2);
		INSERT INTO player_worldsend_records
			(player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
		VALUES
			(10, 101, 900000, 1, 1, 1, CURRENT_TIMESTAMP),
			(10, 102, 950000, 1, 1, 1, CURRENT_TIMESTAMP);
	`)
	require.NoError(t, err)

	// When
	records, err := (&worldsendRecordRepository{db: db}).FindByPlayerIDAndSongDisplayID(context.Background(), db, 10, "WE001")

	// Then
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "WE001", records[0].Song.DisplayID)
}
