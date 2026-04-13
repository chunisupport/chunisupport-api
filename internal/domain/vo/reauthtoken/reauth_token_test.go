package reauthtoken

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectedErr error
	}{
		{
			name:     "前後の空白を除去したトークンを生成できる",
			input:    "  reauth-token  ",
			expected: "reauth-token",
		},
		{
			name:        "空白のみはエラーになる",
			input:       "   ",
			expectedErr: ErrEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given

			// When
			token, err := New(tt.input)

			// Then
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				assert.Equal(t, "", token.String())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, token.String())
		})
	}
}
