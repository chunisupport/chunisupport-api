package entity

import (
	"testing"
	"time"

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
			name:     "player_idが負の場合は生成できない",
			playerID: -1,
			songID:   10,
			wantErr:  "player_id",
		},
		{
			name:     "song_idが0の場合は生成できない",
			playerID: 1,
			songID:   0,
			wantErr:  "song_id",
		},
		{
			name:     "song_idが負の場合は生成できない",
			playerID: 1,
			songID:   -1,
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
		})
	}
}

func TestNewPlayerFavoriteSong_SetsCreatedAt(t *testing.T) {
	// When
	before := time.Now()
	got, err := NewPlayerFavoriteSong(1, 10)
	after := time.Now()

	// Then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.CreatedAt.IsZero())
	assert.WithinRange(t, got.CreatedAt, before, after)
}
