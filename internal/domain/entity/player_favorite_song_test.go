package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlayerFavoriteSong(t *testing.T) {
	tests := []struct {
		name     string
		playerID int
		songID   int
		wantErr  string
	}{
		{
			name:     "正常なIDで生成できる",
			playerID: 1,
			songID:   10,
		},
		{
			name:     "player_idが0の場合は生成できない",
			playerID: 0,
			songID:   10,
			wantErr:  "player_id",
		},
		{
			name:     "song_idが0の場合は生成できない",
			playerID: 1,
			songID:   0,
			wantErr:  "song_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := NewPlayerFavoriteSong(tt.playerID, tt.songID)

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
			assert.False(t, got.CreatedAt.IsZero())
		})
	}
}
