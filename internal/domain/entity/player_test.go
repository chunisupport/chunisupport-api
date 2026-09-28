package entity

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/masterfingerprint"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername/playernametest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlayer(t *testing.T) {
	name := playernametest.New(t, "プレイヤー")

	player := NewPlayer(42, name)

	assert.Equal(t, DefaultPlayerLevel, player.Level)
	assert.Equal(t, DefaultPossessionID, player.PossessionID)
	assert.False(t, player.CreatedAt.IsZero())
	assert.False(t, player.UpdatedAt.IsZero())
	assert.Equal(t, player.CreatedAt, player.UpdatedAt)
}

func TestPlayer_MarkRecalculated_フィンガープリントを記録する(t *testing.T) {
	// Given
	player := NewPlayer(42, playernametest.New(t, "プレイヤー"))
	fingerprint := masterfingerprint.Compute([]byte("master"))

	// When
	player.MarkRecalculated(fingerprint)

	// Then
	require.NotNil(t, player.RecalculatedMasterFingerprint)
	assert.Equal(t, fingerprint, *player.RecalculatedMasterFingerprint)
}

func TestPlayer_計算入力や計算値の変更で再計算済みの記録を消す(t *testing.T) {
	name := playernametest.New(t, "プレイヤー")
	value := 12345.0
	tests := []struct {
		name   string
		change func(p *Player)
	}{
		{
			name:   "ChangeProfile",
			change: func(p *Player) { p.ChangeProfile(name, 10, nil, nil, nil) },
		},
		{
			name:   "ChangeProfileWithPossession",
			change: func(p *Player) { p.ChangeProfileWithPossession(name, 10, nil, nil, DefaultPossessionID, nil) },
		},
		{
			name:   "ChangeCalculatedRatings",
			change: func(p *Player) { p.ChangeCalculatedRatings(17, 17.1, 16.9) },
		},
		{
			name:   "ChangeOverpower",
			change: func(p *Player) { p.ChangeOverpower(&value, nil) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			player := NewPlayer(42, name)
			player.MarkRecalculated(masterfingerprint.Compute([]byte("master")))

			// When
			tt.change(player)

			// Then
			assert.Nil(t, player.RecalculatedMasterFingerprint)
		})
	}
}
