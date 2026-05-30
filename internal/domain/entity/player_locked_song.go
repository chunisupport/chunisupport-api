package entity

import "fmt"

// PlayerLockedSong はプレイヤーごとの未解禁楽曲状態を表します。
type PlayerLockedSong struct {
	PlayerID int
	SongID   int
	IsUltima bool
}

// NewPlayerLockedSong は不変条件を満たす PlayerLockedSong を生成します。
func NewPlayerLockedSong(playerID int, songID int, isUltima bool) (*PlayerLockedSong, error) {
	p := &PlayerLockedSong{
		PlayerID: playerID,
		SongID:   songID,
		IsUltima: isUltima,
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

// Validate は PlayerLockedSong のバリデーションを行います。
func (p *PlayerLockedSong) Validate() error {
	if p.PlayerID <= 0 {
		return fmt.Errorf("player_id: プレイヤーIDは正の整数である必要があります")
	}
	if p.SongID <= 0 {
		return fmt.Errorf("song_id: 楽曲IDは正の整数である必要があります")
	}
	return nil
}
