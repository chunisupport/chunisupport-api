package entity

import (
	"fmt"
	"time"
)

// PlayerFavoriteSong はプレイヤーのお気に入り楽曲を表します。
type PlayerFavoriteSong struct {
	PlayerID  int
	SongID    int
	CreatedAt time.Time
}

// NewPlayerFavoriteSong は不変条件を満たす PlayerFavoriteSong を生成します。
func NewPlayerFavoriteSong(playerID int, songID int) (*PlayerFavoriteSong, error) {
	p := &PlayerFavoriteSong{
		PlayerID:  playerID,
		SongID:    songID,
		CreatedAt: time.Now().UTC(),
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

// Validate は PlayerFavoriteSong のバリデーションを行います。
func (p *PlayerFavoriteSong) Validate() error {
	if p.PlayerID <= 0 {
		return fmt.Errorf("player_id: プレイヤーIDは正の整数である必要があります")
	}
	if p.SongID <= 0 {
		return fmt.Errorf("song_id: 楽曲IDは正の整数である必要があります")
	}
	return nil
}
