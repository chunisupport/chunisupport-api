package api_v1

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// TestToV1WorldsendSongDTO は ToV1WorldsendSongDTO 関数の基本的な変換をテストします。
func TestToV1WorldsendSongDTO(t *testing.T) {
	genreID := 2
	bpm := 200
	jacket := "v1jacket.png"
	reading := "ブイワンテストガッキョク"
	releasedAt := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	levelStar, levelStarErr := levelstar.NewLevelStar(3)
	if levelStarErr != nil {
		require.Failf(t, "前提条件失敗", "levelstar.NewLevelStar failed: %v", levelStarErr)
	}
	attribute := "光"
	notesObj, _ := notes.NewNotes(2000)
	notesDesigner := "譜面作者A"

	song := &entity.Song{
		DisplayID:   "v1test1234567890",
		Title:       "V1テスト楽曲",
		Reading:     &reading,
		Artist:      "V1アーティスト",
		GenreID:     &genreID,
		BPM:         &bpm,
		ReleasedAt:  &releasedAt,
		OfficialIdx: "456",
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

	dto := ToV1WorldsendSongDTO(song, chart, genreNamesByID)

	if dto == nil {
		require.Fail(t, "ToV1WorldsendSongDTO returned nil")
	}

	if dto.DisplayID != "v1test1234567890" {
		assert.Failf(t, "アサーション失敗", "DisplayID = %v, want %v", dto.DisplayID, "v1test1234567890")
	}

	if dto.Title != "V1テスト楽曲" {
		assert.Failf(t, "アサーション失敗", "Title = %v, want %v", dto.Title, "V1テスト楽曲")
	}

	if dto.Reading == nil || *dto.Reading != "ブイワンテストガッキョク" {
		assert.Failf(t, "アサーション失敗", "Reading = %v, want %v", dto.Reading, "ブイワンテストガッキョク")
	}

	if dto.Artist != "V1アーティスト" {
		assert.Failf(t, "アサーション失敗", "Artist = %v, want %v", dto.Artist, "V1アーティスト")
	}

	if dto.Genre == nil {
		t.Error("Genre is nil, want niconico")
	} else if *dto.Genre != "niconico" {
		assert.Failf(t, "アサーション失敗", "Genre = %v, want %v", *dto.Genre, "niconico")
	}

	if dto.BPM == nil || *dto.BPM != 200 {
		assert.Failf(t, "アサーション失敗", "BPM = %v, want %v", dto.BPM, 200)
	}

	if dto.Release == nil {
		t.Error("Release is nil")
	} else if *dto.Release != "2023-12-31" {
		assert.Failf(t, "アサーション失敗", "Release = %v, want %v", *dto.Release, "2023-12-31")
	}

	if dto.Jacket == nil {
		t.Error("Jacket is nil")
	} else if *dto.Jacket != "v1jacket.png" {
		assert.Failf(t, "アサーション失敗", "Jacket = %v, want %v", *dto.Jacket, "v1jacket.png")
	}

	if dto.OfficialIdx != "456" {
		assert.Failf(t, "アサーション失敗", "OfficialIdx = %v, want %v", dto.OfficialIdx, "456")
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

	if weChart.LevelStar == nil || *weChart.LevelStar != 3 {
		assert.Failf(t, "アサーション失敗", "LevelStar = %v, want %v", weChart.LevelStar, 3)
	}

	if weChart.Attribute == nil || *weChart.Attribute != "光" {
		assert.Failf(t, "アサーション失敗", "Attribute = %v, want %v", weChart.Attribute, "光")
	}

	if weChart.Notes == nil || *weChart.Notes != 2000 {
		assert.Failf(t, "アサーション失敗", "Notes = %v, want %v", weChart.Notes, 2000)
	}
	if weChart.NotesDesigner == nil || *weChart.NotesDesigner != "譜面作者A" {
		assert.Failf(t, "アサーション失敗", "NotesDesigner = %v, want %v", weChart.NotesDesigner, "譜面作者A")
	}
}

// TestToV1WorldsendSongDTO_NilSong は Song が nil の場合に nil を返すことを確認します。
func TestToV1WorldsendSongDTO_NilSong(t *testing.T) {
	dto := ToV1WorldsendSongDTO(nil, nil, map[int]string{})
	if dto != nil {
		assert.Failf(t, "アサーション失敗", "expected nil, got %v", dto)
	}
}

// TestV1WorldsendSongDTO_JSONMarshal は V1WorldsendSongDTO のJSONマーシャリングをテストします。
func TestV1WorldsendSongDTO_JSONMarshal(t *testing.T) {
	releaseDate := "2024-06-01"
	jacket := "we_jacket.png"
	bpm := 160
	genre := "VARIETY"
	reading := "ブイワンワールドエンドテスト"
	levelStar := 4
	attribute := "蔵"
	notesVal := 800
	notesDesigner := "譜面作者B"

	songDTO := &V1WorldsendSongDTO{
		DisplayID:   "v1we123456789012",
		Title:       "V1 WE テスト",
		Reading:     &reading,
		Artist:      "V1 WE アーティスト",
		Genre:       &genre,
		BPM:         &bpm,
		Release:     &releaseDate,
		Jacket:      &jacket,
		OfficialIdx: "789",
		Charts: map[string]*V1WorldsendChartDTO{
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

	// release フィールド名であることを確認
	if !containsString(jsonString, `"release":"2024-06-01"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'release' field, got: %s", jsonString)
	}

	if !containsString(jsonString, `"reading":"ブイワンワールドエンドテスト"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain reading field, got: %s", jsonString)
	}

	// genre がジャンル名であることを確認
	if !containsString(jsonString, `"genre":"VARIETY"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain genre name, got: %s", jsonString)
	}

	// official_idx が含まれること
	if !containsString(jsonString, `"official_idx":"789"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain official_idx, got: %s", jsonString)
	}

	// charts.WORLDSEND が含まれること
	if !containsString(jsonString, `"WORLDSEND"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'WORLDSEND' key, got: %s", jsonString)
	}

	// attribute, level_star, notes が含まれること
	if !containsString(jsonString, `"attribute":"蔵"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain attribute, got: %s", jsonString)
	}

	if !containsString(jsonString, `"level_star":4`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain level_star, got: %s", jsonString)
	}

	if !containsString(jsonString, `"notes":800`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain notes, got: %s", jsonString)
	}
	if !containsString(jsonString, `"notes_designer":"譜面作者B"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain notes_designer, got: %s", jsonString)
	}
}
