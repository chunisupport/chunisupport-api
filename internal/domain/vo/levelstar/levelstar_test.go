package levelstar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLevelStar(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{name: "1は有効", value: 1, wantErr: false},
		{name: "5は有効", value: 5, wantErr: false},
		{name: "0は無効", value: 0, wantErr: true},
		{name: "6は無効", value: 6, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLevelStar(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.value, got.Int())
		})
	}
}

func TestLevelStar_Scan(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		want      int
		wantError bool
	}{
		{name: "int64を読み取れる", value: int64(3), want: 3},
		{name: "文字列を読み取れる", value: "4", want: 4},
		{name: "範囲外はエラー", value: int64(9), wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got LevelStar
			err := got.Scan(tt.value)
			if tt.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Int())
		})
	}
}
