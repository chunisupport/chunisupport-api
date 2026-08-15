package maintenancecomment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMaintenanceComment(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
		wantErr  error
	}{
		{
			name:     "通常のコメントを生成できる",
			value:    "データ更新のためメンテナンスを実施しています。",
			expected: "データ更新のためメンテナンスを実施しています。",
		},
		{
			name:     "複数行コメントを保持する",
			value:    "データ更新中です。\nしばらくお待ちください。",
			expected: "データ更新中です。\nしばらくお待ちください。",
		},
		{
			name:     "CRLFとCRをLFへ正規化して前後の空白を除去する",
			value:    " \r\nデータ更新中です。\rしばらくお待ちください。\r\n ",
			expected: "データ更新中です。\nしばらくお待ちください。",
		},
		{
			name:     "Unicodeコードポイントで1000文字を許可する",
			value:    strings.Repeat("保", MaxLength),
			expected: strings.Repeat("保", MaxLength),
		},
		{
			name:    "Unicodeコードポイントで1001文字を拒否する",
			value:   strings.Repeat("保", MaxLength+1),
			wantErr: ErrTooLong,
		},
		{
			name:    "空文字を拒否する",
			value:   "",
			wantErr: ErrRequired,
		},
		{
			name:    "空白と改行だけのコメントを拒否する",
			value:   " \r\n　 ",
			wantErr: ErrRequired,
		},
		{
			name:    "タブを拒否する",
			value:   "更新\t中",
			wantErr: ErrControlCharacter,
		},
		{
			name:    "前後のタブも拒否する",
			value:   "\t更新中\t",
			wantErr: ErrControlCharacter,
		},
		{
			name:    "NULL文字を拒否する",
			value:   "更新\x00中",
			wantErr: ErrControlCharacter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			comment, err := NewMaintenanceComment(tt.value)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, comment.String())
			assert.False(t, comment.IsEmpty())
		})
	}
}

func TestRestoreMaintenanceComment(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
		wantErr  error
	}{
		{
			name:     "無効状態の空コメントを復元できる",
			value:    "",
			expected: "",
		},
		{
			name:     "永続化済みコメントも正規化する",
			value:    " 更新中\r\nです ",
			expected: "更新中\nです",
		},
		{
			name:    "長すぎる永続化値を拒否する",
			value:   strings.Repeat("保", MaxLength+1),
			wantErr: ErrTooLong,
		},
		{
			name:    "不正な制御文字を含む永続化値を拒否する",
			value:   "更新\b中",
			wantErr: ErrControlCharacter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			comment, err := RestoreMaintenanceComment(tt.value)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, comment.String())
			assert.Equal(t, tt.expected == "", comment.IsEmpty())
		})
	}
}
