package dto

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// WorldsendRecordDTO は WORLD'S END レコードを外部へ公開するための DTO です。
// WORLD'S END はレーティング計算の対象外であり、スロット（Best/New等）の概念を持ちません。
type WorldsendRecordDTO struct {
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`         // 楽曲の DisplayID
	Title     string    `json:"title"`      // 楽曲タイトル
	Artist    string    `json:"artist"`     // アーティスト名
	WeStar    *int      `json:"we_star"`    // WORLD'S END 星の数（1～5）
	WeKanji   *string   `json:"we_kanji"`   // WORLD'S END カテゴリ漢字（光、蔵、改、狂、etc.）
	Notes     *int      `json:"notes"`      // ノーツ数
	Score     uint32    `json:"score"`      // スコア
	Img       string    `json:"img"`        // ジャケット画像URL
	ClearLamp string    `json:"clear_lamp"` // クリアランプ
	ComboLamp *string   `json:"combo_lamp"` // コンボランプ（マスタ値が「NONE」の場合はnull）
	FullChain *string   `json:"full_chain"` // フルチェイン（マスタ値が「NONE」の場合はnull）
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
		UpdatedAt: record.UpdatedAt,
		Score:     score,
		ClearLamp: toMasterName(record.ClearLamp),
		ComboLamp: toMasterNamePtr(record.ComboLamp),
		FullChain: toMasterNamePtr(record.FullChain),
	}

	// WORLD'S END 譜面情報を設定
	if record.WorldsendChart != nil {
		dto.WeStar = record.WorldsendChart.WeStar
		dto.WeKanji = record.WorldsendChart.WeKanji
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

// ToNotesIntPtr は Notes の値オブジェクトを *int に変換します。
func ToNotesIntPtr(value *notes.Notes) *int {
	if value == nil {
		return nil
	}

	converted := int(*value)
	return &converted
}

// WorldsendSongDTO は WORLD'S END 楽曲情報を外部へ公開するための DTO です。
type WorldsendSongDTO struct {
	ID          string  `json:"id"`           // DisplayID
	Title       string  `json:"title"`        // 楽曲タイトル
	Artist      string  `json:"artist"`       // アーティスト名
	GenreID     *int    `json:"genre_id"`     // ジャンルID
	BPM         *int    `json:"bpm"`          // BPM
	ReleasedAt  *string `json:"released_at"`  // リリース日
	OfficialIdx string  `json:"official_idx"` // 公式インデックス
	Jacket      *string `json:"jacket"`       // ジャケット画像URL
	WeStar      *int    `json:"we_star"`      // WORLD'S END 星の数（1～5）
	WeKanji     *string `json:"we_kanji"`     // WORLD'S END カテゴリ漢字
	Notes       *int    `json:"notes"`        // ノーツ数
	IsDeleted   bool    `json:"is_deleted"`   // 削除フラグ
}
