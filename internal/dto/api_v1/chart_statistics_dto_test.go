package api_v1

import (
	"testing"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

func TestToChartStatisticsDTO(t *testing.T) {
	now := time.Now()

	t.Run("空のスライスの場合nilを返す", func(t *testing.T) {
		result := ToChartStatisticsDTO([]*entity.ChartStatistics{})
		if result != nil {
			t.Errorf("Expected nil for empty slice, got %v", result)
		}
	})

	t.Run("nilスライスの場合nilを返す", func(t *testing.T) {
		result := ToChartStatisticsDTO(nil)
		if result != nil {
			t.Errorf("Expected nil for nil slice, got %v", result)
		}
	})

	t.Run("単一のレーティング帯", func(t *testing.T) {
		stats := []*entity.ChartStatistics{
			{
				ChartID:     1,
				RatingTier:  150,
				RankS:       10,
				RankSPlus:   25,
				RankSS:      40,
				RankSSPlus:  30,
				RankSSS:     20,
				RankSSSPlus: 5,
				LampAJ:      15,
				LampFC:      45,
				LampOther:   70,
				TotalCount:  130,
				UpdatedAt:   now,
			},
		}

		result := ToChartStatisticsDTO(stats)

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		// 全レーティング帯（28個）が含まれることを確認
		if len(result) != 28 {
			t.Errorf("Expected 28 rating tiers, got %d", len(result))
		}

		rating150, exists := result["15.0"]
		if !exists {
			t.Fatal("Expected rating tier '15.0' to exist")
		}

		// ランク統計の検証
		if rating150.Rank.S != 10 {
			t.Errorf("Expected Rank.S = 10, got %d", rating150.Rank.S)
		}
		if rating150.Rank.SPlus != 25 {
			t.Errorf("Expected Rank.SPlus = 25, got %d", rating150.Rank.SPlus)
		}
		if rating150.Rank.SS != 40 {
			t.Errorf("Expected Rank.SS = 40, got %d", rating150.Rank.SS)
		}
		if rating150.Rank.SSPlus != 30 {
			t.Errorf("Expected Rank.SSPlus = 30, got %d", rating150.Rank.SSPlus)
		}
		if rating150.Rank.SSS != 20 {
			t.Errorf("Expected Rank.SSS = 20, got %d", rating150.Rank.SSS)
		}
		if rating150.Rank.SSSPlus != 5 {
			t.Errorf("Expected Rank.SSSPlus = 5, got %d", rating150.Rank.SSSPlus)
		}

		// ランプ統計の検証
		if rating150.Lamp.AJ != 15 {
			t.Errorf("Expected Lamp.AJ = 15, got %d", rating150.Lamp.AJ)
		}
		if rating150.Lamp.FC != 45 {
			t.Errorf("Expected Lamp.FC = 45, got %d", rating150.Lamp.FC)
		}
		if rating150.Lamp.Other != 70 {
			t.Errorf("Expected Lamp.Other = 70, got %d", rating150.Lamp.Other)
		}
	})

	t.Run("複数のレーティング帯（17.7+を含む）", func(t *testing.T) {
		stats := []*entity.ChartStatistics{
			{
				ChartID:     1,
				RatingTier:  150,
				RankS:       10,
				RankSPlus:   25,
				RankSS:      40,
				RankSSPlus:  30,
				RankSSS:     20,
				RankSSSPlus: 5,
				LampAJ:      15,
				LampFC:      45,
				LampOther:   70,
				TotalCount:  130,
				UpdatedAt:   now,
			},
			{
				ChartID:     1,
				RatingTier:  165,
				RankS:       5,
				RankSPlus:   12,
				RankSS:      20,
				RankSSPlus:  15,
				RankSSS:     10,
				RankSSSPlus: 3,
				LampAJ:      8,
				LampFC:      22,
				LampOther:   35,
				TotalCount:  65,
				UpdatedAt:   now,
			},
			{
				ChartID:     1,
				RatingTier:  177, // 17.7+
				RankS:       1,
				RankSPlus:   3,
				RankSS:      5,
				RankSSPlus:  8,
				RankSSS:     12,
				RankSSSPlus: 20,
				LampAJ:      25,
				LampFC:      15,
				LampOther:   9,
				TotalCount:  49,
				UpdatedAt:   now,
			},
		}

		result := ToChartStatisticsDTO(stats)

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		// 全レーティング帯（28個）が含まれることを確認
		if len(result) != 28 {
			t.Errorf("Expected 28 rating tiers, got %d", len(result))
		}

		// 15.0の検証（データあり）
		if _, exists := result["15.0"]; !exists {
			t.Error("Expected rating tier '15.0' to exist")
		}

		// 16.5の検証（データあり）
		rating165, exists := result["16.5"]
		if !exists {
			t.Fatal("Expected rating tier '16.5' to exist")
		}
		if rating165.Rank.S != 5 {
			t.Errorf("Expected 16.5 Rank.S = 5, got %d", rating165.Rank.S)
		}

		// 17.7+の検証（データあり）
		rating177plus, exists := result["17.7+"]
		if !exists {
			t.Fatal("Expected rating tier '17.7+' to exist")
		}
		if rating177plus.Rank.SSSPlus != 20 {
			t.Errorf("Expected 17.7+ Rank.SSSPlus = 20, got %d", rating177plus.Rank.SSSPlus)
		}
		if rating177plus.Lamp.AJ != 25 {
			t.Errorf("Expected 17.7+ Lamp.AJ = 25, got %d", rating177plus.Lamp.AJ)
		}

		// データがないレーティング帯は0で埋められていることを確認
		rating160, exists := result["16.0"]
		if !exists {
			t.Error("Expected rating tier '16.0' to exist (with zeros)")
		}
		if rating160.Rank.S != 0 || rating160.Lamp.AJ != 0 {
			t.Errorf("Expected all zero values for '16.0', got Rank.S=%d, Lamp.AJ=%d", rating160.Rank.S, rating160.Lamp.AJ)
		}
	})
}
