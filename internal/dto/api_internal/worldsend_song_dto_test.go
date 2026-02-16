package api_internal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// TestToWorldsendSongDTO は ToWorldsendSongDTO 関数の基本的な変換をテストします。
func TestToWorldsendSongDTO(t *testing.T) {
	genreID := 1
	bpm := 180
	jacket := "jacket.png"
	releasedAt := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	weStar := 5
	weKanji := "狂"
	notesObj, _ := notes.NewNotes(1500)

	song := &entity.Song{
		DisplayID:   "0123456789abcdef",
		Title:       "テスト楽曲",
		Artist:      "テストアーティスト",
		GenreID:     &genreID,
		BPM:         &bpm,
		ReleasedAt:  &releasedAt,
		OfficialIdx: "123",
		Jacket:      &jacket,
	}

	chart := &entity.WorldsendChart{
		WeStar:  &weStar,
		WeKanji: &weKanji,
		Notes:   &notesObj,
	}

	genreNamesByID := map[int]string{
		1: "POPS & ANIME",
		2: "niconico",
	}

	dto := ToWorldsendSongDTO(song, chart, genreNamesByID)

	if dto == nil {
		t.Fatal("ToWorldsendSongDTO returned nil")
	}

	if dto.DisplayID != "0123456789abcdef" {
		t.Errorf("DisplayID = %v, want %v", dto.DisplayID, "0123456789abcdef")
	}

	if dto.Title != "テスト楽曲" {
		t.Errorf("Title = %v, want %v", dto.Title, "テスト楽曲")
	}

	if dto.Artist != "テストアーティスト" {
		t.Errorf("Artist = %v, want %v", dto.Artist, "テストアーティスト")
	}

	// Genre は *string でジャンル名に変換される
	if dto.Genre == nil {
		t.Error("Genre is nil, want POPS & ANIME")
	} else if *dto.Genre != "POPS & ANIME" {
		t.Errorf("Genre = %v, want %v", *dto.Genre, "POPS & ANIME")
	}

	if dto.BPM == nil || *dto.BPM != 180 {
		t.Errorf("BPM = %v, want %v", dto.BPM, 180)
	}

	// Release は *string で "YYYY-MM-DD" 形式
	if dto.Release == nil {
		t.Error("Release is nil")
	} else if *dto.Release != "2024-01-15" {
		t.Errorf("Release = %v, want %v", *dto.Release, "2024-01-15")
	}

	if dto.Jacket == nil {
		t.Error("Jacket is nil")
	} else if *dto.Jacket != "jacket.png" {
		t.Errorf("Jacket = %v, want %v", *dto.Jacket, "jacket.png")
	}

	if dto.OfficialIdx != "123" {
		t.Errorf("OfficialIdx = %v, want %v", dto.OfficialIdx, "123")
	}

	// MaxOP は WORLD'S END では常に null
	if dto.MaxOP != nil {
		t.Errorf("MaxOP = %v, want nil", *dto.MaxOP)
	}

	// Charts に WORLDSEND キーが存在すること
	if dto.Charts == nil {
		t.Fatal("Charts is nil")
	}

	weChart, ok := dto.Charts["WORLDSEND"]
	if !ok {
		t.Fatal("Charts does not contain WORLDSEND key")
	}

	if weChart == nil {
		t.Fatal("WORLDSEND chart is nil")
	}

	if weChart.WeStar == nil || *weChart.WeStar != 5 {
		t.Errorf("WeStar = %v, want %v", weChart.WeStar, 5)
	}

	if weChart.WeKanji == nil || *weChart.WeKanji != "狂" {
		t.Errorf("WeKanji = %v, want %v", weChart.WeKanji, "狂")
	}

	if weChart.Notes == nil || *weChart.Notes != 1500 {
		t.Errorf("Notes = %v, want %v", weChart.Notes, 1500)
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
		t.Errorf("Release = %v, want nil", *dto.Release)
	}
}

// TestToWorldsendSongDTO_NilSong は Song が nil の場合に nil を返すことを確認します。
func TestToWorldsendSongDTO_NilSong(t *testing.T) {
	dto := ToWorldsendSongDTO(nil, nil, map[int]string{})
	if dto != nil {
		t.Errorf("expected nil, got %v", dto)
	}
}

// TestToWorldsendChartDTO は ToWorldsendChartDTO 関数の基本的な変換をテストします。
func TestToWorldsendChartDTO(t *testing.T) {
	weStar := 3
	weKanji := "光"
	notesObj, _ := notes.NewNotes(2000)

	chart := &entity.WorldsendChart{
		WeStar:  &weStar,
		WeKanji: &weKanji,
		Notes:   &notesObj,
	}

	dto := ToWorldsendChartDTO(chart)

	if dto == nil {
		t.Fatal("ToWorldsendChartDTO returned nil")
	}

	if dto.WeStar == nil || *dto.WeStar != 3 {
		t.Errorf("WeStar = %v, want %v", dto.WeStar, 3)
	}

	if dto.WeKanji == nil || *dto.WeKanji != "光" {
		t.Errorf("WeKanji = %v, want %v", dto.WeKanji, "光")
	}

	if dto.Notes == nil || *dto.Notes != 2000 {
		t.Errorf("Notes = %v, want %v", dto.Notes, 2000)
	}
}

// TestToWorldsendChartDTO_NilChart は Chart が nil の場合に nil を返すことを確認します。
func TestToWorldsendChartDTO_NilChart(t *testing.T) {
	dto := ToWorldsendChartDTO(nil)
	if dto != nil {
		t.Errorf("expected nil, got %v", dto)
	}
}

// TestWorldsendSongDTO_JSONMarshal は WorldsendSongDTO のJSONマーシャリングをテストします。
// charts内に "WORLDSEND" キーが含まれること、maxop が null であることを確認します。
func TestWorldsendSongDTO_JSONMarshal(t *testing.T) {
	releaseDate := "2024-01-15"
	jacket := "jacket123"
	bpm := 180
	genre := "POPS & ANIME"
	weStar := 5
	weKanji := "狂"
	notesVal := 1500

	songDTO := &WorldsendSongDTO{
		DisplayID:   "0123456789abcdef",
		Title:       "テスト楽曲",
		Artist:      "テストアーティスト",
		Genre:       &genre,
		BPM:         &bpm,
		Release:     &releaseDate,
		Jacket:      &jacket,
		OfficialIdx: "123",
		MaxOP:       nil,
		Charts: map[string]*WorldsendChartDTO{
			"WORLDSEND": {
				WeKanji: &weKanji,
				WeStar:  &weStar,
				Notes:   &notesVal,
			},
		},
	}

	jsonBytes, err := json.Marshal(songDTO)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonString := string(jsonBytes)

	// maxop が null であることを確認
	if !containsString(jsonString, `"maxop":null`) {
		t.Errorf("JSON should contain \"maxop\":null, got: %s", jsonString)
	}

	// release フィールド名であることを確認（released_at ではない）
	if !containsString(jsonString, `"release":"2024-01-15"`) {
		t.Errorf("JSON should contain 'release' field, got: %s", jsonString)
	}

	// genre がジャンル名であることを確認（genre_id ではない）
	// Goの json.Marshal は & を \u0026 にエスケープするため、エスケープ後の文字列で検証
	if !containsString(jsonString, `"genre":"POPS \u0026 ANIME"`) {
		t.Errorf("JSON should contain genre name, got: %s", jsonString)
	}

	// official_idx が含まれること
	if !containsString(jsonString, `"official_idx":"123"`) {
		t.Errorf("JSON should contain official_idx, got: %s", jsonString)
	}

	// charts.WORLDSEND が含まれること
	if !containsString(jsonString, `"WORLDSEND"`) {
		t.Errorf("JSON should contain 'WORLDSEND' key, got: %s", jsonString)
	}

	// we_kanji, we_star, notes が charts 内に含まれること
	if !containsString(jsonString, `"we_kanji":"狂"`) {
		t.Errorf("JSON should contain we_kanji, got: %s", jsonString)
	}

	if !containsString(jsonString, `"we_star":5`) {
		t.Errorf("JSON should contain we_star, got: %s", jsonString)
	}

	if !containsString(jsonString, `"notes":1500`) {
		t.Errorf("JSON should contain notes, got: %s", jsonString)
	}
}
