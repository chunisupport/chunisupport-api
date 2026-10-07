package dto

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// WorldsendRecordDTO は WORLD'S END レコードを外部へ公開するための DTO です。
// WORLD'S END はレーティング計算の対象外であり、スロット（Best/New等）の概念を持ちません。
type WorldsendRecordDTO struct {
	UpdatedAt    *time.Time `json:"updated_at"`
	IsPlayed     bool       `json:"is_played"`
	ID           string     `json:"id"` // 楽曲の表示用ID（Song.DisplayID）
	Title        string     `json:"title"`
	Artist       string     `json:"artist"`
	LevelStar    *int       `json:"level_star"`
	Attribute    *string    `json:"attribute"`
	Notes        *int       `json:"notes"`
	Score        uint32     `json:"score"`
	JusticeCount *int       `json:"justice_count"`
	Img          string     `json:"img"`
	ClearLamp    *string    `json:"clear_lamp"`
	ComboLamp    *string    `json:"combo_lamp"` // マスタ値がNONEの場合はnull
	FullChain    *string    `json:"full_chain"` // マスタ値がNONEの場合はnull
}

// ToWorldsendRecordDTO は PlayerWorldsendRecord エンティティを DTO へ変換します。
func ToWorldsendRecordDTO(record *entity.PlayerWorldsendRecord) *WorldsendRecordDTO {
	if record == nil {
		return nil
	}

	score := uint32(0)
	if scoreVal, err := record.Score.Value(); err == nil {
		score = uint32(scoreVal.(int64)) // #nosec G115 -- ドメインの値オブジェクトがuint32の範囲内であることを保証するため
	}

	dto := &WorldsendRecordDTO{
		Score:        score,
		JusticeCount: calcJusticeCount(score, record.ComboLampID, worldsendRecordNotes(record)),
		ClearLamp:    toMasterNamePtr(record.ClearLamp),
		ComboLamp:    toMasterNamePtr(record.ComboLamp),
		FullChain:    toMasterNamePtr(record.FullChain),
	}
	if !record.UpdatedAt.IsZero() {
		dto.UpdatedAt = &record.UpdatedAt
		dto.IsPlayed = true
	}

	if record.WorldsendChart != nil {
		dto.LevelStar = ToLevelStarIntPtr(record.WorldsendChart.LevelStar)
		dto.Attribute = record.WorldsendChart.Attribute
		dto.Notes = ToNotesIntPtr(record.WorldsendChart.Notes)
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

func worldsendRecordNotes(record *entity.PlayerWorldsendRecord) *int {
	if record.WorldsendChart == nil {
		return nil
	}
	return ToNotesIntPtr(record.WorldsendChart.Notes)
}

// ToNotesIntPtr は Notes の値オブジェクトを *int に変換します。
func ToNotesIntPtr(value *notes.Notes) *int {
	if value == nil {
		return nil
	}

	converted := int(*value)
	return &converted
}

// ToLevelStarIntPtr は LevelStar の値オブジェクトを *int に変換します。
func ToLevelStarIntPtr(value *levelstar.LevelStar) *int {
	if value == nil {
		return nil
	}

	converted := value.Int()
	return &converted
}
