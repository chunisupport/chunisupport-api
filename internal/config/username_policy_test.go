package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadUsernamePolicy(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantExact   []string
		wantContain []string
		wantErr     bool
	}{
		{
			name:        "完全一致と部分一致の禁止語を読み込む",
			content:     `{"exact":["admin","system"],"contains":["word"]}`,
			wantExact:   []string{"admin", "system"},
			wantContain: []string{"word"},
		},
		{
			name:    "壊れたJSONを拒否する",
			content: `{"exact":`,
			wantErr: true,
		},
		{
			name:    "未知のフィールドを拒否する",
			content: `{"unknown":["admin"]}`,
			wantErr: true,
		},
		{
			name:    "nullを拒否する",
			content: `null`,
			wantErr: true,
		},
		{
			name:    "JSONの後続データを拒否する",
			content: `{"exact":["admin"]} {}`,
			wantErr: true,
		},
		{
			name:    "大文字を含む禁止語を拒否する",
			content: `{"exact":["Admin"]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "username_forbidden_words.json")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0600))

			got, err := loadUsernamePolicy(path)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantExact, got.Exact)
			assert.Equal(t, tt.wantContain, got.Contains)
		})
	}
}

func TestLoadUsernamePolicy_重複した禁止語を除去する(t *testing.T) {
	path := filepath.Join(t.TempDir(), "username_forbidden_words.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"exact":["admin","admin"],"contains":["system","system"]}`), 0600))

	got, err := loadUsernamePolicy(path)

	require.NoError(t, err)
	assert.Equal(t, []string{"admin"}, got.Exact)
	assert.Equal(t, []string{"system"}, got.Contains)
}

func TestLoadUsernamePolicy_ファイルが存在しなければエラーを返す(t *testing.T) {
	_, err := loadUsernamePolicy(filepath.Join(t.TempDir(), "missing.json"))

	assert.Error(t, err)
}
