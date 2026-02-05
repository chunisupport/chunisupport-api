package handler

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

const (
	MinDifficultyID = 1
	MaxDifficultyID = 5
)

// BuildChartsMap creates a map of charts keyed by difficulty name.
// T is the type of the Chart DTO (e.g., *dto.ChartDTO or *dto.V1ChartDTO).
func BuildChartsMap[T any](
	charts []*entity.Chart,
	difficultyNames map[int]string,
	converter func(*entity.Chart) T,
) map[string]T {
	// Initialize map with nil for all difficulty levels
	chartsMap := make(map[string]T)
	for diffID, diffName := range difficultyNames {
		if diffID >= MinDifficultyID && diffID <= MaxDifficultyID {
			var zero T
			chartsMap[diffName] = zero
		}
	}

	// Populate map with actual chart data
	for _, chart := range charts {
		if diffName, ok := difficultyNames[chart.DifficultyID]; ok {
			chartsMap[diffName] = converter(chart)
		}
	}

	return chartsMap
}
