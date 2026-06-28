package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupportsScoreHistory(t *testing.T) {
	tests := []struct {
		name       string
		difficulty string
		expected   bool
	}{
		{name: "EXPERTは対象", difficulty: "EXPERT", expected: true},
		{name: "MASTERは対象", difficulty: "MASTER", expected: true},
		{name: "ULTIMAは対象", difficulty: "ULTIMA", expected: true},
		{name: "BASICは対象外", difficulty: "BASIC", expected: false},
		{name: "ADVANCEDは対象外", difficulty: "ADVANCED", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, SupportsScoreHistory(tt.difficulty))
		})
	}
}
