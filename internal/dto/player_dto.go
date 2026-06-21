package dto

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// HonorDTO は称号情報を外部に公開するためのDTOです。
type HonorDTO struct {
	Slot     int    `json:"slot"`      // 称号スロット: 1=上段, 2=中段, 3=下段
	Name     string `json:"name"`      // 称号名
	TypeName string `json:"type_name"` // 称号タイプ名 (normal, copper, silver, gold, platina, rainbow, etc.)
	ImageURL string `json:"image_url"` // 称号画像URL
}

// PlayerDTO はプレイヤー情報を外部に公開するためのDTOです。
type PlayerDTO struct {
	Name              string      `json:"name"`
	Level             int         `json:"level"`
	Rating            *float64    `json:"rating"`
	CalculatedRating  *float64    `json:"-"`
	BestAverageRating *float64    `json:"-"`
	NewAverageRating  *float64    `json:"-"`
	ClassEmblemID     *int        `json:"class_emblem_id"`
	ClassEmblemBaseID *int        `json:"class_emblem_base_id"`
	LastPlayedAt      *time.Time  `json:"last_played_at"`
	OverpowerValue    *float64    `json:"overpower_value"`
	OverpowerPercent  *float64    `json:"overpower_percent"`
	Honors            []*HonorDTO `json:"honors"` // 称号情報（スロット順）
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// ToPlayerDTO はエンティティからDTOへ変換します。
// 注: Honors フィールドはこの関数では設定されません。サービス層で別途設定してください。
func ToPlayerDTO(player *entity.Player) *PlayerDTO {
	if player == nil {
		return nil
	}

	return &PlayerDTO{
		Name:              player.Name.String(),
		Level:             player.Level,
		Rating:            player.OfficialRating,
		CalculatedRating:  player.CalculatedRating,
		BestAverageRating: player.BestAverageRating,
		NewAverageRating:  player.NewAverageRating,
		ClassEmblemID:     player.ClassEmblemID,
		ClassEmblemBaseID: player.ClassEmblemBaseID,
		LastPlayedAt:      player.LastPlayedAt,
		OverpowerValue:    player.OverpowerValue,
		OverpowerPercent:  player.OverpowerPercent,
		Honors:            []*HonorDTO{}, // 空のスライスで初期化（nullを避ける）
		CreatedAt:         player.CreatedAt,
		UpdatedAt:         player.UpdatedAt,
	}
}
