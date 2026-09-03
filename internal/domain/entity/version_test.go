package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVersion(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		date      time.Time
		wantName  string
		wantErr   bool
	}{
		{name: "正常な名前を作成できる", inputName: " CHUNITHM VERSE ", date: time.Date(2025, 12, 11, 12, 34, 56, 0, time.FixedZone("JST", 9*60*60)), wantName: "CHUNITHM VERSE"},
		{name: "接頭辞がない名前は拒否する", inputName: "VERSE", date: time.Now(), wantErr: true},
		{name: "接頭辞だけの名前は拒否する", inputName: "CHUNITHM ", date: time.Now(), wantErr: true},
		{name: "51文字の名前は拒否する", inputName: "CHUNITHM " + strings.Repeat("あ", 42), date: time.Now(), wantErr: true},
		{name: "ゼロ日付は拒否する", inputName: "CHUNITHM VERSE", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := NewVersion(tt.inputName, tt.date)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidVersion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, version.Name)
			assert.Equal(t, "2025-12-11", version.ReleasedAt.Format(time.DateOnly))
			assert.Equal(t, time.UTC, version.ReleasedAt.Location())
		})
	}
}

func TestVersion_Rename(t *testing.T) {
	version, err := NewVersion("CHUNITHM VERSE", time.Now())
	require.NoError(t, err)

	err = version.Rename(" CHUNITHM VERSE II ")

	require.NoError(t, err)
	assert.Equal(t, "CHUNITHM VERSE II", version.Name)
}
