package config

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
)

func TestLoadSongBatchConfigFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected SongBatchConfig
	}{
		{
			name:     "前後の空白を除去して読み込む",
			value:    " https://wikiwiki.jp/chunithmwiki/ ",
			expected: SongBatchConfig{WikiBaseURL: "https://wikiwiki.jp/chunithmwiki/"},
		},
		{
			name:     "未設定の場合は空文字になる",
			value:    "",
			expected: SongBatchConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(info.SongBatchEnvWikiBaseURL, tt.value)

			got := LoadSongBatchConfigFromEnv()

			assert.Equal(t, tt.expected, got)
		})
	}
}
