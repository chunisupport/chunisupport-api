package api_v1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// TestToV1WorldsendSongDTO は ToV1WorldsendSongDTO 関数の基本的な変換をテストします。
func TestToV1WorldsendSongDTO(t *testing.T) {
	genreID := 2
	bpm := 200
	jacket := "v1jacket.png"
	releasedAt := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	levelStar := 3
	attribute := "光"
	notesObj, _ := notes.NewNotes(2000)

	song := &entity.Song{
		DisplayID:   "v1test1234567890",
		Title:       "V1テスト楽曲",
		Artist:      "V1アーティスト",
		GenreID:     &genreID,
		BPM:         &bpm,
		ReleasedAt:  &releasedAt,
		OfficialIdx: "456",
		Jacket:      &jacket,
	}

	chart := &entity.WorldsendChart{
		LevelStar: &levelStar,
		Attribute: &attribute,
		Notes:     &notesObj,
	}

	genreNamesByID := map[int]string{
		1: "POPS & ANIME",
		2: "niconico",
	}

	dto := ToV1WorldsendSongDTO(song, chart, genreNamesByID)

	if dto == nil {
		t.Fatal("ToV1WorldsendSongDTO returned nil")
	}

	if dto.DisplayID != "v1test1234567890" {
		t.Errorf("DisplayID = %v, want %v", dto.DisplayID, "v1test1234567890")
	}

	if dto.Title != "V1テスト楽曲" {
		t.Errorf("Title = %v, want %v", dto.Title, "V1テスト楽曲")
	}

	if dto.Artist != "V1アーティスト" {
		t.Errorf("Artist = %v, want %v", dto.Artist, "V1アーティスト")
	}

	if dto.Genre == nil {
		t.Error("Genre is nil, want niconico")
	} else if *dto.Genre != "niconico" {
		t.Errorf("Genre = %v, want %v", *dto.Genre, "niconico")
	}

	if dto.BPM == nil || *dto.BPM != 200 {
		t.Errorf("BPM = %v, want %v", dto.BPM, 200)
	}

	if dto.Release == nil {
		t.Error("Release is nil")
	} else if *dto.Release != "2023-12-31" {
		t.Errorf("Release = %v, want %v", *dto.Release, "2023-12-31")
	}

	if dto.Jacket == nil {
		t.Error("Jacket is nil")
	} else if *dto.Jacket != "v1jacket.png" {
		t.Errorf("Jacket = %v, want %v", *dto.Jacket, "v1jacket.png")
	}

	if dto.OfficialIdx != "456" {
		t.Errorf("OfficialIdx = %v, want %v", dto.OfficialIdx, "456")
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

	if weChart.LevelStar == nil || *weChart.LevelStar != 3 {
		t.Errorf("LevelStar = %v, want %v", weChart.LevelStar, 3)
	}

	if weChart.Attribute == nil || *weChart.Attribute != "光" {
		t.Errorf("Attribute = %v, want %v", weChart.Attribute, "光")
	}

	if weChart.Notes == nil || *weChart.Notes != 2000 {
		t.Errorf("Notes = %v, want %v", weChart.Notes, 2000)
	}
}

// TestToV1WorldsendSongDTO_NilSong は Song が nil の場合に nil を返すことを確認します。
func TestToV1WorldsendSongDTO_NilSong(t *testing.T) {
	dto := ToV1WorldsendSongDTO(nil, nil, map[int]string{})
	if dto != nil {
		t.Errorf("expected nil, got %v", dto)
	}
}

// TestV1WorldsendSongDTO_JSONMarshal は V1WorldsendSongDTO のJSONマーシャリングをテストします。
func TestV1WorldsendSongDTO_JSONMarshal(t *testing.T) {
	releaseDate := "2024-06-01"
	jacket := "we_jacket.png"
	bpm := 160
	genre := "VARIETY"
	levelStar := 4
	attribute := "蔵"
	notesVal := 800

	songDTO := &V1WorldsendSongDTO{
		DisplayID:   "v1we123456789012",
		Title:       "V1 WE テスト",
		Artist:      "V1 WE アーティスト",
		Genre:       &genre,
		BPM:         &bpm,
		Release:     &releaseDate,
		Jacket:      &jacket,
		OfficialIdx: "789",
		Charts: map[string]*V1WorldsendChartDTO{
			"WORLDSEND": {
				Attribute: &attribute,
				LevelStar: &levelStar,
				Notes:     &notesVal,
			},
		},
	}

	jsonBytes, err := json.Marshal(songDTO)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonString := string(jsonBytes)

	// release フィールド名であることを確認
	if !containsString(jsonString, `"release":"2024-06-01"`) {
		t.Errorf("JSON should contain 'release' field, got: %s", jsonString)
	}

	// genre がジャンル名であることを確認
	if !containsString(jsonString, `"genre":"VARIETY"`) {
		t.Errorf("JSON should contain genre name, got: %s", jsonString)
	}

	// official_idx が含まれること
	if !containsString(jsonString, `"official_idx":"789"`) {
		t.Errorf("JSON should contain official_idx, got: %s", jsonString)
	}

	// charts.WORLDSEND が含まれること
	if !containsString(jsonString, `"WORLDSEND"`) {
		t.Errorf("JSON should contain 'WORLDSEND' key, got: %s", jsonString)
	}

	// attribute, level_star, notes が含まれること
	if !containsString(jsonString, `"attribute":"蔵"`) {
		t.Errorf("JSON should contain attribute, got: %s", jsonString)
	}

	if !containsString(jsonString, `"level_star":4`) {
		t.Errorf("JSON should contain level_star, got: %s", jsonString)
	}

	if !containsString(jsonString, `"notes":800`) {
		t.Errorf("JSON should contain notes, got: %s", jsonString)
	}
}
