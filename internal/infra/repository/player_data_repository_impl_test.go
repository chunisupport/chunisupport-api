package repository

import (
	"context"
	"strings"
	"testing"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMasterData_不正な譜面定数値ならエラーを返す(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "負の譜面定数値なら変換エラーを返す",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupTestDB(t)
			defer db.Close()

			_, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS worldsend_charts (
					id INTEGER PRIMARY KEY,
					song_id INTEGER NOT NULL
				)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, official_idx, is_worldsend, is_deleted)
				VALUES (1, 'c1', 'Song 1', 'Artist 1', 1, 180, 'idx-1', 0, 0)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown, notes)
				VALUES (1, 1, 3, -1.0, 0, 1000)
			`)
			require.NoError(t, err)

			repo := NewPlayerDataRepository(db)

			// When
			result, err := repo.LoadMasterData(context.Background(), []string{"idx-1"})

			// Then
			require.Error(t, err)
			assert.Nil(t, result)
			assert.ErrorContains(t, err, "failed to convert chart model to entity")
			assert.ErrorContains(t, err, "invalid chart constant")
			assert.ErrorContains(t, err, "chart_id=1")
		})
	}
}

func TestLoadMasterData_正常な譜面定数値ならマスタを読み込める(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "譜面と楽曲をキー付きで返す",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupTestDB(t)
			defer db.Close()

			_, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS worldsend_charts (
					id INTEGER PRIMARY KEY,
					song_id INTEGER NOT NULL
				)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, official_idx, is_worldsend, is_deleted)
				VALUES (1, 'c1', 'Song 1', 'Artist 1', 1, 180, 'idx-1', 0, 0)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown, notes)
				VALUES (1, 1, 3, 13.5, 0, 1000)
			`)
			require.NoError(t, err)

			repo := NewPlayerDataRepository(db)

			// When
			result, err := repo.LoadMasterData(context.Background(), []string{"idx-1"})

			// Then
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Songs, 1)
			require.Len(t, result.ChartsByKey, 1)
			require.Len(t, result.ChartsByID, 1)

			song, ok := result.Songs["idx-1"]
			require.True(t, ok)
			assert.Equal(t, 1, song.ID)

			chartByKey, ok := result.ChartsByKey["1:3"]
			require.True(t, ok)
			assert.Equal(t, 1, chartByKey.ID)
			assert.Equal(t, 13.5, float64(chartByKey.Const))

			chartByID, ok := result.ChartsByID[1]
			require.True(t, ok)
			assert.Equal(t, 13.5, float64(chartByID.Const))
		})
	}
}

func TestGetOverpowerTargetStats_対象楽曲の最大OP合計を取得する(t *testing.T) {
	tests := []struct {
		name      string
		filter    domainrepo.OverpowerTargetFilter
		wantCount int
		wantTotal float64
	}{
		{
			name: "WORLD'S ENDと削除済み楽曲を除外して通常楽曲ごとの最大OPを合計する",
			filter: domainrepo.OverpowerTargetFilter{
				ExcludeWorldsend: true,
				ExcludeDeleted:   true,
			},
			wantCount: 2,
			wantTotal: service.CalcSongMaxOP(15.4) +
				service.CalcSongMaxOP(14.5),
		},
		{
			name: "フィルタを指定しなければ全楽曲を対象にする",
			filter: domainrepo.OverpowerTargetFilter{
				ExcludeWorldsend: false,
				ExcludeDeleted:   false,
			},
			wantCount: 4,
			wantTotal: service.CalcSongMaxOP(15.4) +
				service.CalcSongMaxOP(14.5) +
				service.CalcSongMaxOP(13.0) +
				service.CalcSongMaxOP(12.0),
		},
		{
			name: "プレイヤー未解禁曲がある場合は通常未解禁曲を除外しULTIMA未解禁時は下位難易度を採用する",
			filter: domainrepo.OverpowerTargetFilter{
				ExcludeWorldsend: true,
				ExcludeDeleted:   true,
				PlayerID:         intPtrForPlayerDataRepositoryTest(100),
			},
			wantCount: 2,
			wantTotal: service.CalcSongMaxOP(15.0) + // Song1: ULTIMA未解禁のためMASTERを採用
				service.CalcSongMaxOP(14.5), // Song2: 通常譜面は解禁済み
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupTestDB(t)
			defer db.Close()

			_, err := db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, official_idx, is_worldsend, is_deleted)
				VALUES
					(1, 'c1', 'Song 1', 'Artist 1', 1, 180, 'idx-1', 0, 0),
					(2, 'c2', 'Song 2', 'Artist 2', 1, 180, 'idx-2', 0, 0),
					(3, 'c3', 'Worldsend Song', 'Artist 3', 1, 180, 'idx-3', 1, 0),
					(4, 'c4', 'Deleted Song', 'Artist 4', 1, 180, 'idx-4', 0, 1)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown, notes)
				VALUES
					(1, 1, 3, 13.0, 0, 1000),
					(2, 1, 4, 15.0, 0, 1200),
					(6, 1, 5, 15.4, 0, 1250),
					(3, 2, 4, 14.5, 1, 1100),
					(4, 3, 4, 13.0, 0, 900),
					(5, 4, 4, 12.0, 0, 800)
			`)
			require.NoError(t, err)
			_, err = db.Exec(`
				CREATE TABLE IF NOT EXISTS player_locked_songs (
					player_id INTEGER NOT NULL,
					song_id INTEGER NOT NULL,
					is_ultima BOOLEAN NOT NULL,
					PRIMARY KEY (player_id, song_id, is_ultima)
				)
			`)
			require.NoError(t, err)
			_, err = db.Exec(`
				INSERT INTO player_locked_songs (player_id, song_id, is_ultima)
				VALUES
					(100, 1, 1)
			`)
			require.NoError(t, err)

			repo := NewPlayerDataRepository(db)

			// When
			stats, err := repo.GetOverpowerTargetStats(context.Background(), tt.filter)

			// Then
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, tt.wantCount, stats.SongCount)
			assert.InDelta(t, tt.wantTotal, stats.MaxOverpowerTotal, 0.0001)
		})
	}
}

func TestGetOverpowerTargetStatsWithExecutor_トランザクション内の未解禁設定を反映する(t *testing.T) {
	// Given
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, official_idx, is_worldsend, is_deleted)
		VALUES
			(1, 'c1', 'Song 1', 'Artist 1', 1, 180, 'idx-1', 0, 0),
			(2, 'c2', 'Song 2', 'Artist 2', 1, 180, 'idx-2', 0, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES
			(1, 1, 4, 15.0, 0, 1200),
			(2, 2, 4, 14.5, 0, 1100)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS player_locked_songs (
			player_id INTEGER NOT NULL,
			song_id INTEGER NOT NULL,
			is_ultima BOOLEAN NOT NULL,
			PRIMARY KEY (player_id, song_id, is_ultima)
		)
	`)
	require.NoError(t, err)

	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO player_locked_songs (player_id, song_id, is_ultima)
		VALUES (100, 1, 0)
	`)
	require.NoError(t, err)

	repo := NewPlayerDataRepository(db)

	// When
	stats, err := repo.GetOverpowerTargetStatsWithExecutor(ctx, tx, domainrepo.OverpowerTargetFilter{
		ExcludeWorldsend: true,
		ExcludeDeleted:   true,
		PlayerID:         intPtrForPlayerDataRepositoryTest(100),
	})

	// Then
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 1, stats.SongCount)
	assert.InDelta(t, service.CalcSongMaxOP(14.5), stats.MaxOverpowerTotal, 0.0001)
}

func intPtrForPlayerDataRepositoryTest(value int) *int {
	return &value
}

func TestSavePlayerData_execがnilならエラーを返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	repo := NewPlayerDataRepository(db)

	// When
	err := repo.SavePlayerData(context.Background(), nil, domainrepo.PlayerDataSaveInput{})

	// Then
	require.Error(t, err)
	assert.ErrorContains(t, err, "executor")
}

func TestFullRecordChangedCondition_比較対象カラムを過不足なく含む(t *testing.T) {
	// Given
	expected := "score <> VALUES(score) OR clear_lamp_id <> VALUES(clear_lamp_id) OR combo_lamp_id <> VALUES(combo_lamp_id) OR full_chain_id <> VALUES(full_chain_id)"

	// When
	got := fullRecordChangedCondition

	// Then
	assert.Equal(t, expected, got)
}

func TestWorldsendRecordChangedCondition_比較対象カラムを過不足なく含む(t *testing.T) {
	// Given
	expected := "score <> VALUES(score) OR clear_lamp_id <> VALUES(clear_lamp_id) OR combo_lamp_id <> VALUES(combo_lamp_id) OR full_chain_id <> VALUES(full_chain_id)"

	// When
	got := worldsendRecordChangedCondition

	// Then
	assert.Equal(t, expected, got)
}

func TestFullRecordUpsertQuery_updatedAtの評価が比較対象更新より先に行われる(t *testing.T) {
	// Given
	query := fullRecordUpsertQuery

	// When
	updatedAtIndex := strings.Index(query, "updated_at = IF(")
	scoreIndex := strings.Index(query, "score = VALUES(score)")

	// Then
	require.NotEqual(t, -1, updatedAtIndex)
	require.NotEqual(t, -1, scoreIndex)
	assert.Less(t, updatedAtIndex, scoreIndex)
}

func TestWorldsendRecordUpsertQuery_updatedAtの評価が比較対象更新より先に行われる(t *testing.T) {
	// Given
	query := worldsendRecordUpsertQuery

	// When
	updatedAtIndex := strings.Index(query, "updated_at = IF(")
	scoreIndex := strings.Index(query, "score = VALUES(score)")

	// Then
	require.NotEqual(t, -1, updatedAtIndex)
	require.NotEqual(t, -1, scoreIndex)
	assert.Less(t, updatedAtIndex, scoreIndex)
}

func TestReplaceQueryPlaceholder_パーセント記号を含む置換文字列もそのまま埋め込める(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		placeholder   string
		replacement   string
		expectedQuery string
	}{
		{
			name:          "LIKE句のパーセント記号を保持する",
			query:         "SELECT * FROM users WHERE {{CONDITION}}",
			placeholder:   "{{CONDITION}}",
			replacement:   "name LIKE 'abc%'",
			expectedQuery: "SELECT * FROM users WHERE name LIKE 'abc%'",
		},
		{
			name:          "同じプレースホルダをすべて置換する",
			query:         "{{CONDITION}} OR {{CONDITION}}",
			placeholder:   "{{CONDITION}}",
			replacement:   "score >= 1000000",
			expectedQuery: "score >= 1000000 OR score >= 1000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given

			// When
			got := replaceQueryPlaceholder(tt.query, tt.placeholder, tt.replacement)

			// Then
			assert.Equal(t, tt.expectedQuery, got)
		})
	}
}

func TestFindPlayerRecordStatesByChartIDs_保存前状態を譜面IDキーで返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupPlayerRecordRepositoryDB(t, db)
	updatedAt := "2026-04-27T00:00:00Z"
	_, err := db.Exec(`
		INSERT INTO player_records (player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, slot_id, slot_order, updated_at)
		VALUES
			(10, 101, 1000000, 2, 3, 1, 2, 5, ?),
			(10, 102, 990000, 1, 1, 1, 1, NULL, ?),
			(20, 101, 980000, 1, 1, 1, 1, NULL, ?)
	`, updatedAt, updatedAt, updatedAt)
	require.NoError(t, err)
	repo := NewPlayerDataRepository(db)

	// When
	states, err := repo.FindPlayerRecordStatesByChartIDs(context.Background(), db, 10, []int{101, 999})

	// Then
	require.NoError(t, err)
	require.Len(t, states, 1)
	state := states[101]
	assert.Equal(t, 1000000, state.Score)
	assert.Equal(t, 2, state.ClearLampID)
	assert.Equal(t, 3, state.ComboLampID)
	assert.Equal(t, 1, state.FullChainID)
	assert.Equal(t, 2, state.SlotID)
	require.NotNil(t, state.SlotOrder)
	assert.Equal(t, 5, *state.SlotOrder)
}

func TestFindWorldsendRecordStatesByChartIDs_保存前状態を譜面IDキーで返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	setupPlayerRecordRepositoryDB(t, db)
	updatedAt := "2026-04-27T00:00:00Z"
	_, err := db.Exec(`
		INSERT INTO player_worldsend_records (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
		VALUES
			(10, 201, 1000000, 2, 3, 1, ?),
			(20, 201, 980000, 1, 1, 1, ?)
	`, updatedAt, updatedAt)
	require.NoError(t, err)
	repo := NewPlayerDataRepository(db)

	// When
	states, err := repo.FindWorldsendRecordStatesByChartIDs(context.Background(), db, 10, []int{201, 999})

	// Then
	require.NoError(t, err)
	require.Len(t, states, 1)
	state := states[201]
	assert.Equal(t, 1000000, state.Score)
	assert.Equal(t, 2, state.ClearLampID)
	assert.Equal(t, 3, state.ComboLampID)
	assert.Equal(t, 1, state.FullChainID)
}

func TestFindRecordStatesByChartIDs_execがnilならエラーを返す(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	repo := NewPlayerDataRepository(db)

	// When
	fullStates, fullErr := repo.FindPlayerRecordStatesByChartIDs(context.Background(), nil, 10, []int{101})
	worldsendStates, worldsendErr := repo.FindWorldsendRecordStatesByChartIDs(context.Background(), nil, 10, []int{201})

	// Then
	assert.Nil(t, fullStates)
	require.Error(t, fullErr)
	assert.ErrorContains(t, fullErr, "executor")
	assert.Nil(t, worldsendStates)
	require.Error(t, worldsendErr)
	assert.ErrorContains(t, worldsendErr, "executor")
}
