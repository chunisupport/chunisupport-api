package entity

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/stretchr/testify/assert"
)

func TestNewPlayer(t *testing.T) {
	name := playername.MustNewPlayerName("プレイヤー")

	player := NewPlayer(42, name)

	assert.Equal(t, 42, player.UserID)
	assert.Equal(t, name.String(), player.Name.String())
	assert.Equal(t, DefaultPlayerLevel, player.Level)
	assert.False(t, player.CreatedAt.IsZero())
	assert.False(t, player.UpdatedAt.IsZero())
	assert.Equal(t, player.CreatedAt, player.UpdatedAt)
}
