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
	ID           string     `json:"id"`            // 楽曲の DisplayID
	Title        string     `json:"title"`         // 楽曲タイトル
	Artist       string     `json:"artist"`        // アーティスト名
	LevelStar    *int       `json:"level_star"`    // WORLD'S END レベル（1～5）
	Attribute    *string    `json:"attribute"`     // WORLD'S END 属性（光、蔵、改、狂、etc.）
	Notes        *int       `json:"notes"`         // ノーツ数
	Score        uint32     `json:"score"`         // スコア
	JusticeCount *int       `json:"justice_count"` // JUSTICE数
	Img          string     `json:"img"`           // ジャケット画像URL
	ClearLamp    *string    `json:"clear_lamp"`    // クリアランプ
	ComboLamp    *string    `json:"combo_lamp"`    // コンボランプ（マスタ値が「NONE」の場合はnull）
	FullChain    *string    `json:"full_chain"`    // フルチェイン（マスタ値が「NONE」の場合はnull）
}

// ToWorldsendRecordDTO は PlayerWorldsendRecord エンティティを DTO へ変換します。
func ToWorldsendRecordDTO(record *entity.PlayerWorldsendRecord) *WorldsendRecordDTO {
	if record == nil {
		return nil
	}

	// スコア値を取得
	score := uint32(0)
	if scoreVal, err := record.Score.Value(); err == nil {
		score = uint32(scoreVal.(int64)) // #nosec G115 -- Score value is guaranteed to be within uint32 range by domain VO
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

	// WORLD'S END 譜面情報を設定
	if record.WorldsendChart != nil {
		dto.LevelStar = ToLevelStarIntPtr(record.WorldsendChart.LevelStar)
		dto.Attribute = record.WorldsendChart.Attribute
		dto.Notes = ToNotesIntPtr(record.WorldsendChart.Notes)
	}

	// 楽曲情報を設定
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
