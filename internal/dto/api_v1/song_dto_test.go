package api_v1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// TestToV1SongDTO はToV1SongDTO関数の基本的な変換をテストします。
func TestToV1SongDTO(t *testing.T) {
	// テストデータの準備
	genreID := 2
	bpm := 200
	imgURL := "https://example.com/v1jacket.jpg"
	reading := "ブイワンテストガッキョク"
	releaseDate := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	masterConst, _ := chartconstant.NewChartConstant(13.0)
	ultimaConst, _ := chartconstant.NewChartConstant(15.0)

	song := &entity.Song{
		DisplayID:      "test123456789012",
		Title:          "テスト楽曲",
		Reading:        &reading,
		Artist:         "テストアーティスト",
		GenreID:        &genreID,
		BPM:            &bpm,
		ReleasedAt:     &releaseDate,
		Jacket:         &imgURL,
		IsMaxOPUnknown: true,
		Charts: []*entity.Chart{
			{DifficultyID: 4, Const: masterConst},
			{DifficultyID: 5, Const: ultimaConst},
		},
	}

	genreNamesByID := map[int]string{
		1: "POPS & ANIME",
		2: "niconico",
	}

	// 変換実行
	dto := ToV1SongDTO(song, genreNamesByID, 90)

	// アサーション
	if dto == nil {
		t.Fatal("ToV1SongDTO returned nil")
	}

	if dto.DisplayID != "test123456789012" {
		t.Errorf("DisplayID = %v, want %v", dto.DisplayID, "test123456789012")
	}

	if dto.Title != "テスト楽曲" {
		t.Errorf("Title = %v, want %v", dto.Title, "テスト楽曲")
	}

	if dto.Reading == nil || *dto.Reading != "ブイワンテストガッキョク" {
		t.Errorf("Reading = %v, want %v", dto.Reading, "ブイワンテストガッキョク")
	}

	if dto.Artist != "テストアーティスト" {
		t.Errorf("Artist = %v, want %v", dto.Artist, "テストアーティスト")
	}

	// Genre は *string なので null チェック
	if dto.Genre == nil {
		t.Error("Genre is nil, want niconico")
	} else if *dto.Genre != "niconico" {
		t.Errorf("Genre = %v, want %v", *dto.Genre, "niconico")
	}

	if dto.BPM == nil || *dto.BPM != 200 {
		t.Errorf("BPM = %v, want %v", dto.BPM, 200)
	}

	// Release は *string で "YYYY-MM-DD" 形式
	if dto.Release == nil {
		t.Error("Release is nil")
	} else if *dto.Release != "2023-12-31" {
		t.Errorf("Release = %v, want %v", *dto.Release, "2023-12-31")
	}

	// Jacket (旧 Img)
	if dto.Jacket == nil {
		t.Error("Jacket is nil")
	} else if *dto.Jacket != "https://example.com/v1jacket.jpg" {
		t.Errorf("Jacket = %v, want %v", *dto.Jacket, "https://example.com/v1jacket.jpg")
	}

	if dto.MaxOP != 90 {
		t.Errorf("MaxOP = %v, want %v", dto.MaxOP, 90.0)
	}

	// IsMaxOPUnknown が反映されていることを確認
	if !dto.IsMaxOPUnknown {
		t.Errorf("IsMaxOPUnknown = %v, want %v", dto.IsMaxOPUnknown, true)
	}

	// Charts は空の map として初期化される
	if dto.Charts == nil {
		t.Error("Charts is nil, want empty map")
	}
}

// TestToV1ChartDTO はToV1ChartDTO関数の基本的な変換をテストします。
func TestToV1ChartDTO(t *testing.T) {
	// テストデータの準備
	notesValue := 999
	notesObj, err := notes.NewNotes(notesValue)
	if err != nil {
		t.Fatalf("notes.NewNotes failed: %v", err)
	}

	chartConst, err := chartconstant.NewChartConstant(14.9)
	if err != nil {
		t.Fatalf("chartconstant.NewChartConstant failed: %v", err)
	}

	chart := &entity.Chart{
		DifficultyID:   5, // ultima
		Const:          chartConst,
		IsConstUnknown: true,
		Notes:          &notesObj,
		NotesDesigner:  stringPtr("譜面作者B"),
	}

	// 変換実行
	dto := ToV1ChartDTO(chart)

	// アサーション
	if dto == nil {
		t.Fatal("ToV1ChartDTO returned nil")
	}

	if dto.Const != chartConst {
		t.Errorf("Const = %v, want %v", dto.Const, chartConst)
	}

	if dto.IsConstUnknown != true {
		t.Errorf("IsConstUnknown = %v, want %v", dto.IsConstUnknown, true)
	}

	if dto.Notes == nil {
		t.Error("Notes is nil")
	} else if *dto.Notes != 999 {
		t.Errorf("Notes = %v, want %v", *dto.Notes, 999)
	}
	if dto.NotesDesigner == nil {
		t.Error("NotesDesigner is nil")
	} else if *dto.NotesDesigner != "譜面作者B" {
		t.Errorf("NotesDesigner = %v, want %v", *dto.NotesDesigner, "譜面作者B")
	}
}

// TestV1SongDTO_JSONMarshal はV1SongDTOのJSONマーシャリングをテストします。
// 全ての難易度キーが含まれ、譜面がない場合はnullになることを確認します。
func TestV1SongDTO_JSONMarshal(t *testing.T) {
	// テストデータの準備
	releaseDate := "2024-01-15"
	jacket := "jacket456"
	bpm := 150
	genre := "VARIETY"
	reading := "ブイワンテスト"

	chartBasic, _ := chartconstant.NewChartConstant(2.0)
	chartExpert, _ := chartconstant.NewChartConstant(10.5)

	v1SongDTO := &V1SongDTO{
		DisplayID: "v1abc123456789ab",
		Title:     "V1テスト楽曲",
		Reading:   &reading,
		Artist:    "V1アーティスト",
		Genre:     &genre,
		BPM:       &bpm,
		Release:   &releaseDate,
		Jacket:    &jacket,
		MaxOP:     85,
		Charts: V1OrderedChartsMap{
			"BASIC":  &V1ChartDTO{Const: chartBasic, IsConstUnknown: false},
			"EXPERT": &V1ChartDTO{Const: chartExpert, IsConstUnknown: false},
		},
	}

	// JSONマーシャル
	jsonBytes, err := json.Marshal(v1SongDTO)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonString := string(jsonBytes)

	if !containsString(jsonString, `"maxop":85`) {
		t.Errorf("JSON should contain maxop field, got: %s", jsonString)
	}

	// is_maxop_unknown がJSONに含まれることを確認
	if !containsString(jsonString, `"is_maxop_unknown":`) {
		t.Errorf("JSON should contain is_maxop_unknown field, got: %s", jsonString)
	}

	// releaseフィールドがreleaseであることを確認（release_dateではない）
	if !containsString(jsonString, `"release":"2024-01-15"`) {
		t.Errorf("JSON should contain 'release' field, got: %s", jsonString)
	}

	if !containsString(jsonString, `"reading":"ブイワンテスト"`) {
		t.Errorf("JSON should contain reading field, got: %s", jsonString)
	}

	// 全ての難易度キーが含まれることを確認
	if !containsString(jsonString, `"BASIC"`) {
		t.Errorf("JSON should contain 'BASIC' key, got: %s", jsonString)
	}
	if !containsString(jsonString, `"ADVANCED"`) {
		t.Errorf("JSON should contain 'ADVANCED' key, got: %s", jsonString)
	}
	if !containsString(jsonString, `"EXPERT"`) {
		t.Errorf("JSON should contain 'EXPERT' key, got: %s", jsonString)
	}
	if !containsString(jsonString, `"MASTER"`) {
		t.Errorf("JSON should contain 'MASTER' key, got: %s", jsonString)
	}
	if !containsString(jsonString, `"ULTIMA"`) {
		t.Errorf("JSON should contain 'ULTIMA' key, got: %s", jsonString)
	}

	// 譜面がない難易度はnullになることを確認
	if !containsString(jsonString, `"ADVANCED":null`) {
		t.Errorf("JSON should contain 'ADVANCED':null, got: %s", jsonString)
	}
	if !containsString(jsonString, `"MASTER":null`) {
		t.Errorf("JSON should contain 'MASTER':null, got: %s", jsonString)
	}
	if !containsString(jsonString, `"ULTIMA":null`) {
		t.Errorf("JSON should contain 'ULTIMA':null, got: %s", jsonString)
	}

	// charts内のキー順序を確認（BASIC→ADVANCED→EXPERT→MASTER→ULTIMA の順）
	basicIdx := indexOfString(jsonString, `"BASIC"`)
	advancedIdx := indexOfString(jsonString, `"ADVANCED"`)
	expertIdx := indexOfString(jsonString, `"EXPERT"`)
	masterIdx := indexOfString(jsonString, `"MASTER"`)
	ultimaIdx := indexOfString(jsonString, `"ULTIMA"`)

	if basicIdx == -1 || advancedIdx == -1 || expertIdx == -1 || masterIdx == -1 || ultimaIdx == -1 {
		t.Fatalf("Missing difficulty keys in JSON: %s", jsonString)
	}

	if !(basicIdx < advancedIdx && advancedIdx < expertIdx && expertIdx < masterIdx && masterIdx < ultimaIdx) {
		t.Errorf("Charts keys are not in correct order (BASIC→ADVANCED→EXPERT→MASTER→ULTIMA), got: %s", jsonString)
	}
}

// containsString はstrがsubstrを含むかどうかを判定します。
func containsString(str, substr string) bool {
	return indexOfString(str, substr) != -1
}

// indexOfString はstrの中でsubstrが最初に現れる位置を返します。見つからない場合は-1を返します。
func indexOfString(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func stringPtr(value string) *string {
	return &value
}
