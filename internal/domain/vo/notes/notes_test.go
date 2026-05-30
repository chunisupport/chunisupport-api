package notes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotes_Scan(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		initial   Notes
		want      Notes
		wantError bool
	}{
		{name: "int64を読み取れる", value: int64(1200), want: Notes(1200)},
		{name: "文字列を読み取れる", value: "1500", want: Notes(1500)},
		{name: "[]byteを読み取れる", value: []byte("1800"), want: Notes(1800)},
		{name: "nilは0に正規化される", value: nil, initial: Notes(200), want: Notes(0)},
		{name: "負の値はエラーで既存値を維持", value: int64(-1), initial: Notes(200), want: Notes(200), wantError: true},
		{name: "不正な文字列はエラーで既存値を維持", value: "abc", initial: Notes(200), want: Notes(200), wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.initial

			err := got.Scan(tt.value)

			if tt.wantError {
				require.Error(t, err)
				assert.Equal(t, tt.want, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotes_Scan_NilReceiver(t *testing.T) {
	var got *Notes

	err := got.Scan(int64(10))

	require.Error(t, err)
}
