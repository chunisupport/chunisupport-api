package api_internal

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSongDTO(t *testing.T) {
	genreID := 1
	releaseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	song := &entity.Song{
		GenreID:              &genreID,
		ReleasedAt:           &releaseDate,
		NameFolderCode:       "ABCD",
		OpTargetDifficultyID: 5,
		UnlockRequired:       true,
	}

	dto := ToSongDTO(song, map[int]string{1: "POPS & ANIME"}, 90)

	require.NotNil(t, dto)
	require.NotNil(t, dto.Genre)
	assert.Equal(t, "POPS & ANIME", *dto.Genre)
	require.NotNil(t, dto.Release)
	assert.Equal(t, "2024-01-15", *dto.Release)
	require.NotNil(t, dto.OpTargetDifficulty)
	assert.Equal(t, "ULTIMA", *dto.OpTargetDifficulty)
	assert.True(t, dto.UnlockRequired)
	assert.Equal(t, "ABCD", dto.NameFolderCode)
	assert.NotNil(t, dto.Charts)
}

// TestToChartDTO はToChartDTO関数の基本的な変換をテストします。
func TestToChartDTO(t *testing.T) {
	notesValue := 1234
	notesObj, err := notes.NewNotes(notesValue)
	if err != nil {
		require.Failf(t, "前提条件失敗", "notes.NewNotes failed: %v", err)
	}

	chartConst, err := chartconstant.NewChartConstant(13.4)
	if err != nil {
		require.Failf(t, "前提条件失敗", "chartconstant.NewChartConstant failed: %v", err)
	}

	chart := &entity.Chart{
		DifficultyID:   3,
		Const:          chartConst,
		IsConstUnknown: false,
		Notes:          &notesObj,
		NotesDesigner:  new("譜面作者A"),
	}

	dto := ToChartDTO(chart)

	if dto == nil {
		require.Fail(t, "ToChartDTO returned nil")
	}

	if dto.Const != chartConst {
		assert.Failf(t, "アサーション失敗", "Const = %v, want %v", dto.Const, chartConst)
	}

	if dto.IsConstUnknown != false {
		assert.Failf(t, "アサーション失敗", "IsConstUnknown = %v, want %v", dto.IsConstUnknown, false)
	}

	require.NotNil(t, dto.Notes)
	assert.Equal(t, 1234, *dto.Notes)
	require.NotNil(t, dto.NotesDesigner)
	assert.Equal(t, "譜面作者A", *dto.NotesDesigner)
}

// TestSongDTO_JSONMarshal はSongDTOのJSONマーシャリングをテストします。
// charts内のキー順序がBASIC→ADVANCED→EXPERT→MASTER→ULTIMAであること、
// constが小数点以下1桁表記であることを確認します。
func TestSongDTO_JSONMarshal(t *testing.T) {
	releaseDate := "2024-01-15"
	jacket := "jacket123"
	bpm := 180
	genre := "ORIGINAL"
	reading := "テストガッキョク"

	chartBasic, _ := chartconstant.NewChartConstant(3.0)
	chartAdvanced, _ := chartconstant.NewChartConstant(5.0)
	chartExpert, _ := chartconstant.NewChartConstant(11.3)
	chartMaster, _ := chartconstant.NewChartConstant(14.0)

	songDTO := &SongDTO{
		DisplayID:          "92eaa42ee1d1a70f",
		Title:              "テスト楽曲",
		Reading:            &reading,
		NameFolderCode:     "ABCD",
		Artist:             "テストアーティスト",
		Genre:              &genre,
		BPM:                &bpm,
		Release:            &releaseDate,
		Jacket:             &jacket,
		MaxOP:              85,
		OpTargetDifficulty: new("MASTER"),
		IsNew:              true,
		Charts: OrderedChartsMap{
			"BASIC":    &ChartDTO{Const: chartBasic, IsConstUnknown: false, Notes: nil},
			"ADVANCED": &ChartDTO{Const: chartAdvanced, IsConstUnknown: false, Notes: nil},
			"EXPERT":   &ChartDTO{Const: chartExpert, IsConstUnknown: false, Notes: nil},
			"MASTER":   &ChartDTO{Const: chartMaster, IsConstUnknown: false, Notes: nil},
		},
	}

	jsonBytes, err := json.Marshal(songDTO)
	if err != nil {
		require.Failf(t, "前提条件失敗", "json.Marshal failed: %v", err)
	}

	jsonString := string(jsonBytes)

	if !strings.Contains(jsonString, `"maxop":85`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain maxop field, got: %s", jsonString)
	}

	// is_maxop_unknown がJSONに含まれることを確認
	if !strings.Contains(jsonString, `"is_maxop_unknown":`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain is_maxop_unknown field, got: %s", jsonString)
	}

	if !strings.Contains(jsonString, `"op_target_difficulty":"MASTER"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain op_target_difficulty field, got: %s", jsonString)
	}

	if !strings.Contains(jsonString, `"is_new":true`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain is_new field, got: %s", jsonString)
	}

	// releaseフィールドがreleaseであることを確認（release_dateではない）
	if !strings.Contains(jsonString, `"release":"2024-01-15"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain 'release' field, got: %s", jsonString)
	}

	if !strings.Contains(jsonString, `"reading":"テストガッキョク"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain reading field, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"name_folder_code":"ABCD"`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain name_folder_code field, got: %s", jsonString)
	}

	// constが小数点以下1桁表記であることを確認
	if !strings.Contains(jsonString, `"const":3.0`) && !strings.Contains(jsonString, `"const":3`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain const:3.0, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"const":11.3`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain const:11.3, got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"const":14.0`) && !strings.Contains(jsonString, `"const":14`) {
		assert.Failf(t, "アサーション失敗", "JSON should contain const:14.0, got: %s", jsonString)
	}

	// charts内のキー順序を確認（BASIC→ADVANCED→EXPERT→MASTER の順）
	chartsJSON := jsonString[strings.Index(jsonString, `"charts":`):]
	basicIdx := strings.Index(chartsJSON, `"BASIC"`)
	advancedIdx := strings.Index(chartsJSON, `"ADVANCED"`)
	expertIdx := strings.Index(chartsJSON, `"EXPERT"`)
	masterIdx := strings.Index(chartsJSON, `"MASTER"`)

	if basicIdx == -1 || advancedIdx == -1 || expertIdx == -1 || masterIdx == -1 {
		require.Failf(t, "前提条件失敗", "Missing difficulty keys in JSON: %s", jsonString)
	}

	if !(basicIdx < advancedIdx && advancedIdx < expertIdx && expertIdx < masterIdx) {
		assert.Failf(t, "アサーション失敗", "Charts keys are not in correct order (BASIC→ADVANCED→EXPERT→MASTER), got: %s", jsonString)
	}
}

// TestOrderedChartsMap_MarshalJSON_現行のバイト列と一致する は、
// 共通化後もキー順・null・エスケープが共通化前の出力とバイト単位で一致することを確認します。
func TestOrderedChartsMap_MarshalJSON_現行のバイト列と一致する(t *testing.T) {
	chartBasic, err := chartconstant.NewChartConstant(3.0)
	require.NoError(t, err)
	chartMaster, err := chartconstant.NewChartConstant(14.9)
	require.NoError(t, err)
	notes := 1234
	designer := "A \"B\" <C> & D\\譜面\n\u2028"
	basic := &ChartDTO{
		Const:          chartBasic,
		IsConstUnknown: false,
		Notes:          &notes,
		NotesDesigner:  &designer,
	}

	tests := []struct {
		name string
		// Given: 順不同の譜面マップ
		charts OrderedChartsMap
		// Then: 共通化前のJSON
		want string
	}{
		{
			name: "存在する譜面と欠損とnilと未知キー",
			charts: OrderedChartsMap{
				"MASTER":  &ChartDTO{Const: chartMaster, IsConstUnknown: true},
				"BASIC":   basic,
				"UNKNOWN": basic,
				"EXPERT":  nil,
			},
			want: `{"BASIC":{"const":3,"is_const_unknown":false,"notes":1234,"notes_designer":"A \"B\" \u003cC\u003e \u0026 D\\譜面\n\u2028"},"ADVANCED":null,"EXPERT":null,"MASTER":{"const":14.9,"is_const_unknown":true,"notes":null,"notes_designer":null},"ULTIMA":null}`,
		},
		{
			name:   "nilマップは全難易度がnull",
			charts: nil,
			want:   `{"BASIC":null,"ADVANCED":null,"EXPERT":null,"MASTER":null,"ULTIMA":null}`,
		},
		{
			name:   "空マップは全難易度がnull",
			charts: OrderedChartsMap{},
			want:   `{"BASIC":null,"ADVANCED":null,"EXPERT":null,"MASTER":null,"ULTIMA":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := json.Marshal(tt.charts)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

// TestEditorOrderedChartsMap_MarshalJSON_現行のバイト列と一致する は、
// 編集者向け譜面の updated_at を含めて共通化前の出力と一致することを確認します。
func TestEditorOrderedChartsMap_MarshalJSON_現行のバイト列と一致する(t *testing.T) {
	chartUltima, err := chartconstant.NewChartConstant(15.4)
	require.NoError(t, err)
	notes := 999
	designer := "編集者\"<>&\\"
	updatedAt := time.Date(2026, 7, 11, 15, 30, 0, 123000000, time.UTC)
	ultima := &EditorChartDTO{
		Const:          chartUltima,
		IsConstUnknown: false,
		Notes:          &notes,
		NotesDesigner:  &designer,
		UpdatedAt:      &updatedAt,
	}

	tests := []struct {
		name string
		// Given: 順不同の編集者向け譜面マップ
		charts EditorOrderedChartsMap
		// Then: 共通化前のJSON
		want string
	}{
		{
			name: "ULTIMAと未知キー",
			charts: EditorOrderedChartsMap{
				"ULTIMA": ultima,
				"WEIRD":  &EditorChartDTO{Const: chartUltima},
			},
			want: `{"BASIC":null,"ADVANCED":null,"EXPERT":null,"MASTER":null,"ULTIMA":{"const":15.4,"is_const_unknown":false,"notes":999,"notes_designer":"編集者\"\u003c\u003e\u0026\\","updated_at":"2026-07-11T15:30:00.123Z"}}`,
		},
		{
			name:   "nilマップは全難易度がnull",
			charts: nil,
			want:   `{"BASIC":null,"ADVANCED":null,"EXPERT":null,"MASTER":null,"ULTIMA":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := json.Marshal(tt.charts)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestToSongDTO_WikiPageTitle(t *testing.T) {
	tests := []struct {
		name string
		// Given: 楽曲のWikiページタイトル
		wikiPageTitle *string
		// Then: JSONに出力される wiki_page_title
		expectedJSON string
	}{
		{
			name:          "Wikiページタイトルが設定されている場合はその値を返す",
			wikiPageTitle: new("楽曲名(CHUNITHM)"),
			expectedJSON:  `"wiki_page_title":"楽曲名(CHUNITHM)"`,
		},
		{
			name:          "Wikiページタイトルが未設定の場合はnullを返す",
			wikiPageTitle: nil,
			expectedJSON:  `"wiki_page_title":null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			song := &entity.Song{WikiPageTitle: tt.wikiPageTitle}

			// When
			dto := ToSongDTO(song, map[int]string{}, 0)
			jsonBytes, err := json.Marshal(dto)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.wikiPageTitle, dto.WikiPageTitle)
			assert.Contains(t, string(jsonBytes), tt.expectedJSON)
		})
	}
}
