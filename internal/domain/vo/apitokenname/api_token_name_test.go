package apitokenname

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPITokenName(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{name: "前後の空白を除去する", value: "  Discord Bot  ", expected: "Discord Bot"},
		{name: "日本語名を受け付ける", value: "スコア取得ツール", expected: "スコア取得ツール"},
		{name: "50文字を受け付ける", value: strings.Repeat("a", 50), expected: strings.Repeat("a", 50)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := NewAPITokenName(tt.value)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, name.String())
		})
	}
}

func TestNewAPITokenName_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "空文字", value: ""},
		{name: "空白のみ", value: " \t "},
		{name: "51文字", value: strings.Repeat("a", 51)},
		{name: "制御文字を含む", value: "bot\nname"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAPITokenName(tt.value)

			assert.ErrorIs(t, err, ErrInvalidAPITokenName)
		})
	}
}
