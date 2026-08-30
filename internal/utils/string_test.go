package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeLike(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{name: "特殊文字を含まない文字列は変化しない", value: "CHUNITHM", expected: "CHUNITHM"},
		{name: "パーセントをエスケープする", value: "100%", expected: `100\%`},
		{name: "アンダースコアをエスケープする", value: "user_name", expected: `user\_name`},
		{name: "バックスラッシュを先にエスケープする", value: `a\%_`, expected: `a\\\%\_`},
		{name: "空文字列は空文字列のまま", value: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			actual := EscapeLike(tt.value)

			// Then
			assert.Equal(t, tt.expected, actual)
		})
	}
}
