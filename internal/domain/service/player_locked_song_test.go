package service

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestPlayerLockedSongSet_IsLocked(t *testing.T) {
	tests := []struct {
		name        string
		lockedSongs []*entity.PlayerLockedSong
		songID      int
		isUltima    bool
		want        bool
	}{
		{
			name:   "未解禁設定がない譜面は解禁済み",
			songID: 1,
		},
		{
			name:        "通常譜面群の未解禁は通常譜面を未解禁にする",
			lockedSongs: []*entity.PlayerLockedSong{{SongID: 1, IsUltima: false}},
			songID:      1,
			want:        true,
		},
		{
			name:        "通常譜面群の未解禁はULTIMA譜面も未解禁にする",
			lockedSongs: []*entity.PlayerLockedSong{{SongID: 1, IsUltima: false}},
			songID:      1,
			isUltima:    true,
			want:        true,
		},
		{
			name:        "ULTIMA未解禁はULTIMA譜面だけを未解禁にする",
			lockedSongs: []*entity.PlayerLockedSong{{SongID: 1, IsUltima: true}},
			songID:      1,
			isUltima:    true,
			want:        true,
		},
		{
			name:        "ULTIMA未解禁だけでは通常譜面を未解禁にしない",
			lockedSongs: []*entity.PlayerLockedSong{{SongID: 1, IsUltima: true}},
			songID:      1,
		},
		{
			name: "通常譜面群とULTIMAの両方が未解禁でも通常譜面を未解禁にする",
			lockedSongs: []*entity.PlayerLockedSong{
				{SongID: 1, IsUltima: false},
				{SongID: 1, IsUltima: true},
			},
			songID: 1,
			want:   true,
		},
		{
			name:        "別楽曲の未解禁設定は影響しない",
			lockedSongs: []*entity.PlayerLockedSong{{SongID: 2, IsUltima: false}},
			songID:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			set := NewPlayerLockedSongSet(tt.lockedSongs)

			// When
			got := set.IsLocked(tt.songID, tt.isUltima)

			// Then
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewPlayerLockedSongSet_nil要素を無視する(t *testing.T) {
	// Given
	lockedSongs := []*entity.PlayerLockedSong{nil, {SongID: 1, IsUltima: true}}

	// When
	set := NewPlayerLockedSongSet(lockedSongs)

	// Then
	assert.False(t, set.IsEmpty())
	assert.True(t, set.IsLocked(1, true))
}

func TestPlayerLockedSongSet_IsEmpty(t *testing.T) {
	tests := []struct {
		name        string
		lockedSongs []*entity.PlayerLockedSong
		want        bool
	}{
		{name: "未解禁設定がない場合は空", want: true},
		{name: "nil要素だけの場合は空", lockedSongs: []*entity.PlayerLockedSong{nil}, want: true},
		{name: "未解禁設定がある場合は空ではない", lockedSongs: []*entity.PlayerLockedSong{{SongID: 1, IsUltima: false}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			set := NewPlayerLockedSongSet(tt.lockedSongs)

			// When
			got := set.IsEmpty()

			// Then
			assert.Equal(t, tt.want, got)
		})
	}
}
