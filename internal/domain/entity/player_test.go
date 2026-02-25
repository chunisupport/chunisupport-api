package entity

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
)

func TestNewPlayer(t *testing.T) {
	name := playername.MustNewPlayerName("プレイヤー")

	player := NewPlayer(42, name)

	if player.UserID != 42 {
		t.Fatalf("UserID = %d, want %d", player.UserID, 42)
	}
	if player.Name.String() != name.String() {
		t.Fatalf("Name = %s, want %s", player.Name.String(), name.String())
	}
	if player.Level != DefaultPlayerLevel {
		t.Fatalf("Level = %d, want %d", player.Level, DefaultPlayerLevel)
	}
	if player.CreatedAt.IsZero() {
		t.Fatal("CreatedAt must not be zero")
	}
	if player.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt must not be zero")
	}
	if !player.CreatedAt.Equal(player.UpdatedAt) {
		t.Fatalf("CreatedAt (%v) and UpdatedAt (%v) must be equal", player.CreatedAt, player.UpdatedAt)
	}
}
