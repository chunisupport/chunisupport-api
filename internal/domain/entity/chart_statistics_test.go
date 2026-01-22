package entity

import (
	"testing"
)

func TestChartStatistics_IsValidRatingTier(t *testing.T) {
	tests := []struct {
		name       string
		ratingTier int
		want       bool
	}{
		{"最小値 15.0", 150, true},
		{"中間値 16.5", 165, true},
		{"最大値（通常）17.6", 176, true},
		{"特殊値 17.7+", 177, true},
		{"範囲外（小）", 149, false},
		{"範囲外（大）", 178, false},
		{"範囲外（負）", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChartStatistics{RatingTier: tt.ratingTier}
			if got := cs.IsValidRatingTier(); got != tt.want {
				t.Errorf("IsValidRatingTier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChartStatistics_GetRatingTierString(t *testing.T) {
	tests := []struct {
		name       string
		ratingTier int
		want       string
	}{
		{"15.0", 150, "15.0"},
		{"15.1", 151, "15.1"},
		{"15.9", 159, "15.9"},
		{"16.0", 160, "16.0"},
		{"17.6", 176, "17.6"},
		{"17.7+（特殊）", 177, "17.7+"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChartStatistics{RatingTier: tt.ratingTier}
			if got := cs.GetRatingTierString(); got != tt.want {
				t.Errorf("GetRatingTierString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatRatingTier(t *testing.T) {
	tests := []struct {
		tier int
		want string
	}{
		{150, "15.0"},
		{151, "15.1"},
		{152, "15.2"},
		{159, "15.9"},
		{160, "16.0"},
		{165, "16.5"},
		{170, "17.0"},
		{176, "17.6"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatRatingTier(tt.tier); got != tt.want {
				t.Errorf("formatRatingTier(%d) = %v, want %v", tt.tier, got, tt.want)
			}
		})
	}
}
