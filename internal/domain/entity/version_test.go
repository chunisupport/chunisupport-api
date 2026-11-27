package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testVersionDisplayID displayid.DisplayID = "0123456789abcdef"

func TestNewVersion(t *testing.T) {
	tests := []struct {
		name           string
		inputName      string
		inputShortName string
		date           time.Time
		wantName       string
		wantShortName  string
		wantErr        bool
	}{
		{name: "正常な名前を作成できる", inputName: " CHUNITHM VERSE ", inputShortName: " VRS ", date: time.Date(2025, 12, 11, 12, 34, 56, 0, time.FixedZone("JST", 9*60*60)), wantName: "CHUNITHM VERSE", wantShortName: "VRS"},
		{name: "10文字の超ショート名を作成できる", inputName: "CHUNITHM VERSE", inputShortName: strings.Repeat("×", 10), date: time.Date(2025, 12, 11, 0, 0, 0, 0, time.UTC), wantName: "CHUNITHM VERSE", wantShortName: strings.Repeat("×", 10)},
		{name: "接頭辞がない名前は拒否する", inputName: "VERSE", inputShortName: "VRS", date: time.Now(), wantErr: true},
		{name: "接頭辞だけの名前は拒否する", inputName: "CHUNITHM ", inputShortName: "VRS", date: time.Now(), wantErr: true},
		{name: "51文字の名前は拒否する", inputName: "CHUNITHM " + strings.Repeat("あ", 42), inputShortName: "VRS", date: time.Now(), wantErr: true},
		{name: "50文字の名前を作成できる", inputName: "CHUNITHM " + strings.Repeat("あ", 41), inputShortName: "VRS", date: time.Date(2025, 12, 11, 0, 0, 0, 0, time.UTC), wantName: "CHUNITHM " + strings.Repeat("あ", 41), wantShortName: "VRS"},
		{name: "JSTの午前8時台でも稼働日当日の0時UTCとして保存する", inputName: "CHUNITHM VERSE", inputShortName: "VRS", date: time.Date(2025, 12, 11, 8, 59, 0, 0, time.FixedZone("JST", 9*60*60)), wantName: "CHUNITHM VERSE", wantShortName: "VRS"},
		{name: "空白だけの超ショート名は拒否する", inputName: "CHUNITHM VERSE", inputShortName: "  ", date: time.Now(), wantErr: true},
		{name: "11文字の超ショート名は拒否する", inputName: "CHUNITHM VERSE", inputShortName: strings.Repeat("×", 11), date: time.Now(), wantErr: true},
		{name: "ゼロ日付は拒否する", inputName: "CHUNITHM VERSE", inputShortName: "VRS", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := NewVersion(testVersionDisplayID, tt.inputName, tt.inputShortName, tt.date)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidVersion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testVersionDisplayID, version.DisplayID)
			assert.Equal(t, tt.wantName, version.Name)
			assert.Equal(t, tt.wantShortName, version.ShortName)
			assert.Equal(t, time.Date(2025, 12, 11, 0, 0, 0, 0, time.UTC), version.ReleasedAt)
		})
	}
}

func TestNewVersion_不正な公開IDは拒否する(t *testing.T) {
	// When
	version, err := NewVersion("1", "CHUNITHM VERSE", "VRS", time.Now())

	// Then
	assert.ErrorIs(t, err, ErrInvalidVersion)
	assert.Nil(t, version)
}

func TestVersion_Rename(t *testing.T) {
	version, err := NewVersion(testVersionDisplayID, "CHUNITHM VERSE", "VRS", time.Now())
	require.NoError(t, err)

	err = version.Rename(" CHUNITHM VERSE II ", " VRS2 ")

	require.NoError(t, err)
	assert.Equal(t, "CHUNITHM VERSE II", version.Name)
	assert.Equal(t, "VRS2", version.ShortName)
}

func TestVersion_Rename_不正な超ショート名は変更しない(t *testing.T) {
	// Given
	version, err := NewVersion(testVersionDisplayID, "CHUNITHM VERSE", "VRS", time.Now())
	require.NoError(t, err)

	// When
	err = version.Rename("CHUNITHM VERSE II", strings.Repeat("×", 11))

	// Then
	assert.ErrorIs(t, err, ErrInvalidVersion)
	assert.Equal(t, "CHUNITHM VERSE", version.Name)
	assert.Equal(t, "VRS", version.ShortName)
}
