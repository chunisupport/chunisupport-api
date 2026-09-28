package ratingband

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRatingBand_Contains(t *testing.T) {
	lower, upper := 17.0, 17.1
	tests := []struct {
		name     string
		band     RatingBand
		rating   float64
		expected bool
	}{
		{name: "下限と上限がない帯はすべてのレーティングを含む", band: RatingBand{}, rating: 0, expected: true},
		{name: "下限と同じ値は含む", band: RatingBand{MinInclusive: &lower, MaxExclusive: &upper}, rating: 17.0, expected: true},
		{name: "上限と同じ値は含まない", band: RatingBand{MinInclusive: &lower, MaxExclusive: &upper}, rating: 17.1, expected: false},
		{name: "下限未満は含まない", band: RatingBand{MinInclusive: &lower, MaxExclusive: &upper}, rating: 16.99, expected: false},
		{name: "上限だけの帯は上限未満を含む", band: RatingBand{MaxExclusive: &upper}, rating: 1.0, expected: true},
		{name: "下限だけの帯は下限以上を含む", band: RatingBand{MinInclusive: &lower}, rating: 18.0, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			result := tt.band.Contains(tt.rating)

			// Then
			assert.Equal(t, tt.expected, result)
		})
	}
}
