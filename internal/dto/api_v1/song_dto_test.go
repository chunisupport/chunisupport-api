package api_v1

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

func TestToV1SongDTO(t *testing.T) {
	genreID := 2
	releaseDate := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	song := &entity.Song{
		GenreID:              &genreID,
		ReleasedAt:           &releaseDate,
		OpTargetDifficultyID: 5,
	}

	dto := ToV1SongDTO(song, map[int]string{2: "niconico"}, 90)

	require.NotNil(t, dto)
	require.NotNil(t, dto.Genre)
	assert.Equal(t, "niconico", *dto.Genre)
	require.NotNil(t, dto.Release)
	assert.Equal(t, "2023-12-31", *dto.Release)
	require.NotNil(t, dto.OpTargetDifficulty)
	assert.Equal(t, "ULTIMA", *dto.OpTargetDifficulty)
	assert.NotNil(t, dto.Charts)
}

// TestToV1ChartDTO はToV1ChartDTO関数の基本的な変換をテストします。
func TestToV1ChartDTO(t *testing.T) {
	// テストデータの準備
	notesValue := 999
	notesObj, err := notes.NewNotes(notesValue)
	if err != nil {
		require.Failf(t, "前提条件失敗", "notes.NewNotes failed: %v", err)
	}

	chartConst, err := chartconstant.NewChartConstant(14.9)
	if err != nil {
		require.Failf(t, "前提条件失敗", "chartconstant.NewChartConstant failed: %v", err)
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
		require.Fail(t, "ToV1ChartDTO returned nil")
	}

	if dto.Const != chartConst {
		assert.Failf(t, "アサーション失敗", "Const = %v, want %v", dto.Const, chartConst)
	}

	if dto.IsConstUnknown != true {
		assert.Failf(t, "アサーション失敗", "IsConstUnknown = %v, want %v", dto.IsConstUnknown, true)
	}

	if dto.Notes == nil {
		t.Error("Notes is nil")
	} else if *dto.Notes != 999 {
		assert.Failf(t, "アサーション失敗", "Notes = %v, want %v", *dto.Notes, 999)
	}
	if dto.NotesDesigner == nil {
		t.Error("NotesDesigner is nil")
	} else if *dto.NotesDesigner != "譜面作者B" {
		assert.Failf(t, "アサーション失敗", "NotesDesigner = %v, want %v", *dto.NotesDesigner, "譜面作者B")
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
		DisplayID:          "v1abc123456789ab",
		Title:              "V1テスト楽曲",
		Reading:            &reading,
		Artist:             "V1アーティスト",
		Genre:              &genre,
		BPM:                &bpm,
		Release:            &releaseDate,
		Jacket:             &jacket,
		MaxOP:              85,
		OpTargetDifficulty: stringPtr("EXPERT"),
		IsNew:              true,
		Charts: V1OrderedChartsMap{
			"BASIC":  &V1ChartDTO{Const: chartBasic, IsConstUnknown: false},
			"EXPERT": &V1ChartDTO{Const: chartExpert, IsConstUnknown: false},
		},
	}

	// JSONマーシャル
	jsonBytes, err := json.Marshal(v1SongDTO)
	if err != nil {
		require.Failf(t, "前提条件失敗", "json.Marshal failed: %v", err)
	}

	jsonString := string(jsonBytes)

	if !containsString(jsonString, `"maxop":85`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain maxop field, got: %s", jsonString)
	}

	// is_maxop_unknown がJSONに含まれることを確認
	if !containsString(jsonString, `"is_maxop_unknown":`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain is_maxop_unknown field, got: %s", jsonString)
	}

	if !containsString(jsonString, `"op_target_difficulty":"EXPERT"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain op_target_difficulty field, got: %s", jsonString)
	}

	if !containsString(jsonString, `"is_new":true`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain is_new field, got: %s", jsonString)
	}

	// releaseフィールドがreleaseであることを確認（release_dateではない）
	if !containsString(jsonString, `"release":"2024-01-15"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'release' field, got: %s", jsonString)
	}

	if !containsString(jsonString, `"reading":"ブイワンテスト"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain reading field, got: %s", jsonString)
	}

	// 全ての難易度キーが含まれることを確認
	if !containsString(jsonString, `"BASIC"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'BASIC' key, got: %s", jsonString)
	}
	if !containsString(jsonString, `"ADVANCED"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'ADVANCED' key, got: %s", jsonString)
	}
	if !containsString(jsonString, `"EXPERT"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'EXPERT' key, got: %s", jsonString)
	}
	if !containsString(jsonString, `"MASTER"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'MASTER' key, got: %s", jsonString)
	}
	if !containsString(jsonString, `"ULTIMA"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'ULTIMA' key, got: %s", jsonString)
	}

	// 譜面がない難易度はnullになることを確認
	if !containsString(jsonString, `"ADVANCED":null`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'ADVANCED':null, got: %s", jsonString)
	}
	if !containsString(jsonString, `"MASTER":null`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'MASTER':null, got: %s", jsonString)
	}
	if !containsString(jsonString, `"ULTIMA":null`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'ULTIMA':null, got: %s", jsonString)
	}

	// charts内のキー順序を確認（BASIC→ADVANCED→EXPERT→MASTER→ULTIMA の順）
	chartsJSON := jsonString[indexOfString(jsonString, `"charts":`):]
	basicIdx := indexOfString(chartsJSON, `"BASIC"`)
	advancedIdx := indexOfString(chartsJSON, `"ADVANCED"`)
	expertIdx := indexOfString(chartsJSON, `"EXPERT"`)
	masterIdx := indexOfString(chartsJSON, `"MASTER"`)
	ultimaIdx := indexOfString(chartsJSON, `"ULTIMA"`)

	if basicIdx == -1 || advancedIdx == -1 || expertIdx == -1 || masterIdx == -1 || ultimaIdx == -1 {
		require.Failf(t, "前提条件失敗", "Missing difficulty keys in JSON: %s", jsonString)
	}

	if !(basicIdx < advancedIdx && advancedIdx < expertIdx && expertIdx < masterIdx && masterIdx < ultimaIdx) {
		assert.Failf(t, "アサーション失敗", "Charts keys are not in correct order (BASIC→ADVANCED→EXPERT→MASTER→ULTIMA), got: %s", jsonString)
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
