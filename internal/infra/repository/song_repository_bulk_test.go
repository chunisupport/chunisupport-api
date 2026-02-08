package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// setupTestDB はテスト用のインメモリSQLiteデータベースをセットアップします。
func setupTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// テスト用のスキーマを作成（SQLite互換）
	schema := `
		CREATE TABLE IF NOT EXISTS genres (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE
		);
		INSERT INTO genres (id, name) VALUES (1, 'ORIGINAL'), (2, 'POPS');

		CREATE TABLE IF NOT EXISTS difficulties (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE
		);
		INSERT INTO difficulties (id, name) VALUES 
			(1, 'BASIC'), (2, 'ADVANCED'), (3, 'EXPERT'), (4, 'MASTER'), (5, 'ULTIMA');

		CREATE TABLE IF NOT EXISTS songs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			display_id TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			artist TEXT NOT NULL,
			genre_id INTEGER NOT NULL,
			bpm INTEGER,
			released_at TEXT,
			official_idx TEXT NOT NULL UNIQUE,
			jacket TEXT,
			is_worldsend INTEGER NOT NULL DEFAULT 0,
			is_deleted INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (genre_id) REFERENCES genres(id)
		);

		CREATE TABLE IF NOT EXISTS charts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			song_id INTEGER NOT NULL,
			difficulty_id INTEGER NOT NULL,
			const REAL NOT NULL,
			is_const_unknown INTEGER NOT NULL DEFAULT 1,
			notes INTEGER,
			FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
			FOREIGN KEY (difficulty_id) REFERENCES difficulties(id),
			UNIQUE (song_id, difficulty_id)
		);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db
}

// TestBulkUpdateSongs_ArgumentOrder はバルク更新時のSQL引数順序が正しいかテストします。
// 複数の楽曲を更新する際、引数の順序が誤っているとデータが入れ違いに更新される問題を検出します。
func TestBulkUpdateSongs_ArgumentOrder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// 初期データを挿入（2曲）
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES 
			(1, 'DISPLAY001', 'Original Title 1', 'Original Artist 1', 1, 180, '2024-01-01', 'IDX001', 'jacket1.png', 0, 0),
			(2, 'DISPLAY002', 'Original Title 2', 'Original Artist 2', 2, 200, '2024-02-01', 'IDX002', 'jacket2.png', 0, 0)
	`)
	require.NoError(t, err)

	// 各楽曲に譜面を追加
	_, err = db.Exec(`
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES 
			(1, 4, 13.5, 0, 1000),
			(2, 4, 14.0, 0, 1200)
	`)
	require.NoError(t, err)

	// リポジトリを作成
	repo := &songRepository{db: db}

	// 更新データを準備（それぞれ異なる値に更新）
	releasedAt1 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	releasedAt2 := time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC)
	jacket1 := "new_jacket1.png"
	jacket2 := "new_jacket2.png"
	genreID1 := 1
	genreID2 := 2
	bpm1 := 150
	bpm2 := 220

	songs := []*entity.Song{
		{
			DisplayID:  "DISPLAY001",
			Title:      "Updated Title 1",
			Artist:     "Updated Artist 1",
			GenreID:    &genreID1,
			BPM:        &bpm1,
			ReleasedAt: &releasedAt1,
			Jacket:     &jacket1,
			Charts:     []*entity.Chart{},
		},
		{
			DisplayID:  "DISPLAY002",
			Title:      "Updated Title 2",
			Artist:     "Updated Artist 2",
			GenreID:    &genreID2,
			BPM:        &bpm2,
			ReleasedAt: &releasedAt2,
			Jacket:     &jacket2,
			Charts:     []*entity.Chart{},
		},
	}

	// DisplayID → SongID マッピングを作成
	displayIDToSongID := map[string]int{
		"DISPLAY001": 1,
		"DISPLAY002": 2,
	}

	// バルク更新を実行
	err = repo.bulkUpdateSongs(ctx, db, songs, displayIDToSongID)
	require.NoError(t, err)

	// 更新結果を検証
	var result []struct {
		ID         int     `db:"id"`
		DisplayID  string  `db:"display_id"`
		Title      string  `db:"title"`
		Artist     string  `db:"artist"`
		GenreID    int     `db:"genre_id"`
		BPM        int     `db:"bpm"`
		ReleasedAt string  `db:"released_at"`
		Jacket     *string `db:"jacket"`
	}

	err = db.Select(&result, "SELECT id, display_id, title, artist, genre_id, bpm, released_at, jacket FROM songs ORDER BY id")
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Song1の検証: 各フィールドが正しい値に更新されているか
	assert.Equal(t, "DISPLAY001", result[0].DisplayID, "Song1: DisplayID should match")
	assert.Equal(t, "Updated Title 1", result[0].Title, "Song1: Title should be 'Updated Title 1'")
	assert.Equal(t, "Updated Artist 1", result[0].Artist, "Song1: Artist should be 'Updated Artist 1'")
	assert.Equal(t, 1, result[0].GenreID, "Song1: GenreID should be 1")
	assert.Equal(t, 150, result[0].BPM, "Song1: BPM should be 150")
	assert.Equal(t, "new_jacket1.png", *result[0].Jacket, "Song1: Jacket should be 'new_jacket1.png'")

	// Song2の検証
	assert.Equal(t, "DISPLAY002", result[1].DisplayID, "Song2: DisplayID should match")
	assert.Equal(t, "Updated Title 2", result[1].Title, "Song2: Title should be 'Updated Title 2'")
	assert.Equal(t, "Updated Artist 2", result[1].Artist, "Song2: Artist should be 'Updated Artist 2'")
	assert.Equal(t, 2, result[1].GenreID, "Song2: GenreID should be 2")
	assert.Equal(t, 220, result[1].BPM, "Song2: BPM should be 220")
	assert.Equal(t, "new_jacket2.png", *result[1].Jacket, "Song2: Jacket should be 'new_jacket2.png'")
}

// TestBulkUpdateCharts_ArgumentOrder はバルク更新時の譜面データのSQL引数順序が正しいかテストします。
func TestBulkUpdateCharts_ArgumentOrder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// 初期データを挿入（2曲）
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES 
			(1, 'DISPLAY001', 'Song 1', 'Artist 1', 1, 180, '2024-01-01', 'IDX001', NULL, 0, 0),
			(2, 'DISPLAY002', 'Song 2', 'Artist 2', 1, 200, '2024-02-01', 'IDX002', NULL, 0, 0)
	`)
	require.NoError(t, err)

	// 各楽曲に2つずつ譜面を追加
	_, err = db.Exec(`
		INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes)
		VALUES 
			(1, 3, 12.0, 1, 800),   -- Song1 EXPERT: const=12.0, notes=800
			(1, 4, 13.5, 1, 1000),  -- Song1 MASTER: const=13.5, notes=1000
			(2, 3, 12.5, 1, 900),   -- Song2 EXPERT: const=12.5, notes=900
			(2, 4, 14.0, 1, 1200)   -- Song2 MASTER: const=14.0, notes=1200
	`)
	require.NoError(t, err)

	// リポジトリを作成
	repo := &songRepository{db: db}

	// 更新データを準備
	notes1Expert := notes.Notes(850)
	notes1Master := notes.Notes(1050)
	notes2Expert := notes.Notes(950)
	notes2Master := notes.Notes(1250)

	songs := []*entity.Song{
		{
			DisplayID: "DISPLAY001",
			Charts: []*entity.Chart{
				{DifficultyID: 3, Const: chartconstant.ChartConstant(12.3), IsConstUnknown: false, Notes: &notes1Expert},
				{DifficultyID: 4, Const: chartconstant.ChartConstant(13.8), IsConstUnknown: false, Notes: &notes1Master},
			},
		},
		{
			DisplayID: "DISPLAY002",
			Charts: []*entity.Chart{
				{DifficultyID: 3, Const: chartconstant.ChartConstant(12.8), IsConstUnknown: false, Notes: &notes2Expert},
				{DifficultyID: 4, Const: chartconstant.ChartConstant(14.3), IsConstUnknown: false, Notes: &notes2Master},
			},
		},
	}

	// DisplayID → SongID マッピング
	displayIDToSongID := map[string]int{
		"DISPLAY001": 1,
		"DISPLAY002": 2,
	}

	// バルク更新を実行
	err = repo.bulkUpdateCharts(ctx, db, songs, displayIDToSongID)
	require.NoError(t, err)

	// 更新結果を検証
	var result []struct {
		SongID         int     `db:"song_id"`
		DifficultyID   int     `db:"difficulty_id"`
		Const          float64 `db:"const"`
		IsConstUnknown bool    `db:"is_const_unknown"`
		Notes          *int    `db:"notes"`
	}

	err = db.Select(&result, "SELECT song_id, difficulty_id, const, is_const_unknown, notes FROM charts ORDER BY song_id, difficulty_id")
	require.NoError(t, err)
	require.Len(t, result, 4)

	// Song1 EXPERT (song_id=1, difficulty_id=3)
	assert.Equal(t, 1, result[0].SongID, "Song1 EXPERT: SongID")
	assert.Equal(t, 3, result[0].DifficultyID, "Song1 EXPERT: DifficultyID")
	assert.InDelta(t, 12.3, result[0].Const, 0.01, "Song1 EXPERT: Const should be 12.3")
	assert.False(t, result[0].IsConstUnknown, "Song1 EXPERT: IsConstUnknown should be false")
	assert.Equal(t, 850, *result[0].Notes, "Song1 EXPERT: Notes should be 850")

	// Song1 MASTER (song_id=1, difficulty_id=4)
	assert.Equal(t, 1, result[1].SongID, "Song1 MASTER: SongID")
	assert.Equal(t, 4, result[1].DifficultyID, "Song1 MASTER: DifficultyID")
	assert.InDelta(t, 13.8, result[1].Const, 0.01, "Song1 MASTER: Const should be 13.8")
	assert.False(t, result[1].IsConstUnknown, "Song1 MASTER: IsConstUnknown should be false")
	assert.Equal(t, 1050, *result[1].Notes, "Song1 MASTER: Notes should be 1050")

	// Song2 EXPERT (song_id=2, difficulty_id=3)
	assert.Equal(t, 2, result[2].SongID, "Song2 EXPERT: SongID")
	assert.Equal(t, 3, result[2].DifficultyID, "Song2 EXPERT: DifficultyID")
	assert.InDelta(t, 12.8, result[2].Const, 0.01, "Song2 EXPERT: Const should be 12.8")
	assert.False(t, result[2].IsConstUnknown, "Song2 EXPERT: IsConstUnknown should be false")
	assert.Equal(t, 950, *result[2].Notes, "Song2 EXPERT: Notes should be 950")

	// Song2 MASTER (song_id=2, difficulty_id=4)
	assert.Equal(t, 2, result[3].SongID, "Song2 MASTER: SongID")
	assert.Equal(t, 4, result[3].DifficultyID, "Song2 MASTER: DifficultyID")
	assert.InDelta(t, 14.3, result[3].Const, 0.01, "Song2 MASTER: Const should be 14.3")
	assert.False(t, result[3].IsConstUnknown, "Song2 MASTER: IsConstUnknown should be false")
	assert.Equal(t, 1250, *result[3].Notes, "Song2 MASTER: Notes should be 1250")
}
