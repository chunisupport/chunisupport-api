package dto

import (
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/service"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/chartconstant"
)

// PlayerRecordDTO はプレイヤーレコードを外部へ公開するためのDTOです。
type PlayerRecordDTO struct {
	UpdatedAt      time.Time                   `json:"updated_at"`
	Difficulty     string                      `json:"difficulty"`
	ID             string                      `json:"id"`
	Title          string                      `json:"title"`
	Artist         string                      `json:"artist"`
	Const          chartconstant.ChartConstant `json:"const"`
	IsConstUnknown bool                        `json:"is_const_unknown"`
	Score          uint32                      `json:"score"`
	Rating         float64                     `json:"rating"`
	Overpower      float64                     `json:"overpower"`
	Img            string                      `json:"img"`
	ClearLamp      string                      `json:"clear_lamp"`
	ComboLamp      *string                     `json:"combo_lamp"`
	FullChain      *string                     `json:"full_chain"`
	Slot           *string                     `json:"slot"`
}

// ToPlayerRecordDTO は PlayerRecord エンティティをDTOへ変換します。
func ToPlayerRecordDTO(record *entity.PlayerRecord) *PlayerRecordDTO {
	if record == nil {
		return nil
	}

	// レーティング・OVER POWER計算用の値を取得
	score := uint32(record.Score)
	var chartConst chartconstant.ChartConstant
	var isConstUnknown bool
	if record.Chart != nil {
		chartConst = record.Chart.Const
		isConstUnknown = record.Chart.IsConstUnknown
	}

	dto := &PlayerRecordDTO{
		UpdatedAt:      record.UpdatedAt,
		Const:          chartConst,
		IsConstUnknown: isConstUnknown,
		Score:          score,
		Rating:         service.CalcSingleRating(score, float64(chartConst)),
		Overpower:      service.CalcSingleOverpower(score, float64(chartConst), record.ComboLampID),
		ClearLamp:      toMasterName(record.ClearLamp),
		ComboLamp:      toMasterNamePtr(record.ComboLamp),
		FullChain:      toMasterNamePtr(record.FullChain),
		Slot:           toMasterNamePtr(record.Slot),
	}

	if record.ChartDifficulty != nil {
		dto.Difficulty = record.ChartDifficulty.Name
	}

	if record.Song != nil {
		dto.ID = record.Song.DisplayID
		dto.Title = record.Song.Title
		dto.Artist = record.Song.Artist
		if record.Song.Jacket != nil {
			dto.Img = *record.Song.Jacket
		}
	}

	return dto
}

// toMasterName はマスタエンティティからName文字列を取り出します。nilの場合は空文字を返します。
func toMasterName(master any) string {
	switch v := master.(type) {
	case *entity.ClearLampType:
		if v == nil {
			return ""
		}
		return v.Name
	case *entity.ComboLampType:
		if v == nil {
			return ""
		}
		return v.Name
	case *entity.FullChainType:
		if v == nil {
			return ""
		}
		return v.Name
	case *entity.Slot:
		if v == nil {
			return ""
		}
		return v.Name
	default:
		return ""
	}
}

// isNoneValue は「存在しない」を表す便宜上のマスタ値かどうかを判定します。
func isNoneValue(name string) bool {
	return name == "NONE" || name == "none"
}

// toMasterNamePtr はマスタエンティティからName文字列のポインタを取り出します。
// nilの場合、または「NONE」「none」など便宜上の値の場合はnilを返します。
func toMasterNamePtr(master any) *string {
	name := toMasterName(master)
	if name == "" || isNoneValue(name) {
		return nil
	}
	return &name
}

// UserRecordResponseDTO はユーザーレコードAPIレスポンス全体のDTOです。
type UserRecordResponseDTO struct {
	UpdatedAt     time.Time             `json:"updated_at"`
	Best          []*PlayerRecordDTO    `json:"best"`
	BestCandidate []*PlayerRecordDTO    `json:"best_candidate"`
	New           []*PlayerRecordDTO    `json:"new"`
	NewCandidate  []*PlayerRecordDTO    `json:"new_candidate"`
	All           []*PlayerRecordDTO    `json:"all"`
	WorldsEnd     []*WorldsendRecordDTO `json:"worldsend"` // WORLD'S END レコード（全件）
}
