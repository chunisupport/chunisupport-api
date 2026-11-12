package consolidation

import (
	"context"
	"fmt"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/importer"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/songchart"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// mockDifficultyRepo はテスト用の DifficultyRepository モック
type mockDifficultyRepo struct{}

func (m *mockDifficultyRepo) FindAll(ctx context.Context) ([]songbatch.Difficulty, error) {
	return []songbatch.Difficulty{
		{ID: 1, Name: "BASIC"},
		{ID: 2, Name: "ADVANCED"},
		{ID: 3, Name: "EXPERT"},
		{ID: 4, Name: "MASTER"},
		{ID: 5, Name: "ULTIMA"},
	}, nil
}

// mockGenreRepo はテスト用の GenreRepository モック
type mockGenreRepo struct{}

func (m *mockGenreRepo) FindAll(ctx context.Context) ([]songbatch.Genre, error) {
	return []songbatch.Genre{
		{ID: 1, Name: "POPS & ANIME"},
		{ID: 2, Name: "niconico"},
		{ID: 3, Name: "東方Project"},
		{ID: 4, Name: "VARIETY"},
		{ID: 5, Name: "イロドリミドリ"},
		{ID: 6, Name: "ゲキマイ"},
		{ID: 7, Name: "ORIGINAL"},
	}, nil
}

// TestLoadActiveOfficialSongsFromMySQL は loadActiveOfficialSongs が MySQL から正しくデータを読み込むことを確認
func TestLoadActiveOfficialSongsFromMySQL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// テスト用の SQLite DB をモック MySQL として使用
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	// スキーマ作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE songs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			display_id TEXT NOT NULL,
			title TEXT NOT NULL,
			artist TEXT NOT NULL,
			genre_id INTEGER NOT NULL,
			official_idx TEXT,
			is_worldsend INTEGER NOT NULL DEFAULT 0,
			is_deleted INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create songs table: %v", err)
	}

	// テストデータ挿入
	_, err = db.ExecContext(ctx, `
		INSERT INTO songs (display_id, title, artist, genre_id, official_idx, is_deleted)
		VALUES 
			('disp-001', 'Song 1', 'Artist 1', 1, 'OFF001', 0),
			('disp-002', 'Song 2', 'Artist 2', 1, 'OFF002', 0),
			('disp-003', 'Song 3', 'Artist 3', 1, 'OFF003', 1),
			('disp-004', 'Song 4', 'Artist 4', 1, NULL, 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert test songs: %v", err)
	}

	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	consolidator := &OfficialConsolidator{
		db:        db,
		workspace: ws,
	}

	result, err := consolidator.loadActiveOfficialSongs(ctx)
	if err != nil {
		t.Fatalf("loadActiveOfficialSongs returned error: %v", err)
	}

	// 検証: is_deleted=0 かつ official_idx が NULL でない曲のみが返される
	expected := map[string]int{
		"OFF001": 1,
		"OFF002": 2,
	}

	if len(result) != len(expected) {
		t.Errorf("expected %d songs, got %d", len(expected), len(result))
	}

	for idx, expectedID := range expected {
		if gotID, ok := result[idx]; !ok {
			t.Errorf("expected official_idx %s to be present", idx)
		} else if gotID != expectedID {
			t.Errorf("for official_idx %s: expected id=%d, got id=%d", idx, expectedID, gotID)
		}
	}

	// is_deleted=1 と official_idx=NULL の曲は含まれないことを確認
	if _, ok := result["OFF003"]; ok {
		t.Errorf("deleted song OFF003 should not be included")
	}
}

// TestDetectMassiveIdxChange は official_idx の大規模変更を検知することを確認
func TestDetectMassiveIdxChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// テスト用の SQLite DB をモック MySQL として使用
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	// スキーマ作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE songs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			display_id TEXT NOT NULL,
			title TEXT NOT NULL,
			artist TEXT NOT NULL,
			genre_id INTEGER NOT NULL,
			official_idx TEXT,
			is_worldsend INTEGER NOT NULL DEFAULT 0,
			is_deleted INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create songs table: %v", err)
	}

	// 15曲のテストデータを挿入
	for i := 1; i <= 15; i++ {
		_, err = db.ExecContext(ctx, `
			INSERT INTO songs (display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
			VALUES (?, ?, ?, 1, ?, 0, 0)
		`, fmt.Sprintf("disp-%03d", i), fmt.Sprintf("Title %d", i), fmt.Sprintf("Artist %d", i), fmt.Sprintf("OLD%03d", i))
		if err != nil {
			t.Fatalf("failed to insert test song %d: %v", i, err)
		}
	}

	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// 新しいデータ: 10曲以上の official_idx が変更されている
	newData := make(importer.OfficialData, 15)
	for i := range 15 {
		newData[i] = importer.OfficialSong{
			ID:     fmt.Sprintf("NEW%03d", i+1), // OLD001 → NEW001 に変更
			Title:  fmt.Sprintf("Title %d", i+1),
			Artist: fmt.Sprintf("Artist %d", i+1),
		}
	}

	consolidator := &OfficialConsolidator{
		db:        db,
		workspace: ws,
		data:      &newData,
	}

	existingActiveSongs, err := consolidator.loadActiveOfficialSongs(ctx)
	if err != nil {
		t.Fatalf("failed to load active songs: %v", err)
	}

	// 大規模変更検知 - エラーが返されるべき
	err = consolidator.detectMassiveIdxChange(ctx, existingActiveSongs)
	if err == nil {
		t.Fatalf("expected error for massive idx change, got nil")
	}

	// エラーメッセージの確認
	expectedMsg := "massive official_idx change detected"
	if !containsString(err.Error(), expectedMsg) {
		t.Errorf("expected error message to contain '%s', got: %v", expectedMsg, err)
	}
}

// TestDetectMassiveIdxChange_NoChange は変更がない場合にエラーが返されないことを確認
func TestDetectMassiveIdxChange_NoChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE songs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			display_id TEXT NOT NULL,
			title TEXT NOT NULL,
			artist TEXT NOT NULL,
			genre_id INTEGER NOT NULL,
			official_idx TEXT,
			is_worldsend INTEGER NOT NULL DEFAULT 0,
			is_deleted INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create songs table: %v", err)
	}

	// 15曲のテストデータを挿入
	for i := 1; i <= 15; i++ {
		_, err = db.ExecContext(ctx, `
			INSERT INTO songs (display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
			VALUES (?, ?, ?, 1, ?, 0, 0)
		`, fmt.Sprintf("disp-%03d", i), fmt.Sprintf("Title %d", i), fmt.Sprintf("Artist %d", i), fmt.Sprintf("OFF%03d", i))
		if err != nil {
			t.Fatalf("failed to insert test song %d: %v", i, err)
		}
	}

	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// 新しいデータ: official_idx に変更なし
	newData := make(importer.OfficialData, 15)
	for i := range 15 {
		newData[i] = importer.OfficialSong{
			ID:     fmt.Sprintf("OFF%03d", i+1),
			Title:  fmt.Sprintf("Title %d", i+1),
			Artist: fmt.Sprintf("Artist %d", i+1),
		}
	}

	consolidator := &OfficialConsolidator{
		db:        db,
		workspace: ws,
		data:      &newData,
	}

	existingActiveSongs, err := consolidator.loadActiveOfficialSongs(ctx)
	if err != nil {
		t.Fatalf("failed to load active songs: %v", err)
	}

	// 変更なし - エラーが返されないべき
	err = consolidator.detectMassiveIdxChange(ctx, existingActiveSongs)
	if err != nil {
		t.Fatalf("expected no error when no massive change, got: %v", err)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestOfficialConsolidator_Consolidate_EmptyData は空データの場合にエラーにならないことを確認
func TestOfficialConsolidator_Consolidate_EmptyData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// モック用の SQLite DB
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	defer db.Close()

	// 空データ
	var emptyData importer.OfficialData

	consolidator := NewOfficialConsolidator(context.Background(), db, &mockDifficultyRepo{}, &mockGenreRepo{}, ws, &emptyData)

	err = consolidator.Consolidate(ctx)
	if err != nil {
		t.Fatalf("Consolidate with empty data should not return error: %v", err)
	}
}

// TestOfficialConsolidator_Consolidate_NilData は nil データの場合にエラーにならないことを確認
func TestOfficialConsolidator_Consolidate_NilData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	defer db.Close()

	consolidator := NewOfficialConsolidator(context.Background(), db, &mockDifficultyRepo{}, &mockGenreRepo{}, ws, nil)

	err = consolidator.Consolidate(ctx)
	if err != nil {
		t.Fatalf("Consolidate with nil data should not return error: %v", err)
	}
}

// TestOfficialConsolidator_Consolidate_InsertsSongs は楽曲が正しく挿入されることを確認
func TestOfficialConsolidator_Consolidate_InsertsSongs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	// モック MySQL（genres と difficulties を含むスキーマ）
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	defer db.Close()

	// genres と difficulties のテーブルを作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE genres (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);
		INSERT INTO genres (id, name) VALUES
			(1, 'POPS & ANIME'),
			(2, 'niconico'),
			(3, '東方Project'),
			(4, 'VARIETY'),
			(5, 'イロドリミドリ'),
			(6, 'ゲキマイ'),
			(7, 'ORIGINAL');

		CREATE TABLE difficulties (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);
		INSERT INTO difficulties (id, name) VALUES
			(1, 'BASIC'),
			(2, 'ADVANCED'),
			(3, 'EXPERT'),
			(4, 'MASTER'),
			(5, 'ULTIMA');

		CREATE TABLE songs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			official_idx TEXT,
			title TEXT,
			artist TEXT,
			is_deleted INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// テスト用公式データ
	officialData := importer.OfficialData{
		{
			ID:      "OFF001",
			Newflag: "1",
			Title:   "Test Song 1",
			Reading: "てすとそんぐ1",
			Artist:  "Test Artist 1",
			Catname: "POPS & ANIME",
		},
		{
			ID:      "OFF002",
			Title:   "Test Song 2",
			Artist:  "Test Artist 2",
			Catname: "ORIGINAL",
		},
	}

	consolidator := NewOfficialConsolidator(context.Background(), db, &mockDifficultyRepo{}, &mockGenreRepo{}, ws, &officialData)

	err = consolidator.Consolidate(ctx)
	if err != nil {
		t.Fatalf("Consolidate returned error: %v", err)
	}

	// ワークスペースに楽曲が挿入されていることを確認
	var count int
	err = ws.DB().GetContext(ctx, &count, "SELECT COUNT(*) FROM songs")
	if err != nil {
		t.Fatalf("failed to count songs: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 songs, got %d", count)
	}

	// 各楽曲の内容を確認
	var title1, title2 string
	var reading1 string
	var isNew1, isNew2 int
	err = ws.DB().GetContext(ctx, &title1, "SELECT title FROM songs WHERE official_idx = 'OFF001'")
	if err != nil {
		t.Fatalf("failed to get song 1: %v", err)
	}
	err = ws.DB().GetContext(ctx, &reading1, "SELECT reading FROM songs WHERE official_idx = 'OFF001'")
	if err != nil {
		t.Fatalf("failed to get song 1 reading: %v", err)
	}
	err = ws.DB().GetContext(ctx, &title2, "SELECT title FROM songs WHERE official_idx = 'OFF002'")
	if err != nil {
		t.Fatalf("failed to get song 2: %v", err)
	}
	err = ws.DB().GetContext(ctx, &isNew1, "SELECT is_new FROM songs WHERE official_idx = 'OFF001'")
	if err != nil {
		t.Fatalf("failed to get song 1 is_new: %v", err)
	}
	err = ws.DB().GetContext(ctx, &isNew2, "SELECT is_new FROM songs WHERE official_idx = 'OFF002'")
	if err != nil {
		t.Fatalf("failed to get song 2 is_new: %v", err)
	}

	if title1 != "Test Song 1" {
		t.Errorf("expected 'Test Song 1', got '%s'", title1)
	}
	if reading1 != "てすとそんぐ1" {
		t.Errorf("expected 'てすとそんぐ1', got '%s'", reading1)
	}
	if title2 != "Test Song 2" {
		t.Errorf("expected 'Test Song 2', got '%s'", title2)
	}
	if isNew1 != 1 {
		t.Errorf("expected OFF001 is_new=1, got %d", isNew1)
	}
	if isNew2 != 0 {
		t.Errorf("expected OFF002 is_new=0, got %d", isNew2)
	}
}

// TestOfficialConsolidator_Consolidate_SkipsInvalidData は無効なデータをスキップすることを確認
func TestOfficialConsolidator_Consolidate_SkipsInvalidData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	defer db.Close()

	// genres と difficulties のテーブルを作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE genres (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);
		INSERT INTO genres (id, name) VALUES
			(1, 'POPS & ANIME'),
			(7, 'ORIGINAL');

		CREATE TABLE difficulties (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);
		INSERT INTO difficulties (id, name) VALUES
			(1, 'BASIC'),
			(2, 'ADVANCED'),
			(3, 'EXPERT'),
			(4, 'MASTER'),
			(5, 'ULTIMA');

		CREATE TABLE songs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			official_idx TEXT,
			title TEXT,
			artist TEXT,
			is_deleted INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// 無効なデータを含む
	officialData := importer.OfficialData{
		{
			ID:      "", // 空の ID
			Title:   "Song 1",
			Artist:  "Artist 1",
			Catname: "POPS & ANIME",
		},
		{
			ID:      "OFF001",
			Title:   "Valid Song",
			Artist:  "Valid Artist",
			Catname: "UNKNOWN_GENRE", // 不明なジャンル
		},
		{
			ID:      "OFF002",
			Title:   "Another Valid Song",
			Artist:  "Another Artist",
			Catname: "ORIGINAL",
		},
	}

	consolidator := NewOfficialConsolidator(context.Background(), db, &mockDifficultyRepo{}, &mockGenreRepo{}, ws, &officialData)

	err = consolidator.Consolidate(ctx)
	if err != nil {
		t.Fatalf("Consolidate returned error: %v", err)
	}

	// 有効な楽曲のみが挿入されていることを確認
	var count int
	err = ws.DB().GetContext(ctx, &count, "SELECT COUNT(*) FROM songs")
	if err != nil {
		t.Fatalf("failed to count songs: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 valid song, got %d", count)
	}
}

// TestPrepareSongsForUpsert は楽曲レコードの準備が正しく行われることを確認
func TestPrepareSongsForUpsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer ws.Close()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	defer db.Close()

	// genres と difficulties のテーブルを作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE genres (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);
		INSERT INTO genres (id, name) VALUES
			(1, 'POPS & ANIME'),
			(6, 'ゲキマイ');

		CREATE TABLE difficulties (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);
		INSERT INTO difficulties (id, name) VALUES
			(1, 'BASIC'),
			(2, 'ADVANCED'),
			(3, 'EXPERT'),
			(4, 'MASTER'),
			(5, 'ULTIMA')
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	officialData := importer.OfficialData{
		{
			ID:      "OFF001",
			Newflag: "1",
			Title:   "  Trimmed Title  ",
			Reading: "  とりむたいとる  ",
			Artist:  "  Trimmed Artist  ",
			Catname: "POPS & ANIME",
			Image:   "CHU_UI_Jacket_0001.jpg",
		},
		{
			ID:      "OFF002",
			Title:   "World's End Song",
			Artist:  "WE Artist",
			Catname: "ゲキマイ",
			WeKanji: "招",
			WeStar:  "3",
		},
	}

	consolidator := NewOfficialConsolidator(context.Background(), db, &mockDifficultyRepo{}, &mockGenreRepo{}, ws, &officialData)

	songs, seen := consolidator.prepareSongsForUpsert()

	if len(songs) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(songs))
	}

	// タイトルとアーティストがトリムされていることを確認
	if songs[0].Title != "Trimmed Title" {
		t.Errorf("expected trimmed title, got '%s'", songs[0].Title)
	}
	if songs[0].Reading == nil || *songs[0].Reading != "とりむたいとる" {
		t.Errorf("expected trimmed reading, got %v", songs[0].Reading)
	}
	if songs[0].Artist != "Trimmed Artist" {
		t.Errorf("expected trimmed artist, got '%s'", songs[0].Artist)
	}

	// World's End フラグ
	if songs[0].IsWorldsEnd != 0 {
		t.Errorf("expected normal song (is_worldsend=0), got %d", songs[0].IsWorldsEnd)
	}
	if songs[0].IsNew != 1 {
		t.Errorf("expected new song (is_new=1), got %d", songs[0].IsNew)
	}
	if songs[1].IsWorldsEnd != 1 {
		t.Errorf("expected World's End song (is_worldsend=1), got %d", songs[1].IsWorldsEnd)
	}
	if songs[1].IsNew != 0 {
		t.Errorf("expected non-new song (is_new=0), got %d", songs[1].IsNew)
	}

	// seen マップ
	if _, ok := seen["OFF001"]; !ok {
		t.Error("OFF001 should be in seen map")
	}
	if _, ok := seen["OFF002"]; !ok {
		t.Error("OFF002 should be in seen map")
	}
}

// TestDetectMassiveIdxChange_IgnoreWorldsend はWORLD'S END楽曲が標準楽曲と区別されることを確認
func TestDetectMassiveIdxChange_IgnoreWorldsend(t *testing.T) {
	t.Parallel()

	// Given: 通常曲15曲は idx に変化がなく、WORLD'S END 12曲だけが
	// 同じタイトル・アーティストの通常曲として別 idx で登場する。
	// 通常曲だけでも早期終了のしきい値(10件)を超え、WORLD'S END の12件は
	// 除外がなければ検知条件(10件以上または20%以上)に届く量である。
	ctx := context.Background()

	db, err := sqlx.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()

	// スキーマ作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE songs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			display_id TEXT NOT NULL,
			title TEXT NOT NULL,
			artist TEXT NOT NULL,
			genre_id INTEGER NOT NULL,
			official_idx TEXT,
			is_worldsend INTEGER NOT NULL DEFAULT 0,
			is_deleted INTEGER NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, err)

	// 通常曲15曲は DB と新データで official_idx が一致する
	for i := 1; i <= 15; i++ {
		_, err = db.ExecContext(ctx, `
			INSERT INTO songs (display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
			VALUES (?, ?, ?, 1, ?, 0, 0)
		`, fmt.Sprintf("disp-%03d", i), fmt.Sprintf("Title %d", i), fmt.Sprintf("Artist %d", i), fmt.Sprintf("OFF%03d", i))
		require.NoError(t, err)
	}

	// WORLD'S END 12曲は DB 上では is_worldsend=1 で別 idx を持つ
	for i := 1; i <= 12; i++ {
		_, err = db.ExecContext(ctx, `
			INSERT INTO songs (display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
			VALUES (?, ?, ?, 1, ?, 1, 0)
		`, fmt.Sprintf("disp-we-%03d", i), fmt.Sprintf("WE Title %d", i), fmt.Sprintf("WE Artist %d", i), fmt.Sprintf("WE%03d", i))
		require.NoError(t, err)
	}

	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{})
	require.NoError(t, err)
	defer ws.Close()

	// 新データ: 通常曲15曲は idx 不変、WORLD'S END に対応する12曲は
	// 同じタイトル・アーティストの通常曲(WeKanji・WeStar なし)として別 idx で登場する
	newData := make(importer.OfficialData, 0, 27)
	for i := 1; i <= 15; i++ {
		newData = append(newData, importer.OfficialSong{
			ID:     fmt.Sprintf("OFF%03d", i),
			Title:  fmt.Sprintf("Title %d", i),
			Artist: fmt.Sprintf("Artist %d", i),
		})
	}
	for i := 1; i <= 12; i++ {
		newData = append(newData, importer.OfficialSong{
			ID:     fmt.Sprintf("NEW-WE%03d", i),
			Title:  fmt.Sprintf("WE Title %d", i),
			Artist: fmt.Sprintf("WE Artist %d", i),
		})
	}

	consolidator := &OfficialConsolidator{
		db:        db,
		workspace: ws,
		data:      &newData,
	}

	existingActiveSongs, err := consolidator.loadActiveOfficialSongs(ctx)
	require.NoError(t, err)
	// 通常曲15曲と WORLD'S END 12曲の合計27曲が判定対象になる
	assert.Len(t, existingActiveSongs, 27)

	// When: 変更検知ロジックを実行する
	err = consolidator.detectMassiveIdxChange(ctx, existingActiveSongs)

	// Then: WORLD'S END と通常曲は別物として扱われ、検知されない
	assert.NoError(t, err)
}

// 公開 Consolidate を通し、楽曲だけでなく通常・WE譜面の保存と相互混入の防止を保証します。
func TestOfficialConsolidator_Consolidate_SavesChartsAndWorldsendCharts(t *testing.T) {
	t.Parallel()

	// Given: 通常曲1曲と WORLD'S END 曲1曲の公式データ
	ctx := context.Background()
	ws, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{
		DSN: "file:" + t.Name() + "?mode=memory&cache=shared&_pragma=foreign_keys(ON)",
	})
	require.NoError(t, err)
	defer ws.Close()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE songs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			official_idx TEXT,
			title TEXT,
			artist TEXT,
			is_deleted INTEGER DEFAULT 0
		)
	`)
	require.NoError(t, err)

	officialData := importer.OfficialData{
		{
			ID:      "OFF101",
			Title:   "通常曲",
			Artist:  "通常アーティスト",
			Catname: "POPS & ANIME",
			LevBas:  "5",
			LevMas:  "14+",
		},
		{
			ID:      "OFF102",
			Title:   "WE曲",
			Artist:  "WEアーティスト",
			Catname: "ゲキマイ",
			WeKanji: "招",
			WeStar:  "3",
		},
	}

	consolidator := NewOfficialConsolidator(context.Background(), db, &mockDifficultyRepo{}, &mockGenreRepo{}, ws, &officialData)

	// When: 公開 Consolidate を実行する
	err = consolidator.Consolidate(ctx)
	require.NoError(t, err)

	// Then: 両曲の song_id を取得する
	var normalSongID, weSongID int
	err = ws.DB().GetContext(ctx, &normalSongID, "SELECT id FROM songs WHERE official_idx = 'OFF101'")
	require.NoError(t, err)
	err = ws.DB().GetContext(ctx, &weSongID, "SELECT id FROM songs WHERE official_idx = 'OFF102'")
	require.NoError(t, err)

	// Then: 通常曲の charts に BASIC 5.0 と MASTER 14.5 が保存される
	// 空レベル (ADVANCED/EXPERT/ULTIMA) はスキップされるため2行になる
	type chartRow struct {
		DifficultyID   int     `db:"difficulty_id"`
		Const          float64 `db:"const"`
		IsConstUnknown int     `db:"is_const_unknown"`
	}
	var normalCharts []chartRow
	err = ws.DB().SelectContext(ctx, &normalCharts, "SELECT difficulty_id, const, is_const_unknown FROM charts WHERE song_id = ? ORDER BY difficulty_id", normalSongID)
	require.NoError(t, err)
	assert.Equal(t, []chartRow{
		{DifficultyID: 1, Const: 5.0, IsConstUnknown: 0},
		{DifficultyID: 4, Const: 14.5, IsConstUnknown: 1},
	}, normalCharts)

	// Then: WE 曲には通常譜面が保存されない (相互混入なし)
	var weChartCount int
	err = ws.DB().GetContext(ctx, &weChartCount, "SELECT COUNT(*) FROM charts WHERE song_id = ?", weSongID)
	require.NoError(t, err)
	assert.Equal(t, 0, weChartCount)

	// Then: WE 曲の worldsend_charts に星数と漢字が保存される
	// 公式 WeStar "3" は星数 2 に換算される (vo.ParseWeStarFromOfficial の仕様)
	var weChart struct {
		LevelStar *int    `db:"level_star"`
		Attribute *string `db:"attribute"`
	}
	err = ws.DB().GetContext(ctx, &weChart, "SELECT level_star, attribute FROM worldsend_charts WHERE song_id = ?", weSongID)
	require.NoError(t, err)
	require.NotNil(t, weChart.LevelStar)
	assert.Equal(t, 2, *weChart.LevelStar)
	require.NotNil(t, weChart.Attribute)
	assert.Equal(t, "招", *weChart.Attribute)

	// Then: 通常曲には WORLD'S END 譜面が保存されない (相互混入なし)
	var normalWECount int
	err = ws.DB().GetContext(ctx, &normalWECount, "SELECT COUNT(*) FROM worldsend_charts WHERE song_id = ?", normalSongID)
	require.NoError(t, err)
	assert.Equal(t, 0, normalWECount)
}
