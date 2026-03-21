package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeString(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)

	tests := []struct {
		name             string
		input            string
		expected         time.Time
		location         *time.Location
		wantSameLocation bool
		wantNil          bool
		wantErr          bool
	}{
		{
			name:     "RFC3339Nanoを解釈できる",
			input:    "2026-03-20T12:34:56.123456789Z",
			expected: time.Date(2026, 3, 20, 12, 34, 56, 123456789, time.UTC),
			location: jst,
		},
		{
			name:     "スペース区切りのオフセット付き日時を解釈できる",
			input:    "2026-03-20 21:34:56+09:00",
			expected: time.Date(2026, 3, 20, 21, 34, 56, 0, jst),
			location: time.UTC,
		},
		{
			name:             "小数秒付き日時を解釈できる",
			input:            "2026-03-20 12:34:56.123456",
			expected:         time.Date(2026, 3, 20, 12, 34, 56, 123456000, jst),
			location:         jst,
			wantSameLocation: true,
		},
		{
			name:             "タイムゾーンなし日時はローカル時刻として解釈する",
			input:            "2026-03-20 12:34:56",
			expected:         time.Date(2026, 3, 20, 12, 34, 56, 0, jst),
			location:         jst,
			wantSameLocation: true,
		},
		{
			name:     "空文字はnilを返す",
			input:    "   ",
			location: jst,
			wantNil:  true,
		},
		{
			name:     "不正な文字列はエラー",
			input:    "not-a-time",
			location: jst,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimeStringInLocation(tt.input, tt.location)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.True(t, tt.expected.Equal(*got))
			if tt.wantSameLocation {
				assert.Equal(t, tt.expected.Location(), got.Location())
			}
		})
	}
}
