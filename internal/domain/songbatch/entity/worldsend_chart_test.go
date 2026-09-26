package entity_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
	vo "github.com/chunisupport/chunisupport-api/internal/domain/songbatch/valueobject"
)

func TestNewWorldsEndChart(t *testing.T) {
	t.Run("正常系: 新しいWorldsEndChartが生成される", func(t *testing.T) {
		weStar, _ := vo.WeStarFromOfficialValue(5)
		weKanji := vo.NewWeKanji("狂")

		chart := entity.NewWorldsEndChart(1, weStar, weKanji)

		if chart.SongID() != 1 {
			t.Errorf("SongID() = %v, want 1", chart.SongID())
		}
		if chart.WeStar() != weStar {
			t.Errorf("WeStar() = %v, want %v", chart.WeStar(), weStar)
		}
		if chart.WeKanji() != weKanji {
			t.Errorf("WeKanji() = %v, want %v", chart.WeKanji(), weKanji)
		}
		if chart.Notes() != nil {
			t.Errorf("Notes() = %v, want nil", chart.Notes())
		}
	})
}

func TestWorldsEndChart_SetNotes(t *testing.T) {
	weStar, _ := vo.WeStarFromOfficialValue(5)
	weKanji := vo.NewWeKanji("狂")

	chart := entity.NewWorldsEndChart(1, weStar, weKanji)

	chart.SetNotes(500)

	if chart.Notes() == nil {
		t.Error("Notes() = nil, want 500")
	} else if *chart.Notes() != 500 {
		t.Errorf("Notes() = %v, want 500", *chart.Notes())
	}
}
