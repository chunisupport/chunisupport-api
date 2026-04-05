package handler

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

func TestBuildChartsMap_マスタIDが非連番でも通常難易度キーを初期化する(t *testing.T) {
	difficultyNamesByID := map[int]string{
		10: "BASIC",
		20: "ADVANCED",
		30: "EXPERT",
		40: "MASTER",
		50: "ULTIMA",
		60: "WORLD'S END",
	}

	charts := []*entity.Chart{
		{DifficultyID: 10},
		{DifficultyID: 50},
	}

	chartMap := BuildChartsMap(charts, difficultyNamesByID, func(chart *entity.Chart) *entity.Chart {
		return chart
	})

	expectedDiffs := []string{"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"}
	for _, diff := range expectedDiffs {
		if _, ok := chartMap[diff]; !ok {
			t.Fatalf("%s が初期化されていません", diff)
		}
	}

	if _, exists := chartMap["WORLD'S END"]; exists {
		t.Fatalf("WORLD'S END は通常譜面マップに含まれてはいけません")
	}

	if chartMap["ADVANCED"] != nil {
		t.Fatalf("ADVANCED は譜面未登録のため nil を期待します")
	}
	if chartMap["EXPERT"] != nil {
		t.Fatalf("EXPERT は譜面未登録のため nil を期待します")
	}
	if chartMap["MASTER"] != nil {
		t.Fatalf("MASTER は譜面未登録のため nil を期待します")
	}
}
