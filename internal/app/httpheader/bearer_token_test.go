package httpheader

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "Bearerトークンを抽出できる",
			header:   "Bearer firebase-token",
			expected: "firebase-token",
		},
		{
			name:     "schemeの大文字小文字は区別しない",
			header:   "bearer firebase-token",
			expected: "firebase-token",
		},
		{
			name:     "トークン前後の空白を除去する",
			header:   "Bearer   firebase-token   ",
			expected: "firebase-token",
		},
		{
			name:     "Authorizationヘッダがない場合は空文字を返す",
			header:   "",
			expected: "",
		},
		{
			name:     "Bearer以外のschemeは拒否する",
			header:   "Basic firebase-token",
			expected: "",
		},
		{
			name:     "トークンが空なら空文字を返す",
			header:   "Bearer    ",
			expected: "",
		},
		{
			name:     "空白なしのBearer単体は空文字を返す",
			header:   "Bearer",
			expected: "",
		},
		{
			name:     "空白なしのBearertokenは空文字を返す",
			header:   "Bearertoken",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			header := make(http.Header)
			if tt.header != "" {
				header.Set(authorizationHeader, tt.header)
			}

			// When
			got := ExtractBearerToken(header)

			// Then
			assert.Equal(t, tt.expected, got)
		})
	}
}
