package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerFavoriteSongModel はデータベース用のプレイヤーお気に入り楽曲モデルです。
type PlayerFavoriteSongModel struct {
	PlayerID  int       `db:"player_id"`
	SongID    int       `db:"song_id"`
	CreatedAt time.Time `db:"created_at"`
}

// ToEntity は PlayerFavoriteSongModel を entity.PlayerFavoriteSong に変換します。
func (m *PlayerFavoriteSongModel) ToEntity() *entity.PlayerFavoriteSong {
	return &entity.PlayerFavoriteSong{
		PlayerID:  m.PlayerID,
		SongID:    m.SongID,
		CreatedAt: m.CreatedAt,
	}
}

// FromPlayerFavoriteSongEntity は entity.PlayerFavoriteSong を PlayerFavoriteSongModel に変換します。
func FromPlayerFavoriteSongEntity(e *entity.PlayerFavoriteSong) *PlayerFavoriteSongModel {
	return &PlayerFavoriteSongModel{
		PlayerID:  e.PlayerID,
		SongID:    e.SongID,
		CreatedAt: e.CreatedAt,
	}
}
