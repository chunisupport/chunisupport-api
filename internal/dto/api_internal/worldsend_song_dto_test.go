package api_internal

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToWorldsendSongDTO は ToWorldsendSongDTO 関数の基本的な変換をテストします。
func TestToWorldsendSongDTO(t *testing.T) {
	genreID := 1
	bpm := 180
	jacket := "jacket.png"
	reading := "テストガッキョク"
	releasedAt := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	levelStar, levelStarErr := levelstar.NewLevelStar(5)
	if levelStarErr != nil {
		require.Failf(t, "前提条件失敗", "levelstar.NewLevelStar failed: %v", levelStarErr)
	}
	attribute := "狂"
	notesObj, _ := notes.NewNotes(1500)
	notesDesigner := "譜面作者A"

	song := &entity.Song{
		DisplayID:   "0123456789abcdef",
		Title:       "テスト楽曲",
		Reading:     &reading,
		Artist:      "テストアーティスト",
		GenreID:     &genreID,
		BPM:         &bpm,
		ReleasedAt:  &releasedAt,
		OfficialIdx: "123",
		Jacket:      &jacket,
	}

	chart := &entity.WorldsendChart{
		LevelStar:     &levelStar,
		Attribute:     &attribute,
		Notes:         &notesObj,
		NotesDesigner: &notesDesigner,
	}

	genreNamesByID := map[int]string{
		1: "POPS & ANIME",
		2: "niconico",
	}

	dto := ToWorldsendSongDTO(song, chart, genreNamesByID)

	if dto == nil {
		require.Fail(t, "ToWorldsendSongDTO returned nil")
	}

	if dto.DisplayID != "0123456789abcdef" {
		assert.Failf(t, "アサーション失敗", "DisplayID = %v, want %v", dto.DisplayID, "0123456789abcdef")
	}

	if dto.Title != "テスト楽曲" {
		assert.Failf(t, "アサーション失敗", "Title = %v, want %v", dto.Title, "テスト楽曲")
	}

	require.NotNil(t, dto.Reading)
	assert.Equal(t, "テストガッキョク", *dto.Reading)

	if dto.Artist != "テストアーティスト" {
		assert.Failf(t, "アサーション失敗", "Artist = %v, want %v", dto.Artist, "テストアーティスト")
	}

	// Genre は *string でジャンル名に変換される
	if dto.Genre == nil {
		t.Error("Genre is nil, want POPS & ANIME")
	} else if *dto.Genre != "POPS & ANIME" {
		assert.Failf(t, "アサーション失敗", "Genre = %v, want %v", *dto.Genre, "POPS & ANIME")
	}

	if dto.BPM == nil || *dto.BPM != 180 {
		assert.Failf(t, "アサーション失敗", "BPM = %v, want %v", dto.BPM, 180)
	}

	// Release は *string で "YYYY-MM-DD" 形式
	if dto.Release == nil {
		t.Error("Release is nil")
	} else if *dto.Release != "2024-01-15" {
		assert.Failf(t, "アサーション失敗", "Release = %v, want %v", *dto.Release, "2024-01-15")
	}

	if dto.Jacket == nil {
		t.Error("Jacket is nil")
	} else if *dto.Jacket != "jacket.png" {
		assert.Failf(t, "アサーション失敗", "Jacket = %v, want %v", *dto.Jacket, "jacket.png")
	}

	if dto.OfficialIdx != "123" {
		assert.Failf(t, "アサーション失敗", "OfficialIdx = %v, want %v", dto.OfficialIdx, "123")
	}

	// Charts に WORLDSEND キーが存在すること
	if dto.Charts == nil {
		require.Fail(t, "Charts is nil")
	}

	weChart, ok := dto.Charts["WORLDSEND"]
	if !ok {
		require.Fail(t, "Charts does not contain WORLDSEND key")
	}

	if weChart == nil {
		require.Fail(t, "WORLDSEND chart is nil")
	}

	if weChart.LevelStar == nil || *weChart.LevelStar != 5 {
		assert.Failf(t, "アサーション失敗", "LevelStar = %v, want %v", weChart.LevelStar, 5)
	}

	if weChart.Attribute == nil || *weChart.Attribute != "狂" {
		assert.Failf(t, "アサーション失敗", "Attribute = %v, want %v", weChart.Attribute, "狂")
	}

	if weChart.Notes == nil || *weChart.Notes != 1500 {
		assert.Failf(t, "アサーション失敗", "Notes = %v, want %v", weChart.Notes, 1500)
	}
	if weChart.NotesDesigner == nil || *weChart.NotesDesigner != "譜面作者A" {
		assert.Failf(t, "アサーション失敗", "NotesDesigner = %v, want %v", weChart.NotesDesigner, "譜面作者A")
	}
}

// TestToWorldsendSongDTO_ReleaseDateCanBeNil はリリース日がnilの場合のテストです。
func TestToWorldsendSongDTO_ReleaseDateCanBeNil(t *testing.T) {
	song := &entity.Song{
		DisplayID:   "0123456789abcdef",
		Title:       "test song",
		Artist:      "test artist",
		OfficialIdx: "123",
	}

	dto := ToWorldsendSongDTO(song, nil, map[int]string{})
	if dto.Release != nil {
		assert.Failf(t, "アサーション失敗", "Release = %v, want nil", *dto.Release)
	}
}

// TestToWorldsendSongDTO_NilSong は Song が nil の場合に nil を返すことを確認します。
func TestToWorldsendSongDTO_NilSong(t *testing.T) {
	dto := ToWorldsendSongDTO(nil, nil, map[int]string{})
	if dto != nil {
		assert.Failf(t, "アサーション失敗", "expected nil, got %v", dto)
	}
}

// TestToWorldsendChartDTO は ToWorldsendChartDTO 関数の基本的な変換をテストします。
func TestToWorldsendChartDTO(t *testing.T) {
	levelStar, levelStarErr := levelstar.NewLevelStar(3)
	if levelStarErr != nil {
		require.Failf(t, "前提条件失敗", "levelstar.NewLevelStar failed: %v", levelStarErr)
	}
	attribute := "光"
	notesObj, _ := notes.NewNotes(2000)
	notesDesigner := "譜面作者B"

	chart := &entity.WorldsendChart{
		LevelStar:     &levelStar,
		Attribute:     &attribute,
		Notes:         &notesObj,
		NotesDesigner: &notesDesigner,
	}

	dto := ToWorldsendChartDTO(chart)

	if dto == nil {
		require.Fail(t, "ToWorldsendChartDTO returned nil")
	}

	if dto.LevelStar == nil || *dto.LevelStar != 3 {
		assert.Failf(t, "アサーション失敗", "LevelStar = %v, want %v", dto.LevelStar, 3)
	}

	if dto.Attribute == nil || *dto.Attribute != "光" {
		assert.Failf(t, "アサーション失敗", "Attribute = %v, want %v", dto.Attribute, "光")
	}

	if dto.Notes == nil || *dto.Notes != 2000 {
		assert.Failf(t, "アサーション失敗", "Notes = %v, want %v", dto.Notes, 2000)
	}
	if dto.NotesDesigner == nil || *dto.NotesDesigner != "譜面作者B" {
		assert.Failf(t, "アサーション失敗", "NotesDesigner = %v, want %v", dto.NotesDesigner, "譜面作者B")
	}
}

// TestToWorldsendChartDTO_NilChart は Chart が nil の場合に nil を返すことを確認します。
func TestToWorldsendChartDTO_NilChart(t *testing.T) {
	dto := ToWorldsendChartDTO(nil)
	if dto != nil {
		assert.Failf(t, "アサーション失敗", "expected nil, got %v", dto)
	}
}

// TestWorldsendSongDTO_JSONMarshal は WorldsendSongDTO のJSONマーシャリングをテストします。
// charts内に "WORLDSEND" キーが含まれることを確認します。
func TestWorldsendSongDTO_JSONMarshal(t *testing.T) {
	releaseDate := "2024-01-15"
	jacket := "jacket123"
	bpm := 180
	genre := "POPS & ANIME"
	reading := "テストガッキョク"
	levelStar := 5
	attribute := "狂"
	notesVal := 1500
	notesDesigner := "譜面作者C"

	songDTO := &WorldsendSongDTO{
		DisplayID:   "0123456789abcdef",
		Title:       "テスト楽曲",
		Reading:     &reading,
		Artist:      "テストアーティスト",
		Genre:       &genre,
		BPM:         &bpm,
		Release:     &releaseDate,
		Jacket:      &jacket,
		OfficialIdx: "123",
		Charts: map[string]*WorldsendChartDTO{
			"WORLDSEND": {
				Attribute:     &attribute,
				LevelStar:     &levelStar,
				Notes:         &notesVal,
				NotesDesigner: &notesDesigner,
			},
		},
	}

	jsonBytes, err := json.Marshal(songDTO)
	if err != nil {
		require.Failf(t, "前提条件失敗", "json.Marshal failed: %v", err)
	}

	jsonString := string(jsonBytes)

	// release フィールド名であることを確認（released_at ではない）
	if !strings.Contains(jsonString, `"release":"2024-01-15"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'release' field, got: %s", jsonString)
	}

	if !strings.Contains(jsonString, `"reading":"テストガッキョク"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain reading field, got: %s", jsonString)
	}

	// genre がジャンル名であることを確認（genre_id ではない）
	// Goの json.Marshal は & を \u0026 にエスケープするため、エスケープ後の文字列で検証
	if !strings.Contains(jsonString, `"genre":"POPS \u0026 ANIME"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain genre name, got: %s", jsonString)
	}

	// official_idx が含まれること
	if !strings.Contains(jsonString, `"official_idx":"123"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain official_idx, got: %s", jsonString)
	}

	// charts.WORLDSEND が含まれること
	if !strings.Contains(jsonString, `"WORLDSEND"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'WORLDSEND' key, got: %s", jsonString)
	}

	// attribute, level_star, notes が charts 内に含まれること
	if !strings.Contains(jsonString, `"attribute":"狂"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain attribute, got: %s", jsonString)
	}

	if !strings.Contains(jsonString, `"level_star":5`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain level_star, got: %s", jsonString)
	}

	if !strings.Contains(jsonString, `"notes":1500`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain notes, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"notes_designer":"譜面作者C"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain notes_designer, got: %s", jsonString)
	}
}
