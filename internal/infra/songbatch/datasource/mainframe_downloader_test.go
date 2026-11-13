package datasource

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainframeDownloader_parseSheetData(t *testing.T) {
	// テスト用のダミーデータ（末尾2行は短行のため除外される）
	mockData := &batchGetResponse{
		ValueRanges: []struct {
			Range  string     `json:"range"`
			Values [][]string `json:"values"`
		}{
			{
				Range: "Sheet1",
				Values: [][]string{
					{"曲名1", "BAS", "POPS", "", "10.5"},
					{"曲名2", "ADV", "niconico", "", "12.3"},
					{"曲名3", "EXP", "ORIGINAL", "", "13.7"},
					// 短行のため除外：先頭列が難易度略称でタイトル列が存在しない
					{"BAS", "POPS", "", "10.5"},
				},
			},
			{
				Range: "Sheet2",
				Values: [][]string{
					{"曲名4", "MAS", "東方Project", "", "14.2"},
					{"曲名5", "ULT", "VARIETY", "", "15.0"},
					// 短行のため除外：ジャンル列はあるが定数列が存在しない
					{"短行", "ULT", "VARIETY"},
				},
			},
		},
	}

	d := NewMainframeDownloader("test_output", "dummy_key", "dummy_id", "https://sheets.googleapis.com/v4/spreadsheets")
	charts := d.parseSheetData(mockData)

	// 結果全体を固定し、Diff/Genreの破損も検知できるようにする
	expected := []MainframeChartData{
		{Title: "曲名1", Diff: "BAS", Genre: "POPS", Const: 10.5},
		{Title: "曲名2", Diff: "ADV", Genre: "niconico", Const: 12.3},
		{Title: "曲名3", Diff: "EXP", Genre: "ORIGINAL", Const: 13.7},
		{Title: "曲名4", Diff: "MAS", Genre: "東方Project", Const: 14.2},
		{Title: "曲名5", Diff: "ULT", Genre: "VARIETY", Const: 15.0},
	}
	assert.ElementsMatch(t, expected, charts)

	// 重複排除のテスト
	mockDataWithDuplicates := &batchGetResponse{
		ValueRanges: []struct {
			Range  string     `json:"range"`
			Values [][]string `json:"values"`
		}{
			{
				Range: "Sheet1",
				Values: [][]string{
					{"曲名1", "BAS", "POPS", "", "10.5"},
					{"曲名1", "BAS", "POPS", "", "10.5"}, // 重複
				},
			},
		},
	}

	chartsWithDuplicates := d.parseSheetData(mockDataWithDuplicates)
	require.Len(t, chartsWithDuplicates, 1)
	assert.Equal(t, []MainframeChartData{
		{Title: "曲名1", Diff: "BAS", Genre: "POPS", Const: 10.5},
	}, chartsWithDuplicates)
}

func TestMainframeDownloader_parseSheetData_skipsUnfilledConstants(t *testing.T) {
	d := NewMainframeDownloader("test_output", "dummy_key", "dummy_id", "https://sheets.googleapis.com/v4/spreadsheets")
	data := &batchGetResponse{
		ValueRanges: []struct {
			Range  string     `json:"range"`
			Values [][]string `json:"values"`
		}{
			{
				Range: "Sheet1",
				Values: [][]string{
					{"定数あり", "MAS", "ORIGINAL", "", "14.5"},
					{"未入力", "EXP", "POPS", "", ""},
					{"空白のみ", "ADV", "VARIETY", "", "   "},
					{"空白付き数値", "BAS", "niconico", "", " 12.0 "},
					{"不正値", "ULT", "東方Project", "", "未定"},
				},
			},
		},
	}

	charts := d.parseSheetData(data)

	require.Len(t, charts, 2)
	// 結果全体を固定し、Diff/Genreの破損も検知できるようにする
	expected := []MainframeChartData{
		{Title: "定数あり", Diff: "MAS", Genre: "ORIGINAL", Const: 14.5},
		{Title: "空白付き数値", Diff: "BAS", Genre: "niconico", Const: 12.0},
	}
	assert.ElementsMatch(t, expected, charts)
}

func TestMainframeDownloader_isDifficultyShort(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"BAS", true},
		{"ADV", true},
		{"EXP", true},
		{"MAS", true},
		{"ULT", true},
		{"BASIC", false},
		{"ADVANCED", false},
		{"OTHER", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := isDifficultyShort(tc.input)
		assert.Equal(t, tc.expected, result, "isDifficultyShort(%q)", tc.input)
	}
}

func TestMainframeDownloader_Download_Integration(t *testing.T) {
	// 統合テストは環境変数が必要なためスキップ可能に
	apiKey := os.Getenv("CHUNISUPPORT_BATCH_GOOGLE_CLOUD_API_KEY")
	sheetID := os.Getenv("CHUNISUPPORT_BATCH_GOOGLE_SHEET_ID")

	if apiKey == "" || sheetID == "" {
		t.Skip("Skipping integration test: CHUNISUPPORT_BATCH_GOOGLE_CLOUD_API_KEY or CHUNISUPPORT_BATCH_GOOGLE_SHEET_ID not set")
	}

	// 一時ディレクトリを作成
	tempDir := t.TempDir()

	downloader := NewMainframeDownloader(tempDir, apiKey, sheetID, "https://sheets.googleapis.com/v4/spreadsheets")

	err := downloader.Download(context.Background())
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// ファイルが作成されたか確認
	filePath := filepath.Join(tempDir, "mainframe.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist", filePath)
	}

	// JSONが正しくパースできるか確認
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var charts []MainframeChartData
	if err := json.Unmarshal(data, &charts); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(charts) == 0 {
		t.Error("Expected at least one chart in the output")
	}

	t.Logf("Successfully downloaded and parsed %d charts", len(charts))
}
