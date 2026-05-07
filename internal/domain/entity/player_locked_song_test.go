package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlayerLockedSong(t *testing.T) {
	tests := []struct {
		name     string
		playerID int
		songID   int
		isUltima bool
		wantErr  string
	}{
		{
			name:     "通常譜面群の未解禁状態を生成できる",
			playerID: 1,
			songID:   10,
			isUltima: false,
		},
		{
			name:     "ULTIMA譜面の未解禁状態を生成できる",
			playerID: 1,
			songID:   10,
			isUltima: true,
		},
		{
			name:     "プレイヤーIDが0の場合は生成できない",
			playerID: 0,
			songID:   10,
			wantErr:  "player_id",
		},
		{
			name:     "楽曲IDが0の場合は生成できない",
			playerID: 1,
			songID:   0,
			wantErr:  "song_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := NewPlayerLockedSong(tt.playerID, tt.songID, tt.isUltima)

			// Then
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.playerID, got.PlayerID)
			assert.Equal(t, tt.songID, got.SongID)
			assert.Equal(t, tt.isUltima, got.IsUltima)
		})
	}
}
