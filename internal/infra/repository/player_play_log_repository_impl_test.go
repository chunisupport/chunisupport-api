package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindPlayLogChartCandidates_ジャケットで通常譜面とWORLD_END譜面候補を返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE worldsend_charts (
			id INTEGER PRIMARY KEY,
			song_id INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, jacket, is_worldsend, is_deleted)
		VALUES
			(1, 'normal-song', '通常曲', 'artist', 1, 'normal-idx', 'shared-jacket', 0, 0),
			(2, 'worldsend-song', 'WORLD''S END曲', 'artist', 1, 'worldsend-idx', 'shared-jacket', 1, 0)
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown)
		VALUES (101, 1, 4, 14.0, 0)
	`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO worldsend_charts (id, song_id) VALUES (201, 2)`)
	require.NoError(t, err)
	repo := NewPlayerDataRepository(db)

	// When
	got, err := repo.FindPlayLogChartCandidates(context.Background(), db, []string{"shared-jacket"})

	// Then
	require.NoError(t, err)
	assert.ElementsMatch(t, []entity.PlayLogChartCandidate{
		{ID: 101, Title: "通常曲", Jacket: "shared-jacket", DifficultyID: 4},
		{ID: 201, Title: "WORLD'S END曲", Jacket: "shared-jacket"},
	}, got)
}

func TestReplacePlayLogs_既存履歴を削除して受信順で保存する(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE player_play_logs (
			player_id INTEGER NOT NULL,
			play_order INTEGER NOT NULL,
			chart_id INTEGER,
			worldsend_chart_id INTEGER,
			jacket_img_id TEXT,
			difficulty_id INTEGER,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL,
			played_at DATETIME NOT NULL,
			PRIMARY KEY (player_id, play_order)
		)
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO player_play_logs (
			player_id, play_order, difficulty_id, score,
			clear_lamp_id, combo_lamp_id, full_chain_id, played_at
		) VALUES (1, 0, 4, 900000, 1, 1, 1, ?),
		         (2, 0, 4, 900000, 1, 1, 1, ?)
	`, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	chartID := 101
	logs := []entity.PlayerPlayLog{
		{
			ChartID: &chartID, Jacket: new("jacket-1"), DifficultyID: new(4),
			Score: 1000000, ClearLampID: 2, ComboLampID: 3, FullChainID: 1,
			PlayedAt: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			Jacket: new("unknown-jacket"), Score: 990000,
			ClearLampID: 1, ComboLampID: 1, FullChainID: 1,
			PlayedAt: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	repo := NewPlayerDataRepository(db)

	// When
	err = repo.ReplacePlayLogs(context.Background(), db, 1, logs)

	// Then
	require.NoError(t, err)
	var rows []struct {
		PlayerID     int     `db:"player_id"`
		PlayOrder    int     `db:"play_order"`
		ChartID      *int    `db:"chart_id"`
		Jacket       *string `db:"jacket_img_id"`
		DifficultyID *int    `db:"difficulty_id"`
		Score        int     `db:"score"`
	}
	require.NoError(t, db.Select(&rows, `SELECT player_id, play_order, chart_id, jacket_img_id, difficulty_id, score FROM player_play_logs ORDER BY player_id, play_order`))
	require.Len(t, rows, 3)
	assert.Equal(t, 1, rows[0].PlayerID)
	assert.Equal(t, 0, rows[0].PlayOrder)
	require.NotNil(t, rows[0].ChartID)
	assert.Equal(t, 101, *rows[0].ChartID)
	assert.Equal(t, new("jacket-1"), rows[0].Jacket)
	assert.Equal(t, new(4), rows[0].DifficultyID)
	assert.Equal(t, 1000000, rows[0].Score)
	assert.Equal(t, 1, rows[1].PlayerID)
	assert.Equal(t, 1, rows[1].PlayOrder)
	assert.Nil(t, rows[1].ChartID)
	assert.Equal(t, new("unknown-jacket"), rows[1].Jacket)
	assert.Nil(t, rows[1].DifficultyID)
	assert.Equal(t, 2, rows[2].PlayerID)
}

func TestFindPlayHistory_譜面とランプを解決して受信順で返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE worldsend_charts (id INTEGER PRIMARY KEY, song_id INTEGER NOT NULL)`,
		`CREATE TABLE clear_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE full_chain_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE player_play_logs (
			player_id INTEGER NOT NULL, play_order INTEGER NOT NULL,
			chart_id INTEGER, worldsend_chart_id INTEGER, jacket_img_id TEXT,
			difficulty_id INTEGER, score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL, combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL, played_at DATETIME NOT NULL,
			PRIMARY KEY (player_id, play_order)
		)`,
	} {
		_, err := db.Exec(statement)
		require.NoError(t, err)
	}
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (100, 'normal', '通常曲', 'artist', 1, 'normal', 'jacket-1', 0, 0),
		       (200, 'worldsend', 'WORLD''S END曲', 'artist', 1, 'worldsend', 'jacket-2', 1, 0)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown) VALUES (1001, 100, 4, 14.0, 0)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO worldsend_charts (id, song_id) VALUES (2001, 200)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO clear_lamp_types VALUES (1, 'NONE'), (2, 'CLEAR')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO combo_lamp_types VALUES (1, 'NONE'), (2, 'FULL COMBO')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO full_chain_types VALUES (1, 'NONE'), (2, 'FULL CHAIN')`)
	require.NoError(t, err)
	firstPlayedAt := time.Date(2026, time.November, 26, 0, 10, 0, 0, time.UTC)
	secondPlayedAt := firstPlayedAt.Add(-time.Minute)
	_, err = db.Exec(`
		INSERT INTO player_play_logs
			(player_id, play_order, chart_id, worldsend_chart_id, jacket_img_id, difficulty_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, played_at)
		VALUES (7, 0, 1001, NULL, 'jacket-1', 4, 1005000, 2, 2, 2, ?),
		       (7, 1, NULL, 2001, 'jacket-2', NULL, 990000, 1, 1, 1, ?),
		       (7, 2, NULL, NULL, 'unknown', 3, 500000, 2, 1, 1, ?)`,
		firstPlayedAt, secondPlayedAt, secondPlayedAt.Add(-time.Minute))
	require.NoError(t, err)

	// When
	entries, err := NewPlayerPlayHistoryQueryService(db).FindByPlayerID(context.Background(), db, 7)

	// Then
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, "通常曲", *entries[0].Title)
	assert.Equal(t, "MASTER", *entries[0].Difficulty)
	assert.Equal(t, 1005000, entries[0].Score)
	assert.Equal(t, "CLEAR", *entries[0].ClearLamp)
	assert.Equal(t, "FULL COMBO", *entries[0].ComboLamp)
	assert.Equal(t, "FULL CHAIN", *entries[0].FullChain)
	assert.Equal(t, firstPlayedAt, entries[0].PlayedAt)
	assert.Equal(t, "WORLD'S END曲", *entries[1].Title)
	assert.Equal(t, "WORLDSEND", *entries[1].Difficulty)
	assert.Nil(t, entries[1].ClearLamp)
	assert.Nil(t, entries[1].ComboLamp)
	assert.Nil(t, entries[1].FullChain)
	assert.Nil(t, entries[2].Title)
	assert.Equal(t, "EXPERT", *entries[2].Difficulty)
}
