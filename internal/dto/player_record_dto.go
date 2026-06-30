package dto

import (
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
)

// PlayerRecordDTO はプレイヤーレコードを外部へ公開するためのDTOです。
type PlayerRecordDTO struct {
	UpdatedAt        *time.Time                  `json:"updated_at"`
	IsPlayed         bool                        `json:"is_played"`
	IsOPTarget       bool                        `json:"is_op_target"`
	Difficulty       string                      `json:"difficulty"`
	ID               string                      `json:"id"`
	Title            string                      `json:"title"`
	Artist           string                      `json:"artist"`
	Const            chartconstant.ChartConstant `json:"const"`
	IsConstUnknown   bool                        `json:"is_const_unknown"`
	Score            uint32                      `json:"score"`
	JusticeCount     *int                        `json:"justice_count"`
	Rating           float64                     `json:"rating"`
	Overpower        float64                     `json:"overpower"`
	OverpowerPercent float64                     `json:"overpower_percent"`
	Img              string                      `json:"img"`
	ClearLamp        *string                     `json:"clear_lamp"`
	ComboLamp        *string                     `json:"combo_lamp"`
	FullChain        *string                     `json:"full_chain"`
	Slot             *string                     `json:"slot"`
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
		Const:            chartConst,
		IsConstUnknown:   isConstUnknown,
		IsOPTarget:       record.IsOPTarget,
		Score:            score,
		JusticeCount:     calcJusticeCount(score, record.ComboLampID, playerRecordNotes(record)),
		Rating:           service.CalcSingleRating(score, chartConst.Float64()),
		Overpower:        service.CalcSingleOverpower(score, chartConst.Float64(), record.ComboLampID),
		OverpowerPercent: service.CalcSingleOverpowerPercent(score, chartConst.Float64(), record.ComboLampID),
		ClearLamp:        toMasterNamePtr(record.ClearLamp),
		ComboLamp:        toMasterNamePtr(record.ComboLamp),
		FullChain:        toMasterNamePtr(record.FullChain),
		Slot:             toMasterNamePtr(record.Slot),
	}
	if !record.UpdatedAt.IsZero() {
		dto.UpdatedAt = &record.UpdatedAt
		dto.IsPlayed = true
	}

	if record.ChartDifficulty != nil {
		dto.Difficulty = strings.ToUpper(record.ChartDifficulty.Name)
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

func playerRecordNotes(record *entity.PlayerRecord) *int {
	if record.Chart == nil {
		return nil
	}
	return ToNotesIntPtr(record.Chart.Notes)
}

// toMasterName はマスタエンティティからName文字列を取り出します。nilの場合は空文字を返します。
func toMasterName(masterValue any) string {
	switch v := masterValue.(type) {
	case *master.ClearLampType:
		if v == nil {
			return ""
		}
		return v.Name
	case *master.ComboLampType:
		if v == nil {
			return ""
		}
		return v.Name
	case *master.FullChainType:
		if v == nil {
			return ""
		}
		return v.Name
	case *master.Slot:
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
func toMasterNamePtr(masterValue any) *string {
	name := toMasterName(masterValue)
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
	All           []*PlayerRecordDTO    `json:"standard"`
	WorldsEnd     []*WorldsendRecordDTO `json:"worldsend"` // WORLD'S END レコード（全件）
}
