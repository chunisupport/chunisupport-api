package profilecomment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProfileComment(t *testing.T) {
	tests := []struct {
		name, input, want string
		wantErr           error
	}{
		{name: "前後の空白を除去", input: "　 よろしくお願いします 🙂　", want: "よろしくお願いします 🙂"},
		{name: "日本語100文字", input: strings.Repeat("あ", 100), want: strings.Repeat("あ", 100)},
		{name: "絵文字100文字", input: strings.Repeat("🙂", 100), want: strings.Repeat("🙂", 100)},
		{name: "101文字は拒否", input: strings.Repeat("あ", 101), wantErr: ErrTooLong},
		{name: "空は拒否", input: "", wantErr: ErrEmpty},
		{name: "空白のみは拒否", input: "　 \t\n", wantErr: ErrEmpty},
		{name: "途中の改行は拒否", input: "a\nb", wantErr: ErrInvalidChar},
		{name: "途中のタブは拒否", input: "a\tb", wantErr: ErrInvalidChar},
		{name: "NULは拒否", input: "a\x00b", wantErr: ErrInvalidChar},
		{name: "DELは拒否", input: "a\x7fb", wantErr: ErrInvalidChar},
		{name: "Unicode制御文字は拒否", input: "a\u0085b", wantErr: ErrInvalidChar},
		{name: "行区切りは拒否", input: "a\u2028b", wantErr: ErrInvalidChar},
		{name: "段落区切りは拒否", input: "a\u2029b", wantErr: ErrInvalidChar},
		{name: "不正なUTF8は拒否", input: "a\xff", wantErr: ErrInvalidChar},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 入力文字列
			// When
			got, err := NewProfileComment(tt.input)
			// Then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}
