package httpheader

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		expected   string
		headerName string
	}{
		{
			name:       "Bearerトークンを抽出できる",
			headerName: authorizationHeader,
			header:     "Bearer firebase-token",
			expected:   "firebase-token",
		},
		{
			name:       "schemeの大文字小文字は区別しない",
			headerName: authorizationHeader,
			header:     "bearer firebase-token",
			expected:   "firebase-token",
		},
		{
			name:       "トークン前後の空白を除去する",
			headerName: authorizationHeader,
			header:     "Bearer   firebase-token   ",
			expected:   "firebase-token",
		},
		{
			name:       "Authorizationヘッダがない場合は空文字を返す",
			headerName: authorizationHeader,
			header:     "",
			expected:   "",
		},
		{
			name:       "Bearer以外のschemeは拒否する",
			headerName: authorizationHeader,
			header:     "Basic firebase-token",
			expected:   "",
		},
		{
			name:       "トークンが空なら空文字を返す",
			headerName: authorizationHeader,
			header:     "Bearer    ",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			header := make(http.Header)
			if tt.header != "" {
				header.Set(tt.headerName, tt.header)
			}

			// When
			got := ExtractBearerToken(header)

			// Then
			assert.Equal(t, tt.expected, got)
		})
	}
}
