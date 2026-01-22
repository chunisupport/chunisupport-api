package chunirec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateLevel(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{12.0, 12.0},
		{12.4, 12.0},
		{12.5, 12.5},
		{12.9, 12.5},
		{13.0, 13.0},
		{14.8, 14.5},
	}

	for _, test := range tests {
		result := calculateLevel(test.input)
		assert.Equal(t, test.expected, result, "Input: %f", test.input)
	}
}
