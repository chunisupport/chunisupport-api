package teamcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTeamCode(t *testing.T) {
	tests := []struct {
		name string
		// Given
		input string
		// Then
		wantErr bool
	}{
		{name: "大文字英数字6桁は有効", input: "AB12CD", wantErr: false},
		{name: "数字のみ6桁は有効", input: "012345", wantErr: false},
		{name: "英字のみ6桁は有効", input: "ABCDEF", wantErr: false},
		{name: "5桁は無効", input: "AB12C", wantErr: true},
		{name: "7桁は無効", input: "AB12CDE", wantErr: true},
		{name: "小文字を含む場合は無効", input: "ab12cd", wantErr: true},
		{name: "記号を含む場合は無効", input: "AB-2CD", wantErr: true},
		{name: "前後の空白を含む場合は無効", input: " AB12CD", wantErr: true},
		{name: "全角英数字は無効", input: "ＡＢ１２ＣＤ", wantErr: true},
		{name: "空文字は無効", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			code, err := NewTeamCode(tt.input)

			// Then
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidTeamCode)
				assert.Empty(t, code.String())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.input, code.String())
		})
	}
}
