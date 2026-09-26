package models_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/difficulty"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
	vo "github.com/chunisupport/chunisupport-api/internal/domain/songbatch/valueobject"
	"github.com/chunisupport/chunisupport-api/internal/infra/songbatch/models"
)

func TestFromChartEntityForUpsert(t *testing.T) {
	t.Run("正常系: UPSERT用モデルに変換", func(t *testing.T) {
		level, _ := vo.ParseLevel("13")
		chart := entity.NewChart(100, difficulty.Expert, level, false)

		model := models.FromChartEntityForUpsert(chart)

		if model.SongID != 100 {
			t.Errorf("SongID = %v, want 100", model.SongID)
		}
		if model.DifficultyID != difficulty.Expert.Int() {
			t.Errorf("DifficultyID = %v, want %v", model.DifficultyID, difficulty.Expert.Int())
		}
		if model.Const != 13.0 {
			t.Errorf("Const = %v, want 13.0", model.Const)
		}
		if model.IsConstUnknown != 0 {
			t.Errorf("IsConstUnknown = %v, want 0", model.IsConstUnknown)
		}
	})
}
