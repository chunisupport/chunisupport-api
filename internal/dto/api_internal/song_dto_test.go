package api_internal

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// TestToSongDTO はToSongDTO関数の基本的な変換をテストします。
func TestToSongDTO(t *testing.T) {
	// テストデータの準備
	genreID := 1
	bpm := 180
	imgURL := "https://example.com/jacket.jpg"
	releaseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	masterConst, _ := chartconstant.NewChartConstant(13.0)
	ultimaConst, _ := chartconstant.NewChartConstant(15.0)

	song := &entity.Song{
		DisplayID:      "test123456789012",
		Title:          "テスト楽曲",
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
	dto := ToSongDTO(song, genreNamesByID, 90)

	// アサーション
	if dto == nil {
		t.Fatal("ToSongDTO returned nil")
	}

	if dto.DisplayID != "test123456789012" {
		t.Errorf("DisplayID = %v, want %v", dto.DisplayID, "test123456789012")
	}

	if dto.Title != "テスト楽曲" {
		t.Errorf("Title = %v, want %v", dto.Title, "テスト楽曲")
	}

	if dto.Artist != "テストアーティスト" {
		t.Errorf("Artist = %v, want %v", dto.Artist, "テストアーティスト")
	}

	// Genre は *string なので null チェック
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

	// Jacket (旧 Img)
	if dto.Jacket == nil {
		t.Error("Jacket is nil")
	} else if *dto.Jacket != "https://example.com/jacket.jpg" {
		t.Errorf("Jacket = %v, want %v", *dto.Jacket, "https://example.com/jacket.jpg")
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

// TestToChartDTO はToChartDTO関数の基本的な変換をテストします。
func TestToChartDTO(t *testing.T) {
	// テストデータの準備
	notesValue := 1234
	notesObj, err := notes.NewNotes(notesValue)
	if err != nil {
		t.Fatalf("notes.NewNotes failed: %v", err)
	}

	chartConst, err := chartconstant.NewChartConstant(13.4)
	if err != nil {
		t.Fatalf("chartconstant.NewChartConstant failed: %v", err)
	}

	chart := &entity.Chart{
		DifficultyID:   3, // expert
		Const:          chartConst,
		IsConstUnknown: false,
		Notes:          &notesObj,
		NotesDesigner:  stringPtr("譜面作者A"),
	}

	// 変換実行
	dto := ToChartDTO(chart)

	// アサーション
	if dto == nil {
		t.Fatal("ToChartDTO returned nil")
	}

	if dto.Const != chartConst {
		t.Errorf("Const = %v, want %v", dto.Const, chartConst)
	}

	if dto.IsConstUnknown != false {
		t.Errorf("IsConstUnknown = %v, want %v", dto.IsConstUnknown, false)
	}

	if dto.Notes == nil {
		t.Error("Notes is nil")
	} else if *dto.Notes != 1234 {
		t.Errorf("Notes = %v, want %v", *dto.Notes, 1234)
	}
	if dto.NotesDesigner == nil {
		t.Error("NotesDesigner is nil")
	} else if *dto.NotesDesigner != "譜面作者A" {
		t.Errorf("NotesDesigner = %v, want %v", *dto.NotesDesigner, "譜面作者A")
	}
}

// TestSongDTO_JSONMarshal はSongDTOのJSONマーシャリングをテストします。
// charts内のキー順序がBASIC→ADVANCED→EXPERT→MASTER→ULTIMAであること、
// constが小数点以下1桁表記であることを確認します。
func TestSongDTO_JSONMarshal(t *testing.T) {
	// テストデータの準備
	releaseDate := "2024-01-15"
	jacket := "jacket123"
	bpm := 180
	genre := "ORIGINAL"

	chartBasic, _ := chartconstant.NewChartConstant(3.0)
	chartAdvanced, _ := chartconstant.NewChartConstant(5.0)
	chartExpert, _ := chartconstant.NewChartConstant(11.3)
	chartMaster, _ := chartconstant.NewChartConstant(14.0)

	songDTO := &SongDTO{
		DisplayID: "92eaa42ee1d1a70f",
		Title:     "テスト楽曲",
		Artist:    "テストアーティスト",
		Genre:     &genre,
		BPM:       &bpm,
		Release:   &releaseDate,
		Jacket:    &jacket,
		MaxOP:     85,
		Charts: OrderedChartsMap{
			"BASIC":    &ChartDTO{Const: chartBasic, IsConstUnknown: false, Notes: nil},
			"ADVANCED": &ChartDTO{Const: chartAdvanced, IsConstUnknown: false, Notes: nil},
			"EXPERT":   &ChartDTO{Const: chartExpert, IsConstUnknown: false, Notes: nil},
			"MASTER":   &ChartDTO{Const: chartMaster, IsConstUnknown: false, Notes: nil},
		},
	}

	// JSONマーシャル
	jsonBytes, err := json.Marshal(songDTO)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonString := string(jsonBytes)

	if !strings.Contains(jsonString, `"maxop":85`) {
		t.Errorf("JSON should contain maxop field, got: %s", jsonString)
	}

	// is_maxop_unknown がJSONに含まれることを確認
	if !strings.Contains(jsonString, `"is_maxop_unknown":`) {
		t.Errorf("JSON should contain is_maxop_unknown field, got: %s", jsonString)
	}

	// releaseフィールドがreleaseであることを確認（release_dateではない）
	if !strings.Contains(jsonString, `"release":"2024-01-15"`) {
		t.Errorf("JSON should contain 'release' field, got: %s", jsonString)
	}

	// constが小数点以下1桁表記であることを確認
	if !strings.Contains(jsonString, `"const":3.0`) && !strings.Contains(jsonString, `"const":3`) {
		t.Errorf("JSON should contain const:3.0, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"const":11.3`) {
		t.Errorf("JSON should contain const:11.3, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"const":14.0`) && !strings.Contains(jsonString, `"const":14`) {
		t.Errorf("JSON should contain const:14.0, got: %s", jsonString)
	}

	// charts内のキー順序を確認（BASIC→ADVANCED→EXPERT→MASTER の順）
	basicIdx := strings.Index(jsonString, `"BASIC"`)
	advancedIdx := strings.Index(jsonString, `"ADVANCED"`)
	expertIdx := strings.Index(jsonString, `"EXPERT"`)
	masterIdx := strings.Index(jsonString, `"MASTER"`)

	if basicIdx == -1 || advancedIdx == -1 || expertIdx == -1 || masterIdx == -1 {
		t.Fatalf("Missing difficulty keys in JSON: %s", jsonString)
	}

	if !(basicIdx < advancedIdx && advancedIdx < expertIdx && expertIdx < masterIdx) {
		t.Errorf("Charts keys are not in correct order (BASIC→ADVANCED→EXPERT→MASTER), got: %s", jsonString)
	}
}

// TestOrderedChartsMap_MarshalJSON はOrderedChartsMapのJSONマーシャリングをテストします。
// 全ての難易度キーが含まれ、譜面がない場合はnullになることを確認します。
func TestOrderedChartsMap_MarshalJSON(t *testing.T) {
	chartBasic, _ := chartconstant.NewChartConstant(3.0)
	chartMaster, _ := chartconstant.NewChartConstant(14.0)

	// 意図的に順不同で追加（BASICとMASTERのみ）
	chartsMap := OrderedChartsMap{
		"MASTER": &ChartDTO{Const: chartMaster, IsConstUnknown: false},
		"BASIC":  &ChartDTO{Const: chartBasic, IsConstUnknown: false},
	}

	jsonBytes, err := json.Marshal(chartsMap)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonString := string(jsonBytes)

	// 全ての難易度キーが含まれることを確認
	if !strings.Contains(jsonString, `"BASIC"`) {
		t.Errorf("JSON should contain 'BASIC' key, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"ADVANCED"`) {
		t.Errorf("JSON should contain 'ADVANCED' key, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"EXPERT"`) {
		t.Errorf("JSON should contain 'EXPERT' key, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"MASTER"`) {
		t.Errorf("JSON should contain 'MASTER' key, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"ULTIMA"`) {
		t.Errorf("JSON should contain 'ULTIMA' key, got: %s", jsonString)
	}

	// 譜面がない難易度はnullになることを確認
	if !strings.Contains(jsonString, `"ADVANCED":null`) {
		t.Errorf("JSON should contain 'ADVANCED':null, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"EXPERT":null`) {
		t.Errorf("JSON should contain 'EXPERT':null, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"ULTIMA":null`) {
		t.Errorf("JSON should contain 'ULTIMA':null, got: %s", jsonString)
	}

	// BASIC→ADVANCED→EXPERT→MASTER→ULTIMAの順で出力されることを確認
	basicIdx := strings.Index(jsonString, `"BASIC"`)
	advancedIdx := strings.Index(jsonString, `"ADVANCED"`)
	expertIdx := strings.Index(jsonString, `"EXPERT"`)
	masterIdx := strings.Index(jsonString, `"MASTER"`)
	ultimaIdx := strings.Index(jsonString, `"ULTIMA"`)

	if basicIdx == -1 || advancedIdx == -1 || expertIdx == -1 || masterIdx == -1 || ultimaIdx == -1 {
		t.Fatalf("Missing difficulty keys in JSON: %s", jsonString)
	}

	if !(basicIdx < advancedIdx && advancedIdx < expertIdx && expertIdx < masterIdx && masterIdx < ultimaIdx) {
		t.Errorf("Charts keys are not in correct order (BASIC→ADVANCED→EXPERT→MASTER→ULTIMA), got: %s", jsonString)
	}
}

func stringPtr(value string) *string {
	return &value
}
