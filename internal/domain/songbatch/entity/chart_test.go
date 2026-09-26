package entity_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/difficulty"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
	vo "github.com/chunisupport/chunisupport-api/internal/domain/songbatch/valueobject"
)

func TestNewChart(t *testing.T) {
	t.Run("正常系: 新しいChartが生成される", func(t *testing.T) {
		level, _ := vo.ParseLevel("14+")
		chart := entity.NewChart(1, difficulty.Master, level, true)

		if chart.SongID() != 1 {
			t.Errorf("SongID() = %v, want 1", chart.SongID())
		}
		if chart.DifficultyID() != difficulty.Master {
			t.Errorf("DifficultyID() = %v, want %v", chart.DifficultyID(), difficulty.Master)
		}
		if chart.Level() != level {
			t.Errorf("Level() = %v, want %v", chart.Level(), level)
		}
		if !chart.IsConstUnknown() {
			t.Error("IsConstUnknown() = false, want true")
		}
		if chart.Notes() != nil {
			t.Errorf("Notes() = %v, want nil", chart.Notes())
		}
	})
}

func TestChart_SetNotes(t *testing.T) {
	level, _ := vo.ParseLevel("13")
	chart := entity.NewChart(1, difficulty.Expert, level, false)

	chart.SetNotes(1500)

	if chart.Notes() == nil {
		t.Error("Notes() = nil, want 1500")
	} else if *chart.Notes() != 1500 {
		t.Errorf("Notes() = %v, want 1500", *chart.Notes())
	}
}
